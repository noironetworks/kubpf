package statsagent

import (
	"fmt"
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
		podStatsKey.Endpoints[0] = srcName
	}
	if dok {
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

func (agent *StatsAgent) RegisterMetrics() {
	agent.registerMetric("v4PodStats", NewInetV4FlowMetricsEntry(agent))
	//agent.registerMetric("v6PodStats", agent.getNewV6PodMetricEntry())
}
