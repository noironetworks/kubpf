package statsagent

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/cilium/ebpf"
	"net"
	"sync"
	"time"
)

type proto_port struct {
	Ip_proto uint8
	Padding  [3]uint8
	Sport    uint16
	Dport    uint16
}

func (l4 *proto_port) GetIpProto() string {
	return fmt.Sprintf("%d", l4.Ip_proto)
}

func (l4 *proto_port) GetSPort() string {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, l4.Sport)
	return fmt.Sprintf("%d", binary.LittleEndian.Uint16(buf.Bytes()))
}

func (l4 *proto_port) GetDPort() string {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, l4.Dport)
	return fmt.Sprintf("%d", binary.LittleEndian.Uint16(buf.Bytes()))
}

type inet_v4_flow struct {
	Src_ip uint32
	Dst_ip uint32
	L4     proto_port
}

func (flow *inet_v4_flow) GetSrcIp() string {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, flow.Src_ip)
	return net.IP(buf.Bytes()).String()
}

func (flow *inet_v4_flow) GetDstIp() string {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, flow.Dst_ip)
	return net.IP(buf.Bytes()).String()
}

func (flow *inet_v4_flow) GetIpProto() string {
	return flow.L4.GetIpProto()
}

func (flow *inet_v4_flow) GetSPort() string {
	return flow.L4.GetSPort()
}

func (flow *inet_v4_flow) GetDPort() string {
	return flow.L4.GetDPort()
}

//Interface MetricsEntry
type InetV4FlowMetricsEntry struct {
	baseMap     map[inet_v4_flow]*FlowStatsEntry
	podStatsMap map[PodStatsKey]*FlowStatsEntry
	agent       *StatsAgent
	//	agingAck    chan bool
	stateMutex sync.Mutex
}

func NewInetV4FlowMetricsEntry(agent *StatsAgent) *InetV4FlowMetricsEntry {
	return &InetV4FlowMetricsEntry{
		baseMap:     make(map[inet_v4_flow]*FlowStatsEntry),
		podStatsMap: make(map[PodStatsKey]*FlowStatsEntry),
		agent:       agent,
		//		agingAck:    make(chan bool),
	}
}

func (metric *InetV4FlowMetricsEntry) GetStats() {
	metric.stateMutex.Lock()
	defer metric.stateMutex.Unlock()
	for k, v := range metric.podStatsMap {
		fmt.Printf("%s<->%s Out:%d bytes %d packets In: %d bytes %d packets, aging_count:%d\n",
			k.Endpoints[0], k.Endpoints[1], v.Stats.Out_bytes, v.Stats.Out_packets, v.Stats.In_bytes, v.Stats.In_packets,
			v.Aging_counter)
	}
}

func (metric *InetV4FlowMetricsEntry) Run(stopCh <-chan struct{}) {
	runMetric(metric, stopCh)
}

func (metric *InetV4FlowMetricsEntry) GetStatsInterval() int {
	return metric.agent.config.StatsInterval
}

func (metric *InetV4FlowMetricsEntry) Init() {
	metric.agent.log.Debug("Setting channel to kickoff stats")
	//metric.agingAck <- true
}

func (metric *InetV4FlowMetricsEntry) UpdateStats() {
	metric.agent.log.Debug("Waiting for channel")
	//<-metric.agingAck
	metric.agent.log.Debug("Got channel")
	metric.stateMutex.Lock()
	m, err := ebpf.LoadPinnedMap("/ebpf/pinned_maps/v4_flow_map")
	if err != nil {
		metric.agent.log.Error(err)
		return
	}
	mIter := m.Iterate()
	t := time.Now()
	var keyOut inet_v4_flow
	var valueOut FlowStats
	var toDeleteList []inet_v4_flow
	for mIter.Next(&keyOut, &valueOut) {
		//metric.agent.log.Debug(parseFlow(&keyOut, &valueOut))
		currStats, preexisting := metric.baseMap[keyOut]
		if !preexisting {
			metric.baseMap[keyOut] = &FlowStatsEntry{}
			metric.baseMap[keyOut].Stats = valueOut
			metric.baseMap[keyOut].Aging_counter = 0
			metric.baseMap[keyOut].TimeStamp = t
			podStatsKey, pok := getPodStatsKey(metric.agent, &keyOut)
			if !pok {
				continue
			}
			_, cok := metric.podStatsMap[podStatsKey]
			if !cok {
				metric.agent.log.Debug("Added podStatsKey", podStatsKey.Endpoints[0], "->", podStatsKey.Endpoints[1])
				metric.podStatsMap[podStatsKey] = &FlowStatsEntry{}
			}
			addFlowStats(&metric.podStatsMap[podStatsKey].Stats, &valueOut)
			metric.podStatsMap[podStatsKey].Aging_counter = 0
			metric.podStatsMap[podStatsKey].TimeStamp = t
			promMetricsKey := podStatsKey.toPromMetricsKey(metric.agent)
			metric.agent.SetPodSvcGauge(promMetricsKey, &metric.podStatsMap[podStatsKey].Stats)
			continue
		}
		if currStats.Stats == valueOut {
			metric.baseMap[keyOut].Aging_counter++
			if metric.baseMap[keyOut].Aging_counter >= 3 {
				toDeleteList = append(toDeleteList, keyOut)
			}
			continue
		}
		metric.baseMap[keyOut].Stats = valueOut
		metric.baseMap[keyOut].Aging_counter = 0
		metric.baseMap[keyOut].TimeStamp = t
		podStatsKey, pok := getPodStatsKey(metric.agent, &keyOut)
		if !pok {
			continue
		}
		diffStats := diffFlowStats(&currStats.Stats, &valueOut)
		if _, psok := metric.podStatsMap[podStatsKey]; !psok {
			metric.agent.log.Error("PodStatsKey missing for ", podStatsKey.Endpoints[0], "->", podStatsKey.Endpoints[1])
			continue
		}
		addFlowStats(&metric.podStatsMap[podStatsKey].Stats, diffStats)
		metric.podStatsMap[podStatsKey].Aging_counter = 0
		metric.podStatsMap[podStatsKey].TimeStamp = t
		promMetricsKey := podStatsKey.toPromMetricsKey(metric.agent)
		metric.agent.SetPodSvcGauge(promMetricsKey, &metric.podStatsMap[podStatsKey].Stats)
	}
	var toDeletePodStatsList []PodStatsKey
	for k, v := range metric.podStatsMap {
		if v.TimeStamp != t {
			metric.podStatsMap[k].Aging_counter++
			if metric.podStatsMap[k].Aging_counter >= 3 {
				toDeletePodStatsList = append(toDeletePodStatsList, k)
			}
		}
		metric.agent.log.Debugf("%s<->%s Out:%d bytes %d packets In: %d bytes %d packets, aging_count:%d",
			k.Endpoints[0], k.Endpoints[1], v.Stats.Out_bytes, v.Stats.Out_packets, v.Stats.In_bytes, v.Stats.In_packets,
			v.Aging_counter)

	}
	m.Close()
	metric.stateMutex.Unlock()

	go func() {
		metric.stateMutex.Lock()
		defer metric.stateMutex.Unlock()
		m2, err2 := ebpf.LoadPinnedMap("/ebpf/pinned_maps/v4_flow_map")
		defer m2.Close()
		if err2 != nil {
			metric.agent.log.Error(err2)
			return
		}
		for _, toDelete := range toDeleteList {
			//pStr := fmt.Sprintf("%s(:%s)--[%s]-->%s(:%s)", toDelete.GetSrcIp(),
			//	toDelete.GetSPort(), toDelete.GetIpProto(), toDelete.GetDstIp(), toDelete.GetDPort())
			//metric.agent.log.Debug("Deleting ", pStr)
			err3 := m2.Delete(toDelete)
			if err3 != nil {
				metric.agent.log.Error("Failed to delete from basemap: ", err3)
			}
			delete(metric.baseMap, toDelete)
		}
		for _, toDeletePodStats := range toDeletePodStatsList {
			metric.agent.log.Debug("Deleting podStatsKey", toDeletePodStats.Endpoints[0], "->", toDeletePodStats.Endpoints[1])
			delete(metric.podStatsMap, toDeletePodStats)
		}
		//metric.agingAck <- true

	}()

}
