package statsagent

import (
	"github.com/prometheus/client_golang/prometheus"
)

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

//SvcStats Prometheus Entries
var SvcPromMetrics = [...]string{
	"svc_tx_bytes",
	"svc_tx_packets",
	"svc_rx_bytes",
	"svc_rx_packets",
}

var SvcPromHelp = [...]string{
	"service egress bytes",
	"service egress packets",
	"service ingress bytes",
	"service ingress packets",
}

type SvcPromSubsystemEntry struct {
	*PromSubsystem
}

func (agent *StatsAgent) SetSvcGauge(
	key *PromMetricsKey,
	stats *FlowStats) {
	var value [4]uint64
	value[0] = stats.Out_bytes
	value[1] = stats.Out_packets
	value[2] = stats.In_bytes
	value[3] = stats.In_packets
	if key.metricName != "svc_stats" {
		return
	}
	for i := 0; i < 4; i++ {
		agent.promSubsystems["svc_stats"].GetGaugeVec(SvcPromMetrics[i]).With(prometheus.Labels{
			"svc_namespace": key.svcNamespace[0],
			"svc_scope":     key.svcScope[0],
			"svc_name":      key.svcName[0]}).Set(float64(value[i]))
	}
}

func (entry *SvcPromSubsystemEntry) SubsystemName() string {
	return entry.Subsystem
}

func (entry *SvcPromSubsystemEntry) RegisterPrometheus(agent *StatsAgent) {
	for i, metricName := range SvcPromMetrics {
		gauge :=
			prometheus.NewGaugeVec(prometheus.GaugeOpts{
				Namespace: "statsagent",
				Subsystem: "svc_stats",
				Name:      metricName,
				Help:      SvcPromHelp[i],
			}, []string{
				"svc_namespace", "svc_name", "svc_scope",
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

func (entry *SvcPromSubsystemEntry) GetGaugeVec(metricName string) *prometheus.GaugeVec {
	return entry.Gauges[metricName].Cache
}

func NewSvcPromSubsystemEntry() PromSubsystemEntry {
	promSubsystem := &PromSubsystem{
		Subsystem: "svc_stats",
		Gauges:    make(map[string]*PromGauge),
	}

	return &SvcPromSubsystemEntry{
		PromSubsystem: promSubsystem,
	}
}

//PodStats Prometheus Entries
var PodPromMetrics = [...]string{
	"pod_tx_bytes",
	"pod_tx_packets",
	"pod_rx_bytes",
	"pod_rx_packets",
}

var PodPromHelp = [...]string{
	"pod egress bytes",
	"pod egress packets",
	"pod ingress bytes",
	"pod ingress packets",
}

type PodPromSubsystemEntry struct {
	*PromSubsystem
}

func (agent *StatsAgent) SetPodGauge(
	key *PromMetricsKey,
	stats *FlowStats) {
	var value [4]uint64
	value[0] = stats.Out_bytes
	value[1] = stats.Out_packets
	value[2] = stats.In_bytes
	value[3] = stats.In_packets
	if key.metricName != "pod_stats" {
		return
	}
	for i := 0; i < 4; i++ {
		agent.promSubsystems["pod_stats"].GetGaugeVec(PodPromMetrics[i]).With(prometheus.Labels{
			"pod_namespace": key.podNamespace[0],
			"pod_name":      key.podName[0]}).Set(float64(value[i]))
	}
}

func (entry *PodPromSubsystemEntry) SubsystemName() string {
	return entry.Subsystem
}

func (entry *PodPromSubsystemEntry) RegisterPrometheus(agent *StatsAgent) {
	for i, metricName := range PodPromMetrics {
		gauge :=
			prometheus.NewGaugeVec(prometheus.GaugeOpts{
				Namespace: "statsagent",
				Subsystem: "pod_stats",
				Name:      metricName,
				Help:      PodPromHelp[i],
			}, []string{
				"pod_namespace", "pod_name",
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

func (entry *PodPromSubsystemEntry) GetGaugeVec(metricName string) *prometheus.GaugeVec {
	return entry.Gauges[metricName].Cache
}

func NewPodPromSubsystemEntry() PromSubsystemEntry {
	promSubsystem := &PromSubsystem{
		Subsystem: "pod_stats",
		Gauges:    make(map[string]*PromGauge),
	}

	return &PodPromSubsystemEntry{
		PromSubsystem: promSubsystem,
	}
}
