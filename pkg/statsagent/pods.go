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
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"k8s.io/kubernetes/pkg/controller"
)

func (agent *StatsAgent) initPodInformerFromClient(
	kubeClient *kubernetes.Clientset) {

	agent.initPodInformerBase(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				return kubeClient.CoreV1().Pods(metav1.NamespaceAll).List(context.TODO(), options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return kubeClient.CoreV1().Pods(metav1.NamespaceAll).Watch(context.TODO(), options)
			},
		})

}

func (agent *StatsAgent) initPodInformerBase(listWatch *cache.ListWatch) {
	agent.podInformer = cache.NewSharedIndexInformer(
		listWatch,
		&v1.Pod{},
		controller.NoResyncPeriodFunc(),
		cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
	)
	agent.podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			agent.podUpdated(obj)
		},
		UpdateFunc: func(_ interface{}, obj interface{}) {
			agent.podUpdated(obj)
		},
		DeleteFunc: func(obj interface{}) {
			agent.podDeleted(obj)
		},
	})
}

func (agent *StatsAgent) podUpdated(obj interface{}) {
	pod := obj.(*v1.Pod)
	podKey := fmt.Sprintf("%s/%s", pod.ObjectMeta.Namespace, pod.ObjectMeta.Name)
	var podInfo PodInfo
	podInfo.PodIP = pod.Status.PodIP
	agent.stateMutex.Lock()
	defer agent.stateMutex.Unlock()
	agent.podInfo[podKey] = podInfo
	agent.podIpToName[pod.Status.PodIP] = podKey
	agent.log.Debug("Added pod ", podKey)
}

func (agent *StatsAgent) podDeleted(obj interface{}) {
	pod := obj.(*v1.Pod)
	podKey := fmt.Sprintf("%s/%s", pod.ObjectMeta.Namespace, pod.ObjectMeta.Name)
	agent.stateMutex.Lock()
	defer agent.stateMutex.Unlock()
	delete(agent.podInfo, podKey)
	delete(agent.podIpToName, pod.Status.PodIP)
	agent.log.Debug("Deleted pod ", podKey)
}
