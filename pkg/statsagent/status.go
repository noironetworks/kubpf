// Copyright 2017 Cisco Systems, Inc.
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
	"encoding/json"
	"fmt"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
)

type agentStatus struct {
	PodCount int `json:"pod-count,omitempty"`
}

func (agent *StatsAgent) RunStatus() {
	if agent.config.StatusPort <= 0 {
		agent.log.Info("Status server is disabled")
		return
	}

	http.Handle("/metrics", promhttp.Handler())

	http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		agent.stateMutex.Lock()
		status := &agentStatus{
			PodCount: len(agent.podInfo),
		}
		json.NewEncoder(w).Encode(status)
		agent.stateMutex.Unlock()
	})
	agent.log.Info("Starting status server on ", agent.config.StatusPort)
	panic(http.ListenAndServe(fmt.Sprintf(":%d", agent.config.StatusPort), nil))
}
