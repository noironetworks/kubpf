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

package main

import (
	"flag"
	"github.com/shastrinator/kubpf/pkg/statsagent"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"
)

func main() {
	log := logrus.New()
	conf := &statsagent.StatsAgentConfig{}
	conf.InitFlags()
	flag.Parse()
	logLevel, err := logrus.ParseLevel(conf.LogLevel)
	if err != nil {
		panic(err.Error())
	}
	log.Level = logLevel
	var env statsagent.Environment
	env, err = statsagent.NewK8sEnvironment(conf, log)
	if err != nil {
		panic(err.Error())
	}
	agent := statsagent.NewStatsAgent(conf, log, env)
	agent.Init()
	agent.Run(wait.NeverStop)
	agent.RunStatus()
}
