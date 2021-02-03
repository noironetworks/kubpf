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
	baseMap       map[inet_v4_flow]*FlowStatsEntry
	podStatsMap   map[PodStatsKey]*FlowStatsEntry
	svcStatsMap   map[PodStatsKey]*FlowStatsEntry
	knownStatsMap map[PodStatsKey]*FlowStatsEntry
	agent         *StatsAgent
	//	agingAck    chan bool
	stateMutex sync.Mutex
}

func NewInetV4FlowMetricsEntry(agent *StatsAgent) *InetV4FlowMetricsEntry {
	return &InetV4FlowMetricsEntry{
		baseMap:       make(map[inet_v4_flow]*FlowStatsEntry),
		podStatsMap:   make(map[PodStatsKey]*FlowStatsEntry),
		svcStatsMap:   make(map[PodStatsKey]*FlowStatsEntry),
		knownStatsMap: make(map[PodStatsKey]*FlowStatsEntry),
		agent:         agent,
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

func (metric *InetV4FlowMetricsEntry) mergeStats(keyType int, podStatsKey PodStatsKey, stats *FlowStats, t *time.Time) {
	copiedStats := *stats
	switch keyType {
	case FROM_POD_KEY:
		(&podStatsKey).clear(1)
		if _, cok := metric.podStatsMap[podStatsKey]; !cok {
			metric.podStatsMap[podStatsKey] = &FlowStatsEntry{}
		}
		metric.podStatsMap[podStatsKey].add(&copiedStats, t)
		promMetricsKey := podStatsKey.toPromMetricsKey(metric.agent)
		metric.agent.SetPodGauge(promMetricsKey, &metric.podStatsMap[podStatsKey].Stats)
	case TO_POD_KEY:
		(&podStatsKey).swap()
		(&podStatsKey).clear(1)
		(&copiedStats).swap()
		if _, cok := metric.podStatsMap[podStatsKey]; !cok {
			metric.podStatsMap[podStatsKey] = &FlowStatsEntry{}
		}
		metric.podStatsMap[podStatsKey].add(&copiedStats, t)
		promMetricsKey := podStatsKey.toPromMetricsKey(metric.agent)
		metric.agent.SetPodGauge(promMetricsKey, &metric.podStatsMap[podStatsKey].Stats)
	case FROM_SVC_KEY:
		(&podStatsKey).clear(1)
		if _, cok := metric.svcStatsMap[podStatsKey]; !cok {
			metric.svcStatsMap[podStatsKey] = &FlowStatsEntry{}
		}
		metric.svcStatsMap[podStatsKey].add(&copiedStats, t)
		promMetricsKey := podStatsKey.toPromMetricsKey(metric.agent)
		metric.agent.SetSvcGauge(promMetricsKey, &metric.svcStatsMap[podStatsKey].Stats)
	case TO_SVC_KEY:
		(&podStatsKey).swap()
		(&podStatsKey).clear(1)
		(&copiedStats).swap()
		if _, cok := metric.svcStatsMap[podStatsKey]; !cok {
			metric.svcStatsMap[podStatsKey] = &FlowStatsEntry{}
		}
		metric.svcStatsMap[podStatsKey].add(&copiedStats, t)
		promMetricsKey := podStatsKey.toPromMetricsKey(metric.agent)
		metric.agent.SetSvcGauge(promMetricsKey, &metric.svcStatsMap[podStatsKey].Stats)
	case FROM_POD_KEY | TO_POD_KEY:
		srcPodStatsKey := podStatsKey
		(&srcPodStatsKey).clear(1)
		if _, cok := metric.podStatsMap[srcPodStatsKey]; !cok {
			metric.podStatsMap[srcPodStatsKey] = &FlowStatsEntry{}
		}
		metric.podStatsMap[srcPodStatsKey].add(&copiedStats, t)
		promMetricsKey := srcPodStatsKey.toPromMetricsKey(metric.agent)
		metric.agent.SetPodGauge(promMetricsKey, &metric.podStatsMap[srcPodStatsKey].Stats)
		dstPodStatsKey := podStatsKey
		(&dstPodStatsKey).swap()
		(&dstPodStatsKey).clear(1)
		(&copiedStats).swap()
		if _, cok := metric.podStatsMap[dstPodStatsKey]; !cok {
			metric.podStatsMap[dstPodStatsKey] = &FlowStatsEntry{}
		}
		metric.podStatsMap[dstPodStatsKey].add(&copiedStats, t)
		promMetricsKey = dstPodStatsKey.toPromMetricsKey(metric.agent)
		metric.agent.SetPodGauge(promMetricsKey, &metric.podStatsMap[dstPodStatsKey].Stats)
	case FROM_SVC_KEY | TO_SVC_KEY:
		srcSvcStatsKey := podStatsKey
		(&srcSvcStatsKey).clear(1)
		if _, cok := metric.svcStatsMap[srcSvcStatsKey]; !cok {
			metric.svcStatsMap[srcSvcStatsKey] = &FlowStatsEntry{}
		}
		metric.svcStatsMap[srcSvcStatsKey].add(&copiedStats, t)
		promMetricsKey := srcSvcStatsKey.toPromMetricsKey(metric.agent)
		metric.agent.SetSvcGauge(promMetricsKey, &metric.svcStatsMap[srcSvcStatsKey].Stats)
		dstSvcStatsKey := podStatsKey
		(&dstSvcStatsKey).swap()
		(&dstSvcStatsKey).clear(1)
		(&copiedStats).swap()
		if _, cok := metric.svcStatsMap[dstSvcStatsKey]; !cok {
			metric.svcStatsMap[dstSvcStatsKey] = &FlowStatsEntry{}
		}
		metric.svcStatsMap[dstSvcStatsKey].add(&copiedStats, t)
		promMetricsKey = dstSvcStatsKey.toPromMetricsKey(metric.agent)
		metric.agent.SetSvcGauge(promMetricsKey, &metric.svcStatsMap[dstSvcStatsKey].Stats)
	case FROM_POD_KEY | TO_SVC_KEY:
		srcPodStatsKey := podStatsKey
		(&srcPodStatsKey).clear(1)
		if _, cok := metric.podStatsMap[srcPodStatsKey]; !cok {
			metric.podStatsMap[srcPodStatsKey] = &FlowStatsEntry{}
		}
		metric.podStatsMap[srcPodStatsKey].add(&copiedStats, t)
		promMetricsKey := srcPodStatsKey.toPromMetricsKey(metric.agent)
		metric.agent.SetPodGauge(promMetricsKey, &metric.podStatsMap[srcPodStatsKey].Stats)
		dstSvcStatsKey := podStatsKey
		(&dstSvcStatsKey).swap()
		(&dstSvcStatsKey).clear(1)
		(&copiedStats).swap()
		if _, cok := metric.svcStatsMap[dstSvcStatsKey]; !cok {
			metric.svcStatsMap[dstSvcStatsKey] = &FlowStatsEntry{}
		}
		metric.svcStatsMap[dstSvcStatsKey].add(&copiedStats, t)
		promMetricsKey = dstSvcStatsKey.toPromMetricsKey(metric.agent)
		metric.agent.SetSvcGauge(promMetricsKey, &metric.svcStatsMap[dstSvcStatsKey].Stats)
		if _, cok := metric.knownStatsMap[podStatsKey]; !cok {
			metric.knownStatsMap[podStatsKey] = &FlowStatsEntry{}
		}
		metric.knownStatsMap[podStatsKey].add(&copiedStats, t)
		promMetricsKey = podStatsKey.toPromMetricsKey(metric.agent)
		metric.agent.SetPodSvcGauge(promMetricsKey, &metric.knownStatsMap[podStatsKey].Stats)
	case FROM_SVC_KEY | TO_POD_KEY:
		srcSvcStatsKey := podStatsKey
		(&srcSvcStatsKey).clear(1)
		if _, cok := metric.svcStatsMap[srcSvcStatsKey]; !cok {
			metric.svcStatsMap[srcSvcStatsKey] = &FlowStatsEntry{}
		}
		metric.svcStatsMap[srcSvcStatsKey].add(&copiedStats, t)
		promMetricsKey := srcSvcStatsKey.toPromMetricsKey(metric.agent)
		metric.agent.SetSvcGauge(promMetricsKey, &metric.svcStatsMap[srcSvcStatsKey].Stats)
		dstPodStatsKey := podStatsKey
		(&dstPodStatsKey).swap()
		(&dstPodStatsKey).clear(1)
		(&copiedStats).swap()
		if _, cok := metric.podStatsMap[dstPodStatsKey]; !cok {
			metric.podStatsMap[dstPodStatsKey] = &FlowStatsEntry{}
		}
		metric.podStatsMap[dstPodStatsKey].add(&copiedStats, t)
		promMetricsKey = dstPodStatsKey.toPromMetricsKey(metric.agent)
		metric.agent.SetPodGauge(promMetricsKey, &metric.podStatsMap[dstPodStatsKey].Stats)
		if _, cok := metric.knownStatsMap[podStatsKey]; !cok {
			metric.knownStatsMap[podStatsKey] = &FlowStatsEntry{}
		}
		metric.knownStatsMap[podStatsKey].add(&copiedStats, t)
		promMetricsKey = podStatsKey.toPromMetricsKey(metric.agent)
		metric.agent.SetPodSvcGauge(promMetricsKey, &metric.knownStatsMap[podStatsKey].Stats)
	}
}

func (metric *InetV4FlowMetricsEntry) UpdateStats() {
	//<-metric.agingAck
	metric.stateMutex.Lock()
	mapPath := metric.agent.config.EbpfMapDir + "/" + "v4_flow_map"
	metric.agent.log.Debug("Reading map ", mapPath)
	m, err := ebpf.LoadPinnedMap(mapPath)
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
			podStatsKey, keyType := getPodStatsKey(metric.agent, &keyOut)
			metric.mergeStats(keyType, podStatsKey, &valueOut, &t)
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
		podStatsKey, keyType := getPodStatsKey(metric.agent, &keyOut)
		diffStats := diffFlowStats(&currStats.Stats, &valueOut)
		metric.mergeStats(keyType, podStatsKey, diffStats, &t)
	}
	var toDeleteKnownStatsList, toDeletePodStatsList, toDeleteSvcStatsList []PodStatsKey
	for k, v := range metric.knownStatsMap {
		if v.TimeStamp != t {
			metric.knownStatsMap[k].Aging_counter++
			if metric.knownStatsMap[k].Aging_counter >= 3 {
				toDeleteKnownStatsList = append(toDeleteKnownStatsList, k)
			}
		}
		metric.agent.log.Debugf("%s<->%s Out:%d bytes %d packets In: %d bytes %d packets, aging_count:%d",
			k.Endpoints[0], k.Endpoints[1], v.Stats.Out_bytes, v.Stats.Out_packets, v.Stats.In_bytes, v.Stats.In_packets,
			v.Aging_counter)

	}
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
	for k, v := range metric.svcStatsMap {
		if v.TimeStamp != t {
			metric.svcStatsMap[k].Aging_counter++
			if metric.svcStatsMap[k].Aging_counter >= 3 {
				toDeleteSvcStatsList = append(toDeleteSvcStatsList, k)
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
		m2, err2 := ebpf.LoadPinnedMap(mapPath)
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
		for _, toDeleteKnownStats := range toDeleteKnownStatsList {
			metric.agent.log.Debug("Deleting podStatsKey", toDeleteKnownStats.Endpoints[0], "->", toDeleteKnownStats.Endpoints[1])
			delete(metric.knownStatsMap, toDeleteKnownStats)
		}
		for _, toDeletePodStats := range toDeletePodStatsList {
			metric.agent.log.Debug("Deleting podStatsKey", toDeletePodStats.Endpoints[0], "->", toDeletePodStats.Endpoints[1])
			delete(metric.podStatsMap, toDeletePodStats)
		}
		for _, toDeleteSvcStats := range toDeleteSvcStatsList {
			metric.agent.log.Debug("Deleting podStatsKey", toDeleteSvcStats.Endpoints[0], "->", toDeleteSvcStats.Endpoints[1])
			delete(metric.svcStatsMap, toDeleteSvcStats)
		}
		//metric.agingAck <- true

	}()

}
