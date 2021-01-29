package statsagent

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"strings"
	"sync"
	"time"
)

//Can use CGO here
//For now using the same name as the C structs
type FlowKey interface {
	GetSrcIp() string
	GetDstIp() string
	GetIpProto() string
	GetSPort() string
	GetDPort() string
}

type FlowStats struct {
	Out_bytes   uint64
	Out_packets uint64
	In_bytes    uint64
	In_packets  uint64
}

func (fs *FlowStats) swap() {
	fs.Out_packets, fs.In_packets = fs.In_packets, fs.Out_packets
	fs.Out_bytes, fs.In_bytes = fs.In_bytes, fs.Out_bytes
}

func addFlowStats(baseStats *FlowStats, incStats *FlowStats) {
	baseStats.Out_bytes += incStats.Out_bytes
	baseStats.Out_packets += incStats.Out_packets
	baseStats.In_bytes += incStats.In_bytes
	baseStats.In_packets += incStats.In_packets
}

func diffFlowStats(oldStats *FlowStats, newStats *FlowStats) *FlowStats {
	if (newStats.Out_bytes < oldStats.Out_bytes) ||
		(newStats.Out_packets < oldStats.Out_packets) ||
		(newStats.In_bytes < oldStats.In_bytes) ||
		(newStats.In_packets < oldStats.In_packets) {
		return &FlowStats{}
	}

	return &FlowStats{
		Out_bytes:   newStats.Out_bytes - oldStats.Out_bytes,
		Out_packets: newStats.Out_packets - oldStats.Out_packets,
		In_bytes:    newStats.In_bytes - oldStats.In_bytes,
		In_packets:  newStats.In_packets - oldStats.In_packets,
	}
}

type PodStatsKey struct {
	Endpoints [2]string
}

func (psk *PodStatsKey) clear(ep int) {
	psk.Endpoints[ep] = ""
}

func (psk *PodStatsKey) swap() {
	psk.Endpoints[0], psk.Endpoints[1] = psk.Endpoints[1], psk.Endpoints[0]
}

type PromMetricsKey struct {
	podNamespace [2]string
	podName      [2]string
	svcNamespace [2]string
	svcScope     [2]string
	svcName      [2]string
	metricName   string
}

const (
	_            = iota
	FROM_POD_KEY = 1 << (iota - 1)
	TO_POD_KEY
	FROM_SVC_KEY
	TO_SVC_KEY
)

func (key *PodStatsKey) toPromMetricsKey(agent *StatsAgent) *PromMetricsKey {
	var promMetricsKey PromMetricsKey
	var keyType int
	svcCount := 0
	podCount := 0
	for i := 0; i < 2; i++ {
		splitStrings := strings.SplitN(key.Endpoints[i], "/", 3)
		switch {
		case len(splitStrings) == 3:
			promMetricsKey.svcNamespace[svcCount] = splitStrings[0]
			promMetricsKey.svcName[svcCount] = splitStrings[1]
			promMetricsKey.svcScope[svcCount] = splitStrings[2]
			svcCount++
			if i == 0 {
				keyType |= FROM_SVC_KEY
			} else {
				keyType |= TO_SVC_KEY
			}
		case len(splitStrings) == 2:
			promMetricsKey.podNamespace[podCount] = splitStrings[0]
			promMetricsKey.podName[podCount] = splitStrings[1]
			podCount++
			if i == 0 {
				keyType |= FROM_POD_KEY
			} else {
				keyType |= TO_POD_KEY
			}
		default:
			// It is not a local pod or an internal IP.
			// Fix when adding svc to ext stats
		}
	}
	switch keyType {
	case FROM_POD_KEY | TO_SVC_KEY:
		promMetricsKey.metricName = "pod_svc_stats"
	case FROM_SVC_KEY | TO_POD_KEY:
		promMetricsKey.metricName = "svc_pod_stats"
	case FROM_SVC_KEY | TO_SVC_KEY:
		promMetricsKey.metricName = "no_explicit_stats"
	case FROM_POD_KEY | TO_POD_KEY:
		promMetricsKey.metricName = "no_explicit_stats"
	case TO_POD_KEY, FROM_POD_KEY:
		promMetricsKey.metricName = "pod_stats"
	case FROM_SVC_KEY, TO_SVC_KEY:
		promMetricsKey.metricName = "svc_stats"
	}
	return &promMetricsKey
}

func getPodStatsKey(agent *StatsAgent, keyOut FlowKey) (PodStatsKey, int) {
	var podStatsKey PodStatsKey
	var keyType int
	srcName, sok := agent.podIpToName[keyOut.GetSrcIp()]
	dstName, dok := agent.podIpToName[keyOut.GetDstIp()]
	if sok {
		podStatsKey.Endpoints[0] = srcName
		keyType |= FROM_POD_KEY
	}
	if dok {
		podStatsKey.Endpoints[1] = dstName
		keyType |= TO_POD_KEY
	}
	srcName, sok = agent.svcIpToName[keyOut.GetSrcIp()]
	dstName, dok = agent.svcIpToName[keyOut.GetDstIp()]
	if sok {
		srcName += "/" + agent.svcInfo[srcName].SvcType
		podStatsKey.Endpoints[0] = srcName
		keyType |= FROM_SVC_KEY
	}
	if dok {
		dstName += "/" + agent.svcInfo[dstName].SvcType
		podStatsKey.Endpoints[1] = dstName
		keyType |= TO_SVC_KEY
	}
	if podStatsKey.Endpoints[0] == "" {
		podStatsKey.Endpoints[0] = keyOut.GetSrcIp()
	}
	if podStatsKey.Endpoints[1] == "" {
		podStatsKey.Endpoints[1] = keyOut.GetDstIp()
	}

	return podStatsKey, keyType
}

type FlowStatsEntry struct {
	Stats         FlowStats
	Aging_counter uint8
	TimeStamp     time.Time
}

func (fs *FlowStatsEntry) add(stats *FlowStats, t *time.Time) {
	addFlowStats(&fs.Stats, stats)
	fs.Aging_counter = 0
	fs.TimeStamp = *t
}

func (fs *FlowStatsEntry) swap() {
	fs.Stats.swap()
}

type MetricsEntry interface {
	GetStatsInterval() int
	Init()
	Run(stopCh <-chan struct{})
	UpdateStats()
}

func runMetric(m MetricsEntry, stopCh <-chan struct{}) {
	go func() {
		ticker := time.NewTicker(time.Duration(m.GetStatsInterval()) * time.Second)
		for {
			select {
			case <-stopCh:
				ticker.Stop()
				break
			case <-ticker.C:
				m.UpdateStats()
			}
		}
	}()
}

func parseFlow(keyIn FlowKey, valueIn *FlowStats) string {
	fwd := fmt.Sprintf("%s(:%s)--[%s]-->%s(:%s)::::%d bytes, %d packets\n", keyIn.GetSrcIp(),
		keyIn.GetSPort(), keyIn.GetIpProto(), keyIn.GetDstIp(), keyIn.GetDPort(), valueIn.Out_bytes, valueIn.Out_packets)
	fwd += fmt.Sprintf("%s(:%s)--[%s]-->%s(:%s)::::%d bytes, %d packets", keyIn.GetDstIp(),
		keyIn.GetDPort(), keyIn.GetIpProto(), keyIn.GetSrcIp(), keyIn.GetSPort(), valueIn.In_bytes, valueIn.In_packets)
	return fwd
}

func (agent *StatsAgent) RunMetrics(stopCh <-chan struct{}) {
	for _, m := range agent.metrics {
		m.Run(stopCh)
	}
}

func (agent *StatsAgent) registerMetric(name string, entry MetricsEntry) {
	agent.metrics[name] = entry
	entry.Init()
	agent.log.Debug("Registered metric ", name)
}

func (agent *StatsAgent) registerMetrics() {
	agent.registerMetric("v4PodStats", NewInetV4FlowMetricsEntry(agent))
	//agent.registerMetric("v6PodStats", agent.getNewV6PodMetricEntry())
}

func (agent *StatsAgent) registerPrometheusSubsystem(entry PromSubsystemEntry) {
	agent.promSubsystems[entry.SubsystemName()] = entry
	entry.RegisterPrometheus(agent)
}

//EDIT this method to add more prometheus metrics
func (agent *StatsAgent) registerPrometheusMetrics() {
	var entry PromSubsystemEntry = NewPodSvcPromSubsystemEntry()
	agent.registerPrometheusSubsystem(entry)
	entry = NewSvcPromSubsystemEntry()
	agent.registerPrometheusSubsystem(entry)
	entry = NewPodPromSubsystemEntry()
	agent.registerPrometheusSubsystem(entry)
}

//Prometheus wrappers
type PromGauge struct {
	Name       string
	GaugeMutex sync.Mutex
	Cache      *prometheus.GaugeVec
}

type PromSubsystem struct {
	Subsystem string
	Gauges    map[string]*PromGauge
}

type PromSubsystemEntry interface {
	SubsystemName() string
	RegisterPrometheus(agent *StatsAgent)
	GetGaugeVec(string) *prometheus.GaugeVec
}
