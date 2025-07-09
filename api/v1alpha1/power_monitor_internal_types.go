// SPDX-FileCopyrightText: 2025 The Kepler Authors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
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
	PowerMonitorKeplerDeploymentSpec `json:",inline"`
	// +kubebuilder:validation:MinLength=3
	Image string `json:"image"`

	// +kubebuilder:validation:MinLength=3
	KubeRbacProxyImage string `json:"kubeRbacProxyImage,omitempty"`

	// +kubebuilder:validation:MinLength=1
	Namespace string `json:"namespace"`
}

type PowerMonitorInternalKeplerConfigSpec struct {
	// +kubebuilder:default="info"
	LogLevel string `json:"logLevel,omitempty"`

	// AdditionalConfigMaps is a list of ConfigMap names that will be merged with the default ConfigMap
	// These AdditionalConfigMaps must exist in the same namespace as PowerMonitor components
	// +optional
	// +listType=atomic
	AdditionalConfigMaps []ConfigMapRef `json:"additionalConfigMaps,omitempty"`
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
}

type PowerMonitorInternalStatus struct {
	Kepler PowerMonitorInternalKeplerStatus `json:"kepler,omitempty"`
	// conditions represent the latest available observations of power-monitor-internal
	// +operator-sdk:csv:customresourcedefinitions:type=status,xDescriptors="urn:alm:descriptor:com.tectonic.ui:conditions"
	// +listType=atomic
	Conditions []Condition `json:"conditions"`
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
