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

type PromMetricsKey struct {
	podNamespace [2]string
	podName      [2]string
	svcNamespace [2]string
	svcScope     [2]string
	svcName      [2]string
	metricName   string
}

func (key *PodStatsKey) toPromMetricsKey(agent *StatsAgent) *PromMetricsKey {
	var promMetricsKey PromMetricsKey
	svcCount := 0
	podCount := 0
	reversedFlow := false
	for i := 0; i < 2; i++ {
		splitStrings := strings.SplitN(key.Endpoints[i], "/", 3)
		if len(splitStrings) == 3 {
			promMetricsKey.svcNamespace[svcCount] = splitStrings[0]
			promMetricsKey.svcName[svcCount] = splitStrings[1]
			promMetricsKey.svcScope[svcCount] = splitStrings[2]
			svcCount++
			if podCount == 0 {
				reversedFlow = true
			}
		} else {
			promMetricsKey.podNamespace[podCount] = splitStrings[0]
			promMetricsKey.podName[podCount] = splitStrings[1]
			podCount++
		}
	}
	switch {
	case podCount == 1 && svcCount == 1:
		if !reversedFlow {
			promMetricsKey.metricName = "pod_svc_stats"
		} else {
			promMetricsKey.metricName = "svc_pod_stats"
		}
	}
	return &promMetricsKey
}

func getPodStatsKey(agent *StatsAgent, keyOut FlowKey) (PodStatsKey, bool) {
	var podStatsKey PodStatsKey
	var foundEndpoints bool = false
	srcName, sok := agent.podIpToName[keyOut.GetSrcIp()]
	dstName, dok := agent.podIpToName[keyOut.GetDstIp()]
	if sok {
		podStatsKey.Endpoints[0] = srcName
	}
	if dok {
		podStatsKey.Endpoints[1] = dstName
	}
	srcName, sok = agent.svcIpToName[keyOut.GetSrcIp()]
	dstName, dok = agent.svcIpToName[keyOut.GetDstIp()]
	if sok {
		srcName += "/" + agent.svcInfo[srcName].SvcType
		podStatsKey.Endpoints[0] = srcName
	}
	if dok {
		dstName += "/" + agent.svcInfo[dstName].SvcType
		podStatsKey.Endpoints[1] = dstName
	}
	if podStatsKey.Endpoints[0] != "" && podStatsKey.Endpoints[1] != "" {
		foundEndpoints = true
	}
	return podStatsKey, foundEndpoints
}

type FlowStatsEntry struct {
	Stats         FlowStats
	Aging_counter uint8
	TimeStamp     time.Time
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

//PodSvcStats Prometheus Entries
var PodSvcPromMetrics = [...]string{
	"pod_to_svc_bytes",
	"pod_to_svc_packets",
	"svc_to_pod_bytes",
	"svc_to_pod_packets",
}

var PodSvcPromHelp = [...]string{
	"pod to service bytes",
	"pod to service packets",
	"service to pod bytes",
	"service to pod packets",
}

type PodSvcPromSubsystemEntry struct {
	*PromSubsystem
}

func (agent *StatsAgent) SetPodSvcGauge(
	key *PromMetricsKey,
	stats *FlowStats) {
	var value [4]uint64
	value[0] = stats.Out_bytes
	value[1] = stats.Out_packets
	value[2] = stats.In_bytes
	value[3] = stats.In_packets
	if key.metricName == "svc_pod_stats" {
		value[0], value[2] = value[2], value[0]
		value[1], value[3] = value[3], value[1]
	} else if key.metricName != "pod_svc_stats" {
		return
	}
	for i := 0; i < 4; i++ {
		agent.promSubsystems["pod_svc_stats"].GetGaugeVec(PodSvcPromMetrics[i]).With(prometheus.Labels{
			"pod_namespace": key.podNamespace[0],
			"pod_name":      key.podName[0],
			"svc_namespace": key.svcNamespace[0],
			"svc_scope":     key.svcScope[0],
			"svc_name":      key.svcName[0]}).Set(float64(value[i]))
	}
}

func (entry *PodSvcPromSubsystemEntry) SubsystemName() string {
	return entry.Subsystem
}

func (entry *PodSvcPromSubsystemEntry) RegisterPrometheus(agent *StatsAgent) {
	for i, metricName := range PodSvcPromMetrics {
		gauge :=
			prometheus.NewGaugeVec(prometheus.GaugeOpts{
				Namespace: "statsagent",
				Subsystem: "pod_svc_stats",
				Name:      metricName,
				Help:      PodSvcPromHelp[i],
			}, []string{
				"pod_namespace", "pod_name", "svc_namespace", "svc_name", "svc_scope",
			})
		entry.Gauges[metricName] = &PromGauge{
			Name:  metricName,
			Cache: gauge,
		}
		err := prometheus.Register(gauge)
		if err != nil {
			agent.log.Error("Failed to register ", metricName, " with Prometheus: ", err)
		} else {
			agent.log.Debug("Registered ", metricName, " with Prometheus: ")
		}
	}
}

func (entry *PodSvcPromSubsystemEntry) GetGaugeVec(metricName string) *prometheus.GaugeVec {
	return entry.Gauges[metricName].Cache
}

func NewPodSvcPromSubsystemEntry() PromSubsystemEntry {
	promSubsystem := &PromSubsystem{
		Subsystem: "pod_svc_stats",
		Gauges:    make(map[string]*PromGauge),
	}

	return &PodSvcPromSubsystemEntry{
		PromSubsystem: promSubsystem,
	}

}
