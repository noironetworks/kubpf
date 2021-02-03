// Copyright 2021 Cisco Systems, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package statsagent

import (
	"flag"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/tools/cache"
	"sync"
)

type PodInfo struct {
	PodIP string
}

type SvcInfo struct {
	ClusterIP string
	SvcType   string
}

type StatsAgent struct {
	config         *StatsAgentConfig
	log            *logrus.Logger
	env            Environment
	podInformer    cache.SharedIndexInformer
	svcInformer    cache.SharedIndexInformer
	podInfo        map[string]PodInfo
	podIpToName    map[string]string
	svcInfo        map[string]SvcInfo
	svcIpToName    map[string]string
	stateMutex     sync.Mutex
	metrics        map[string]MetricsEntry
	promSubsystems map[string]PromSubsystemEntry
}

type StatsAgentConfig struct {
	// Log level
	LogLevel string `json:"log-level,omitempty"`

	// Absolute path to a kubeconfig file
	//KubeConfig string `json:"kubeconfig,omitempty"`

	// Name of Kubernetes node on which this agent is running
	NodeName string `json:"node-name,omitempty"`

	// Path to which ebpf maps should be pinned, some where in /sys/fs/bpf
	EbpfMapDir string `json:"ebpf-map-dir,omitempty"`

	// Cgroup root for kubernetes
	CgroupRoot string `json:"cgroup-root,omitempty"`

	// Interval in which stats should be scanned
	StatsInterval int `json: stats-interval in seconds,omitempty`

	// TCP port to run status server on (or 0 to disable)
	StatusPort int `json:"status-port,omitempty"`
}

func (config *StatsAgentConfig) InitFlags() {
	flag.StringVar(&config.LogLevel, "log-level", "debug", "Log level")
	flag.StringVar(&config.EbpfMapDir, "ebpf-map-dir", "/sys/fs/bpf/pinned_maps", "Path to which ebpf maps should be pinned")
	flag.StringVar(&config.CgroupRoot, "cgroup-root", "/sys/fs/cgroup/unified/kubepods.slice", "Cgroup root for monitored instance of kubernetes")
	flag.StringVar(&config.NodeName, "node-name", "", "Name of Kubernetes node on which this agent is running")
	flag.IntVar(&config.StatusPort, "status-port", 8010, "TCP port to run status server on (or 0 to disable)")
	flag.IntVar(&config.StatsInterval, "stats-interval", 120, "Time in seconds between stats collection runs")
}

func NewStatsAgent(config *StatsAgentConfig, logger *logrus.Logger, env Environment) *StatsAgent {

	statsAgent := &StatsAgent{
		config:         config,
		log:            logger,
		env:            env,
		podInfo:        make(map[string]PodInfo),
		podIpToName:    make(map[string]string),
		svcInfo:        make(map[string]SvcInfo),
		svcIpToName:    make(map[string]string),
		metrics:        make(map[string]MetricsEntry),
		promSubsystems: make(map[string]PromSubsystemEntry),
	}
	return statsAgent
}

func (agent *StatsAgent) Init() {
	err := agent.env.Init(agent)
	if err != nil {
		panic(err.Error())
	}
}

func (agent *StatsAgent) Run(stopCh <-chan struct{}) {
	_, err := agent.env.PrepareRun(stopCh)
	if err != nil {
		panic(err.Error())
	}
	go func() {
		<-stopCh
	}()
}
