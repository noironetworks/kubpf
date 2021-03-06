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
	"bytes"
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"

	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"os"
	"os/exec"
	"path/filepath"
)

type Environment interface {
	Init(agent *StatsAgent) error
	PrepareRun(stopCh <-chan struct{}) (bool, error)
}

type K8sEnvironment struct {
	kubeClient *kubernetes.Clientset
	agent      *StatsAgent
	//        serviceInformer     cache.SharedIndexInformer
	//        nodeInformer        cache.SharedIndexInformer
}

func NewK8sEnvironment(config *StatsAgentConfig, log *logrus.Logger) (*K8sEnvironment, error) {

	if config.NodeName == "" {
		config.NodeName = os.Getenv("KUBERNETES_NODE_NAME")
	}
	if config.NodeName == "" {
		err := errors.New("Node name not specified and $KUBERNETES_NODE_NAME empty")
		log.Error(err.Error())
		return nil, err
	}

	envCgroupRoot := os.Getenv("CGROUP_ROOT")
	if envCgroupRoot != "" {
		config.CgroupRoot = envCgroupRoot
	}
	mapDir, _ := filepath.Rel("/sys/fs/bpf", config.EbpfMapDir)
	mapDir = "/ebpf/" + mapDir
	config.EbpfMapDir = mapDir
	cgroupRoot, _ := filepath.Rel("/sys/fs/cgroup", config.CgroupRoot)
	cgroupRoot = "/cgroup/" + cgroupRoot
	log.Debug("Using cgroup ", cgroupRoot, " map directory ", mapDir)
	mapStr := fmt.Sprintf("EBPF_MAP_DIR=%s", mapDir)
	cgroupStr := fmt.Sprintf("CGROUP_MOUNT=%s", cgroupRoot)
	cmd := exec.Command("/bin/load_attach_bpf_cgroup.sh")
	cmd.Env = append(os.Environ(), mapStr, cgroupStr)
	var out bytes.Buffer
	cmd.Stderr = &out
	err := cmd.Run()
	log.Debug(out.String())
	if err != nil {
		log.Error(err.Error())
	}

	log.WithFields(logrus.Fields{
		"node-name": config.NodeName,
	}).Info("Setting up Kubernetes environment")

	log.Debug("Initializing kubernetes client")
	var restconfig *restclient.Config
	// creates the in-cluster config
	restconfig, err = restclient.InClusterConfig()
	if err != nil {
		return nil, err
	}

	// creates the kubernetes API client
	kubeClient, err := kubernetes.NewForConfig(restconfig)
	if err != nil {
		log.Debug("Failed to intialize kube client")
		return nil, err
	}

	return &K8sEnvironment{kubeClient: kubeClient}, nil
}

func (env *K8sEnvironment) PrepareRun(stopCh <-chan struct{}) (bool, error) {
	//env.agent.log.Debug("Discovering node configuration")
	//env.agent.log.Debug("Starting node informer")
	//go env.agent.nodeInformer.Run(stopCh)
	//env.agent.log.Info("Waiting for node cache sync")
	//cache.WaitForCacheSync(stopCh, env.agent.nodeInformer.HasSynced)
	//env.agent.log.Info("Node cache sync successful")
	//env.agent.log.Debug("Starting remaining informers")
	//env.agent.log.Debug("Exporting node info: ", env.agent.config.NodeName)
	go env.agent.podInformer.Run(stopCh)
	go env.agent.svcInformer.Run(stopCh)
	cache.WaitForCacheSync(stopCh, env.agent.podInformer.HasSynced)
	//go env.agent.controllerInformer.Run(stopCh)
	//env.agent.serviceEndPoints.Run(stopCh)
	//go env.agent.serviceInformer.Run(stopCh)
	//go env.agent.nsInformer.Run(stopCh)
	//env.agent.log.Info("Waiting for cache sync for remaining objects")
	env.agent.log.Info("Cache sync successful")
	go env.agent.RunMetrics(stopCh)
	return true, nil
}

func (env *K8sEnvironment) Init(agent *StatsAgent) error {
	env.agent = agent

	env.agent.log.Debug("Initializing informers")
	//env.agent.initNodeInformerFromClient(env.kubeClient)
	env.agent.initPodInformerFromClient(env.kubeClient)
	env.agent.initServiceInformerFromClient(env.kubeClient)
	//env.agent.serviceEndPoints.InitClientInformer(env.kubeClient)
	//env.agent.initNamespaceInformerFromClient(env.kubeClient)
	env.agent.log.Debug("Registering Metrics")
	env.agent.registerMetrics()
	env.agent.registerPrometheusMetrics()
	return nil
}
