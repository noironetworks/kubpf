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

	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/kubernetes/pkg/controller"
)

func (agent *StatsAgent) initServiceInformerFromClient(
	kubeClient *kubernetes.Clientset) {
	agent.initServiceInformerBase(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				return kubeClient.CoreV1().Services(metav1.NamespaceAll).List(context.TODO(), options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return kubeClient.CoreV1().Services(metav1.NamespaceAll).Watch(context.TODO(), options)
			},
		})
}

func (agent *StatsAgent) initServiceInformerBase(listWatch *cache.ListWatch) {
	agent.svcInformer = cache.NewSharedIndexInformer(
		listWatch,
		&v1.Service{},
		controller.NoResyncPeriodFunc(),
		cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
	)
	agent.svcInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			agent.serviceUpdated(obj)
		},
		UpdateFunc: func(_ interface{}, obj interface{}) {
			agent.serviceUpdated(obj)
		},
		DeleteFunc: func(obj interface{}) {
			agent.serviceDeleted(obj)
		},
	})
}

func (agent *StatsAgent) serviceUpdated(obj interface{}) {
	agent.stateMutex.Lock()
	defer agent.stateMutex.Unlock()

	svc := obj.(*v1.Service)

	key, err := cache.MetaNamespaceKeyFunc(svc)
	if err != nil {
		serviceLogger(agent.log, svc).
			Error("Could not create key:" + err.Error())
		return
	}
	if svc.Spec.ClusterIP == "None" {
		return
	}
	var svcInfo SvcInfo
	svcInfo.ClusterIP = svc.Spec.ClusterIP
	switch svc.Spec.Type {
	case v1.ServiceTypeClusterIP:
		svcInfo.SvcType = "clusterIp"
	case v1.ServiceTypeNodePort:
		svcInfo.SvcType = "nodePort"
	case v1.ServiceTypeLoadBalancer:
		svcInfo.SvcType = "loadBalancer"
	case v1.ServiceTypeExternalName:
		svcInfo.SvcType = "externalName"
	}
	agent.log.Debug("Added svc ", key)
	agent.svcInfo[key] = svcInfo
	agent.svcIpToName[svcInfo.ClusterIP] = key
}

func (agent *StatsAgent) serviceDeleted(obj interface{}) {
	agent.stateMutex.Lock()
	defer agent.stateMutex.Unlock()

	svc := obj.(*v1.Service)
	key, err := cache.MetaNamespaceKeyFunc(svc)
	if err != nil {
		serviceLogger(agent.log, svc).
			Error("Could not create key:" + err.Error())
		return
	}
	if svc.Spec.ClusterIP == "None" {
		return
	}
	agent.log.Debug("Deleting svc ", key)
	delete(agent.svcInfo, key)
	delete(agent.svcIpToName, svc.Spec.ClusterIP)
}

func serviceLogger(log *logrus.Logger, as *v1.Service) *logrus.Entry {
	return log.WithFields(logrus.Fields{
		"namespace": as.ObjectMeta.Namespace,
		"name":      as.ObjectMeta.Name,
		"type":      as.Spec.Type,
	})
}
