/*
Copyright 2024.

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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// InvalidKeplerResource indicates the CR name was invalid
	InvalidPowerMonitorResource ConditionReason = "InvalidPowerMonitorResource"
)

type PowerMonitorKeplerDeploymentSpec struct {
	// Defines which Nodes the Pod is scheduled on
	// +optional
	// +kubebuilder:default={"kubernetes.io/os":"linux"}
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// If specified, define Pod's tolerations
	// +optional
	// +kubebuilder:default={{"key": "", "operator": "Exists", "value": "", "effect": ""}}
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
}

type PowerMonitorKeplerConfigSpec struct {
	// +kubebuilder:default="info"
	// +optional
	LogLevel string `json:"logLevel,omitempty"`
}

type PowerMonitorKeplerSpec struct {
	Deployment PowerMonitorKeplerDeploymentSpec `json:"deployment"`
	Config     PowerMonitorKeplerConfigSpec     `json:"config,omitempty"`
}

// PowerMonitorSpec defines the desired state of Power Monitor
type PowerMonitorSpec struct {
	Kepler PowerMonitorKeplerSpec `json:"kepler"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:scope="Cluster"
//+kubebuilder:subresource:status

// +kubebuilder:printcolumn:name="Desired",type=integer,JSONPath=`.status.kepler.desiredNumberScheduled`
// +kubebuilder:printcolumn:name="Current",type=integer,JSONPath=`.status.kepler.currentNumberScheduled`
// +kubebuilder:printcolumn:name="Ready",type=integer,JSONPath=`.status.kepler.numberReady`
// +kubebuilder:printcolumn:name="Up-to-date",type=integer,JSONPath=`.status.kepler.updatedNumberScheduled`
// +kubebuilder:printcolumn:name="Available",type=integer,JSONPath=`.status.kepler.numberAvailable`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="Node-Selector",type=string,JSONPath=`.spec.kepler.deployment.nodeSelector`,priority=10
// +kubebuilder:printcolumn:name="Tolerations",type=string,JSONPath=`.spec.kepler.deployment.tolerations`,priority=10
//
// PowerMonitor is the Schema for the PowerMonitor API
type PowerMonitor struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PowerMonitorSpec   `json:"spec,omitempty"`
	Status PowerMonitorStatus `json:"status,omitempty"`
}

type PowerMonitorKeplerStatus struct {
	// The number of nodes that are running at least 1 power-monitor pod and are
	// supposed to run the power-monitor pod.
	CurrentNumberScheduled int32 `json:"currentNumberScheduled"`

	// The number of nodes that are running the power-monitor pod, but are not supposed
	// to run the power-monitor pod.
	NumberMisscheduled int32 `json:"numberMisscheduled"`

	// The total number of nodes that should be running the power-monitor
	// pod (including nodes correctly running the power-monitor pod).
	DesiredNumberScheduled int32 `json:"desiredNumberScheduled"`

	// numberReady is the number of nodes that should be running the power-monitor pod
	// and have one or more of the power-monitor pod running with a Ready Condition.
	NumberReady int32 `json:"numberReady"`

	// The total number of nodes that are running updated power-monitor pod
	// +optional
	UpdatedNumberScheduled int32 `json:"updatedNumberScheduled,omitempty"`

	// The number of nodes that should be running the power-monitor pod and have one or
	// more of the power-monitor pod running and available
	// +optional
	NumberAvailable int32 `json:"numberAvailable,omitempty"`

	// The number of nodes that should be running the
	// power-monitor pod and have none of the power-monitor pod running and available
	// +optional
	NumberUnavailable int32 `json:"numberUnavailable,omitempty"`

	// conditions represent the latest available observations of power-monitor
	// +operator-sdk:csv:customresourcedefinitions:type=status,xDescriptors="urn:alm:descriptor:com.tectonic.ui:conditions"
	// +listType=atomic
	Conditions []Condition `json:"conditions"`
}

// PowerMonitorStatus defines the observed state of Power Monitor
type PowerMonitorStatus struct {
	Kepler PowerMonitorKeplerStatus `json:"kepler,omitempty"`
}

//+kubebuilder:object:root=true

// PowerMonitorList contains a list of PowerMonitor
type PowerMonitorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PowerMonitor `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PowerMonitor{}, &PowerMonitorList{})
}
