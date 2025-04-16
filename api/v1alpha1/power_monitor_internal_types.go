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

type PowerMonitorInternalDashboardSpec struct {
	// +kubebuilder:default=false
	Enabled bool `json:"enabled,omitempty"`
}

type PowerMonitorInternalOpenShiftSpec struct {
	// +kubebuilder:default=true
	Enabled   bool                              `json:"enabled"`
	Dashboard PowerMonitorInternalDashboardSpec `json:"dashboard,omitempty"`
}

type PowerMonitorInternalKeplerDeploymentSpec struct {
	// +kubebuilder:validation:MinLength=3
	Image string `json:"image"`

	// +kubebuilder:validation:MinLength=1
	Namespace string `json:"namespace"`

	// Defines which Nodes the Pod is scheduled on
	// +optional
	// +kubebuilder:default={"kubernetes.io/os":"linux"}
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// If specified, define Pod's tolerations
	// +optional
	// +kubebuilder:default={{"key": "", "operator": "Exists", "value": "", "effect": ""}}
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
}

type PowerMonitorInternalKeplerConfigSpec struct {
	// +kubebuilder:default="info"
	LogLevel string `json:"logLevel,omitempty"`
}

type PowerMonitorInternalKeplerSpec struct {
	// +kubebuilder:validation:Required
	Deployment PowerMonitorInternalKeplerDeploymentSpec `json:"deployment"`
	Config     PowerMonitorInternalKeplerConfigSpec     `json:"config,omitempty"`
}

// PowerMonitorInternalSpec defines the desired state of PowerMonitorInternalSpec
type PowerMonitorInternalSpec struct {
	// +kubebuilder:validation:Required
	Kepler    PowerMonitorInternalKeplerSpec    `json:"kepler"`
	OpenShift PowerMonitorInternalOpenShiftSpec `json:"openshift,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:scope="Cluster"
//+kubebuilder:subresource:status

// +kubebuilder:printcolumn:name="Desired",type=integer,JSONPath=`.status.kepler.desiredNumberScheduled`
// +kubebuilder:printcolumn:name="Current",type=integer,JSONPath=`.status.kepler.currentNumberScheduled`
// +kubebuilder:printcolumn:name="Up-to-date",type=integer,JSONPath=`.status.kepler.updatedNumberScheduled`
// +kubebuilder:printcolumn:name="Ready",type=integer,JSONPath=`.status.kepler.numberReady`
// +kubebuilder:printcolumn:name="Available",type=integer,JSONPath=`.status.kepler.numberAvailable`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="Image",type=string,JSONPath=`.spec.kepler.deployment.image`
// +kubebuilder:printcolumn:name="Node-Selector",type=string,JSONPath=`.spec.kepler.deployment.nodeSelector`,priority=10
// +kubebuilder:printcolumn:name="Tolerations",type=string,JSONPath=`.spec.kepler.deployment.tolerations`,priority=10
//
// PowerMonitorInternal is the Schema for the internal kepler 2 API
type PowerMonitorInternal struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PowerMonitorInternalSpec   `json:"spec,omitempty"`
	Status PowerMonitorInternalStatus `json:"status,omitempty"`
}

type PowerMonitorInternalKeplerStatus struct {
	// The number of nodes that are running at least 1 power-monitor-internal pod and are
	// supposed to run the power-monitor-internal pod.
	CurrentNumberScheduled int32 `json:"currentNumberScheduled"`

	// The number of nodes that are running the power-monitor-internal pod, but are not supposed
	// to run the power-monitor-internal pod.
	NumberMisscheduled int32 `json:"numberMisscheduled"`

	// The total number of nodes that should be running the power-monitor-internal
	// pod (including nodes correctly running the power-monitor-internal pod).
	DesiredNumberScheduled int32 `json:"desiredNumberScheduled"`

	// numberReady is the number of nodes that should be running the power-monitor-internal pod
	// and have one or more of the power-monitor-internal pod running with a Ready Condition.
	NumberReady int32 `json:"numberReady"`

	// The total number of nodes that are running updated power-monitor-internal pod
	// +optional
	UpdatedNumberScheduled int32 `json:"updatedNumberScheduled,omitempty"`

	// The number of nodes that should be running the power-monitor-internal pod and have one or
	// more of the power-monitor-internal pod running and available
	// +optional
	NumberAvailable int32 `json:"numberAvailable,omitempty"`

	// The number of nodes that should be running the
	// power-monitor-internal pod and have none of the power-monitor-internal pod running and available
	// +optional
	NumberUnavailable int32 `json:"numberUnavailable,omitempty"`

	// conditions represent the latest available observations of power-monitor-internal
	// +operator-sdk:csv:customresourcedefinitions:type=status,xDescriptors="urn:alm:descriptor:com.tectonic.ui:conditions"
	// +listType=atomic
	Conditions []Condition `json:"conditions"`
}

type PowerMonitorInternalStatus struct {
	Kepler PowerMonitorInternalKeplerStatus `json:"kepler,omitempty"`
}

func (pmi PowerMonitorInternal) Namespace() string {
	return pmi.Spec.Kepler.Deployment.Namespace
}

func (pmi PowerMonitorInternal) DaemonsetName() string {
	return pmi.Name
}

func (pmi PowerMonitorInternal) ServiceAccountName() string {
	return pmi.Name
}

func (pmi PowerMonitorInternal) FQServiceAccountName() string {
	return "system:serviceaccount:" + pmi.Namespace() + ":" + pmi.Name
}

//+kubebuilder:object:root=true

// PowerMonitorInternalList contains a list of PowerMonitorInternal
type PowerMonitorInternalList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PowerMonitorInternal `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PowerMonitorInternal{}, &PowerMonitorInternalList{})
}
