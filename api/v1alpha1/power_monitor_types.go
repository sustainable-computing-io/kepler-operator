// SPDX-FileCopyrightText: 2025 The Kepler Authors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/sustainable.computing.io/kepler-operator/internal/config"
)

const (
	// InvalidKeplerResource indicates the CR name was invalid
	InvalidPowerMonitorResource ConditionReason = "InvalidPowerMonitorResource"
)

type SecurityMode string

const (
	SecurityModeNone SecurityMode = "none"
	SecurityModeRBAC SecurityMode = "rbac"
)

type PowerMonitorKeplerDeploymentSecuritySpec struct {
	// +kubebuilder:validation:Enum=none;rbac
	Mode SecurityMode `json:"mode,omitempty"`
	// +optional
	// +listType=atomic
	AllowedSANames []string `json:"allowedSANames,omitempty"`
}

// MetricsLevelDefault represents the default metric levels for PowerMonitor (node, pod, and vm)
const MetricsLevelDefault = config.MetricsLevelNode | config.MetricsLevelPod | config.MetricsLevelVM

type PowerMonitorKeplerDeploymentSpec struct {
	// Defines which Nodes the Pod is scheduled on
	// +optional
	// +kubebuilder:default={"kubernetes.io/os":"linux"}
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// If specified, define Pod's tolerations
	// +optional
	// +kubebuilder:default={{"key": "", "operator": "Exists", "value": "", "effect": ""}}
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// If set, defines the security mode and allowed SANames
	Security PowerMonitorKeplerDeploymentSecuritySpec `json:"security,omitempty"`
}

type PowerMonitorKeplerConfigSpec struct {
	// +kubebuilder:default="info"
	// +optional
	LogLevel string `json:"logLevel,omitempty"`

	// AdditionalConfigMaps is a list of ConfigMap names that will be merged with the default ConfigMap
	// These AdditionalConfigMaps must exist in the same namespace as PowerMonitor components
	// +optional
	// +listType=atomic
	AdditionalConfigMaps []ConfigMapRef `json:"additionalConfigMaps,omitempty"`

	// MetricLevels specifies which metrics levels to export
	// Valid values are combinations of: node, process, container, vm, pod
	// +optional
	// +listType=set
	// +kubebuilder:default={"node","pod","vm"}
	// +kubebuilder:validation:items:Enum=node;process;container;vm;pod
	MetricLevels []string `json:"metricLevels,omitempty"`

	// Staleness specifies how long to wait before considering calculated power values as stale
	// Must be a positive duration (e.g., "500ms", "5s", "1h"). Negative values are not allowed.
	// +optional
	// +kubebuilder:default="500ms"
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Pattern="^[0-9]+(\\.[0-9]+)?(ns|us|ms|s|m|h)$"
	Staleness *metav1.Duration `json:"staleness,omitempty"`

	// SampleRate specifies the interval for monitoring resources (processes, containers, vms, etc.)
	// Must be a positive duration (e.g., "5s", "1m", "30s"). Negative values are not allowed.
	// +optional
	// +kubebuilder:default="5s"
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Pattern="^[0-9]+(\\.[0-9]+)?(ns|us|ms|s|m|h)$"
	SampleRate *metav1.Duration `json:"sampleRate,omitempty"`

	// MaxTerminated controls terminated workload tracking behavior
	// Negative values: track unlimited terminated workloads (no capacity limit)
	// Zero: disable terminated workload tracking completely
	// Positive values: track top N terminated workloads by energy consumption
	// +optional
	// +kubebuilder:default=500
	MaxTerminated *int32 `json:"maxTerminated,omitempty"`
}

// ConfigMapRef defines a reference to a ConfigMap
type ConfigMapRef struct {
	// Name of the ConfigMap
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
}

type PowerMonitorKeplerSpec struct {
	Deployment PowerMonitorKeplerDeploymentSpec `json:"deployment,omitempty"`
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
}

// PowerMonitorStatus defines the observed state of Power Monitor
type PowerMonitorStatus struct {
	Kepler PowerMonitorKeplerStatus `json:"kepler,omitempty"`
	// conditions represent the latest available observations of power-monitor
	// +operator-sdk:csv:customresourcedefinitions:type=status,xDescriptors="urn:alm:descriptor:com.tectonic.ui:conditions"
	// +listType=atomic
	Conditions []Condition `json:"conditions"`
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
