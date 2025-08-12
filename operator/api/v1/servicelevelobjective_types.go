/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ServiceLevelObjectiveSpec struct {
	// Target service name this SLO applies to
	ServiceName string `json:"serviceName"`

	// Target p95 latency (string, e.g., "250ms" or "0.25s")
	TargetP95Latency string `json:"targetP95Latency"`

	// Target error rate (string, e.g., "0.01" for 1%)
	TargetErrorRate string `json:"targetErrorRate"`

	// Target throughput / requests per second (string, e.g., "1000")
	TargetThroughput string `json:"targetThroughput"`

	// Priority level: "low", "medium", "high"
	Priority string `json:"priority"`
}

// ServiceLevelObjectiveStatus defines the observed state of ServiceLevelObjective
type ServiceLevelObjectiveStatus struct {
	LastSyncTime *metav1.Time `json:"lastSyncTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type ServiceLevelObjective struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ServiceLevelObjectiveSpec   `json:"spec,omitempty"`
	Status ServiceLevelObjectiveStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type ServiceLevelObjectiveList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ServiceLevelObjective `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ServiceLevelObjective{}, &ServiceLevelObjectiveList{})
}
