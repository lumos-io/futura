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

// ClusterOptimizationConfigSpec defines the desired state of ClusterOptimizationConfig
type ClusterOptimizationConfigSpec struct {
	// ApiKey specifies which customer/cluster this optimization is for.
	ApiKey string `json:"apiKey"`

	// CostOptimization settings for budget and instance preferences.
	CostOptimization CostOptimizationSettings `json:"costOptimization,omitempty"`

	// Quotas defines resource limits such as CPU, memory, etc.
	Quotas map[string]int `json:"quotas,omitempty"`

	// ScalingPolicies defines which scaling mechanisms to enable.
	ScalingPolicies ScalingPolicies `json:"scalingPolicies"`

	// SyncPeriodSeconds defines how often the optimization loop runs.
	// +kubebuilder:validation:Minimum=10
	SyncPeriodSeconds int `json:"syncPeriodSeconds,omitempty"`
}

// CostOptimizationSettings defines budget and instance preferences
type CostOptimizationSettings struct {
	CostSensitivity        string   `json:"costSensitivity,omitempty"`
	MaxMonthlyBudgetUSD    string   `json:"maxMonthlyBudgetUSD,omitempty"`
	SpotInstanceAllowed    bool     `json:"spotInstanceAllowed,omitempty"`
	PreferredInstanceTypes []string `json:"preferredInstanceTypes,omitempty"`
	MaxSpotPercentage      string   `json:"maxSpotPercentage,omitempty"`
}

// ScalingPolicies defines whether to enable HPA, VPA, or NodeHandler.
type ScalingPolicies struct {
	EnableHPA         bool `json:"enableHPA"`
	EnableVPA         bool `json:"enableVPA"`
	EnableNodeHandler bool `json:"enableNodeHandler"`
}

// ClusterOptimizationConfigStatus defines the observed state of ClusterOptimizationConfig
type ClusterOptimizationConfigStatus struct {
	LastSynced metav1.Time `json:"lastSynced,omitempty"`
	Synced     bool        `json:"synced,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ClusterOptimizationConfig is the Schema for the clusteroptimizationconfigs API
type ClusterOptimizationConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterOptimizationConfigSpec   `json:"spec,omitempty"`
	Status ClusterOptimizationConfigStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ClusterOptimizationConfigList contains a list of ClusterOptimizationConfig
type ClusterOptimizationConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterOptimizationConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ClusterOptimizationConfig{}, &ClusterOptimizationConfigList{})
}
