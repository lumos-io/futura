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
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	vpa "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
)

type ServiceLevelObjectiveSpec struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	// ScaleTargetRef points to the controller managing the set of pods for the autoscaler to
	// control, e.g., Deployment, StatefulSet. ServiceLevelObjective can be targeted at controller
	// implementing scale subresource (the pod set is retrieved from the controller's ScaleStatus
	// or some well known controllers (e.g., for DaemonSet the pod set is read from the
	// controller's spec). If ServiceLevelObjective cannot use specified target it will report
	// the ConfigUnsupported condition.
	ScaleTargetRef *autoscalingv2.CrossVersionObjectReference `json:"scaleTargetRef"`

	// Describes the rules on how changes are applied to the pods.
	// If not specified, all fields in the `PodUpdatePolicy` are set to their default values.
	// +optional
	UpdatePolicy *PodUpdatePolicy `json:"updatePolicy,omitempty"`

	// Contains the specifications about the metric type and target in terms of resource
	// utilization or workload performance. See the individual metric source types for
	// more information about how each type of metric must respond.
	// +listType=atomic
	// +optional
	Metrics []autoscalingv2.MetricSpec `json:"metrics,omitempty"`

	// Describes the constraints for the number of replicas.
	Constraints *HorizontalScalingConstraints `json:"constraints,omitempty"`
	// Controls how the VPA autoscaler computes recommended resources.
	// The resource policy is also used to set constraints on the recommendations for individual
	// containers. If not specified, the autoscaler computes recommended resources for all
	// containers in the pod, without additional constraints.
	// +optional
	ResourcePolicy *vpa.PodResourcePolicy `json:"resourcePolicy,omitempty"`

	// Recommender responsible for generating recommendation for the set of pods and the deployment.
	// List should be empty (then the default recommender will be used) or contain exactly one
	// recommender.
	// +optional
	Recommenders []*PodAutoscalerRecommenderSelector `json:"recommenders,omitempty"`
}

// ServiceLevelObjectiveStatus defines the observed state of ServiceLevelObjective
type ServiceLevelObjectiveStatus struct {
	LastUpdated metav1.Time `json:"lastUpdated,omitempty"`
	Synced      bool        `json:"synced,omitempty"` // true if stored in API

	// Last time the PodAutoscaler scaled the number of pods and resizes containers;
	// Used by the autoscaler to control how often scaling operations are performed.
	// +optional
	LastScaleTime *metav1.Time `json:"lastScaleTime,omitempty"`

	// Current number of replicas of pods managed by this autoscaler.
	CurrentReplicas int32 `json:"currentReplicas"`

	// Desired number of replicas of pods managed by this autoscaler.
	DesiredReplicas int32 `json:"desiredReplicas"`

	// The most recently computed amount of resources for each controlled pod recommended by the
	// autoscaler.
	// +optional
	Recommendation *vpa.RecommendedPodResources `json:"recommendation,omitempty"`

	// The last read state of the metrics used by this autoscaler.
	// +listType=atomic
	// +optional
	CurrentMetrics []autoscalingv2.MetricStatus `json:"currentMetrics"`

	// Conditions is the set of conditions required for this autoscaler to scale its target, and
	// indicates whether or not those conditions are met.
	// +optional
	// +patchMergeKey=type
	// +patchStrategy=merge
	Conditions []PodAutoscalerCondition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
}

// PodUpdatePolicy describes the rules on how changes are applied to the pods.
type PodUpdatePolicy struct {
	// Controls when autoscaler applies changes to the pod resources.
	// The default is 'Auto'.
	// +optional
	UpdateMode *vpa.UpdateMode `json:"updateMode,omitempty"`
}

// HorizontalScalingConstraints describes the constraints for horizontal scaling.
type HorizontalScalingConstraints struct {
	// Lower limit for the number of pods that can be set by the autoscaler, default 1.
	// +optional
	MinReplicas *int32 `json:"minReplicas,omitempty"`
	// Upper limit for the number of pods that can be set by the autoscaler; cannot be smaller than
	// MinReplicas.
	MaxReplicas *int32 `json:"maxReplicas"`
	// Behavior configures the scaling behavior of the target in both Up and Down direction
	// (scaleUp and scaleDown fields respectively).
	// +optional
	Behavior *autoscalingv2.HorizontalPodAutoscalerBehavior `json:"behavior,omitempty"`
}

// PodAutoscalerRecommenderSelector points to a specific ServiceLevelObjective
// recommender.
// In the future it might pass parameters to the recommender.
type PodAutoscalerRecommenderSelector struct {
	// Name of the recommender responsible for generating recommendation for this object.
	Name string `json:"name"`
}

// PodAutoscalerCondition describes the state of a PodAutoscaler at a certain point.
type PodAutoscalerCondition struct {
	// type describes the current condition
	Type PodAutoscalerConditionType `json:"type"`
	// status is the status of the condition (True, False, Unknown)
	Status v1.ConditionStatus `json:"status"`
	// lastTransitionTime is the last time the condition transitioned from one status to another
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	// reason is the reason for the condition's last transition.
	// +optional
	Reason string `json:"reason,omitempty"`
	// message is a human-readable explanation containing details about the transition
	// +optional
	Message string `json:"message,omitempty"`
}

// PodAutoscalerConditionType are the valid conditions of a PodAutoscaler.
type PodAutoscalerConditionType string

var (
	// RecommendationProvided indicates whether the MPA recommender was able to give a
	// recommendation.
	RecommendationProvided PodAutoscalerConditionType = "RecommendationProvided"
	// LowConfidence indicates whether the MPA recommender has low confidence in the recommendation
	// for some of containers.
	LowConfidence PodAutoscalerConditionType = "LowConfidence"
	// NoPodsMatched indicates that label selector used with MPA object didn't match any pods.
	NoPodsMatched PodAutoscalerConditionType = "NoPodsMatched"
	// FetchingHistory indicates that MPA recommender is in the process of loading additional
	// history samples.
	FetchingHistory PodAutoscalerConditionType = "FetchingHistory"
	// ConfigDeprecated indicates that this MPA configuration is deprecated and will stop being
	// supported soon.
	ConfigDeprecated PodAutoscalerConditionType = "ConfigDeprecated"
	// ConfigUnsupported indicates that this MPA configuration is unsupported and recommendations
	// will not be provided for it.
	ConfigUnsupported PodAutoscalerConditionType = "ConfigUnsupported"
	// ScalingActive indicates that the MPA controller is able to scale if necessary, i.e.,
	// it is correctly configured, can fetch the desired metrics, and isn't disabled.
	ScalingActive PodAutoscalerConditionType = "ScalingActive"
	// AbleToScale indicates a lack of transient issues which prevent scaling from occurring,
	// such as being in a backoff window, or being unable to access/update the target scale.
	AbleToScale PodAutoscalerConditionType = "AbleToScale"
	// ScalingLimited indicates that the calculated scale based on metrics would be above or
	// below the range for the MPA, and has thus been capped.
	ScalingLimited PodAutoscalerConditionType = "ScalingLimited"
)

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
