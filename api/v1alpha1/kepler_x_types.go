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

// NOTE: all internal types can depend on public types

type KeplerXDeploymentSpec struct {
	// +kubebuilder:validation:MinLength=3
	Image string `json:"image"`

	// +kubebuilder:validation:MinLength=1
	Namespace string `json:"namespace"`

	// +kubebuilder:default=9103
	// +kubebuilder:validation:Maximum=65535
	// +kubebuilder:validation:Minimum=1
	Port int32 `json:"port,omitempty"`

	// Defines which Nodes the Pod is scheduled on
	// +optional
	// +kubebuilder:default={"kubernetes.io/os":"linux"}
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// If specified, define Pod's tolerations
	// +optional
	// +kubebuilder:default={{"key": "", "operator": "Exists", "value": "", "effect": ""}}
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
}

type KeplerXConfigSpec struct {
	// +kubebuilder:default=3
	// +kubebuilder:validation:Maximum=500
	// +kubebuilder:validation:Minimum=1
	MonitorInterval int32 `json:"monitorInterval,omitempty"`

	// +kubebuilder:default=3
	// +kubebuilder:validation:Maximum=500
	// +kubebuilder:validation:Minimum=1
	ExporterInterval int32 `json:"exporterInterval,omitempty"`

	// +kubebuilder:default=100
	// +kubebuilder:validation:Minimum=0
	MaxTrackedTerminatedProcesses int32 `json:"maxTrackedTerminatedProcesses,omitempty"`

	// +kubebuilder:default=3
	// +kubebuilder:validation:Minimum=1
	ProcessRetentionPeriod int32 `json:"processRetentionPeriod,omitempty"`
}

type KeplerXDashboardSpec struct {
	// +kubebuilder:default=false
	Enabled bool `json:"enabled,omitempty"`
}

type KeplerXOpenShiftSpec struct {
	// +kubebuilder:default=true
	Enabled   bool                 `json:"enabled"`
	Dashboard KeplerXDashboardSpec `json:"dashboard,omitempty"`
}

// KeplerXSpec defines the desired state of KeplerX
type KeplerXSpec struct {
	// +kubebuilder:validation:Required
	Deployment    KeplerXDeploymentSpec `json:"deployment"`
	Configuration KeplerXConfigSpec     `json:"configuration,omitempty"`
	OpenShift     KeplerXOpenShiftSpec  `json:"openshift,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:scope="Cluster"
//+kubebuilder:subresource:status

// +kubebuilder:printcolumn:name="Port",type=integer,JSONPath=`.spec.deployment.port`
// +kubebuilder:printcolumn:name="Desired",type=integer,JSONPath=`.status.desiredNumberScheduled`
// +kubebuilder:printcolumn:name="Current",type=integer,JSONPath=`.status.currentNumberScheduled`
// +kubebuilder:printcolumn:name="Up-to-date",type=integer,JSONPath=`.status.updatedNumberScheduled`
// +kubebuilder:printcolumn:name="Ready",type=integer,JSONPath=`.status.numberReady`
// +kubebuilder:printcolumn:name="Available",type=integer,JSONPath=`.status.numberAvailable`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="Image",type=string,JSONPath=`.spec.deployment.image`
// +kubebuilder:printcolumn:name="Node-Selector",type=string,JSONPath=`.spec.deployment.nodeSelector`,priority=10
// +kubebuilder:printcolumn:name="Tolerations",type=string,JSONPath=`.spec.deployment.tolerations`,priority=10
//
// KeplerX is the Schema for the internal kepler 2 API
type KeplerX struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KeplerXSpec   `json:"spec,omitempty"`
	Status KeplerXStatus `json:"status,omitempty"`
}

type KeplerXStatus struct {
	// The number of nodes that are running at least 1 kepler-x pod and are
	// supposed to run the kepler-x pod.
	CurrentNumberScheduled int32 `json:"currentNumberScheduled"`

	// The number of nodes that are running the kepler-x pod, but are not supposed
	// to run the kepler-x pod.
	NumberMisscheduled int32 `json:"numberMisscheduled"`

	// The total number of nodes that should be running the kepler-x
	// pod (including nodes correctly running the kepler-x pod).
	DesiredNumberScheduled int32 `json:"desiredNumberScheduled"`

	// numberReady is the number of nodes that should be running the kepler-x pod
	// and have one or more of the kepler-x pod running with a Ready Condition.
	NumberReady int32 `json:"numberReady"`

	// The total number of nodes that are running updated kepler-x pod
	// +optional
	UpdatedNumberScheduled int32 `json:"updatedNumberScheduled,omitempty"`

	// The number of nodes that should be running the kepler-x pod and have one or
	// more of the kepler-x pod running and available
	// +optional
	NumberAvailable int32 `json:"numberAvailable,omitempty"`

	// The number of nodes that should be running the
	// kepler-x pod and have none of the kepler-x pod running and available
	// +optional
	NumberUnavailable int32 `json:"numberUnavailable,omitempty"`

	// conditions represent the latest available observations of kepler-x
	// +operator-sdk:csv:customresourcedefinitions:type=status,xDescriptors="urn:alm:descriptor:com.tectonic.ui:conditions"
	// +listType=atomic
	Conditions []Condition `json:"conditions"`
}

func (kx KeplerX) Namespace() string {
	return kx.Spec.Deployment.Namespace
}

func (kx KeplerX) DaemonsetName() string {
	return kx.Name
}

func (kx KeplerX) ServiceAccountName() string {
	return kx.Name
}

func (kx KeplerX) FQServiceAccountName() string {
	return "system:serviceaccount:" + kx.Namespace() + ":" + kx.Name
}

//+kubebuilder:object:root=true

// KeplerXList contains a list of KeplerX
type KeplerXList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KeplerX `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KeplerX{}, &KeplerXList{})
}
