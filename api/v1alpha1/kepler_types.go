// SPDX-FileCopyrightText: 2025 The Kepler Authors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ExporterDeploymentSpec struct {
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

// RedfishSpec for connecting to Redfish API
type RedfishSpec struct {
	// SecretRef refers to the name of secret which contains credentials to initialize RedfishClient
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=63
	SecretRef string `json:"secretRef"`

	// ProbeInterval controls how frequently power info is queried from Redfish
	// +optional
	// +kubebuilder:default:="60s"
	ProbeInterval metav1.Duration `json:"probeInterval,omitempty"`

	// SkipSSLVerify controls if RedfishClient will skip verifying server
	// +optional
	// +kubebuilder:validation:Required
	// +kubebuilder:default:=false
	SkipSSLVerify bool `json:"skipSSLVerify,omitempty"`
}

type ExporterSpec struct {
	Deployment ExporterDeploymentSpec `json:"deployment,omitempty"`

	// +optional
	// +kubebuilder:validation:optional
	Redfish *RedfishSpec `json:"redfish,omitempty"`
}

// KeplerSpec defines the desired state of Kepler
type KeplerSpec struct {
	Exporter ExporterSpec `json:"exporter,omitempty"`
}

type ConditionType string

const (
	Available  ConditionType = "Available"
	Reconciled ConditionType = "Reconciled"
)

type ConditionReason string

const (
	// ReconcileComplete indicates the CR was successfully reconciled
	ReconcileComplete ConditionReason = "ReconcileSuccess"

	// ReconcileError indicates an error was encountered while reconciling the CR
	ReconcileError ConditionReason = "ReconcileError"

	// InvalidKeplerResource indicates the CR name was invalid
	InvalidKeplerResource ConditionReason = "InvalidKeplerResource"

	// KeplerResourceDeprecated indicates that the Kepler CR is deprecated
	KeplerResourceDeprecated ConditionReason = "KeplerResourceDeprecated"

	// DaemonSetNotFound indicates the DaemonSet created for a kepler was not found
	DaemonSetNotFound           ConditionReason = "DaemonSetNotFound"
	DaemonSetError              ConditionReason = "DaemonSetError"
	DaemonSetInProgess          ConditionReason = "DaemonSetInProgress"
	DaemonSetUnavailable        ConditionReason = "DaemonSetUnavailable"
	DaemonSetPartiallyAvailable ConditionReason = "DaemonSetPartiallyAvailable"
	DaemonSetPodsNotRunning     ConditionReason = "DaemonSetPodsNotRunning"
	DaemonSetRolloutInProgress  ConditionReason = "DaemonSetRolloutInProgress"
	DaemonSetReady              ConditionReason = "DaemonSetReady"
	DaemonSetOutOfSync          ConditionReason = "DaemonSetOutOfSync"
)

// These are valid condition statuses.
// "ConditionTrue" means a resource is in the condition.
// "ConditionFalse" means a resource is not in the condition.
// "ConditionUnknown" means kubernetes can't decide if a resource is in the condition or not.
// In the future, we could add other intermediate conditions, e.g. ConditionDegraded.
type ConditionStatus string

const (
	ConditionTrue     ConditionStatus = "True"
	ConditionFalse    ConditionStatus = "False"
	ConditionUnknown  ConditionStatus = "Unknown"
	ConditionDegraded ConditionStatus = "Degraded"
)

type Condition struct {
	// Type of Kepler Condition - Reconciled, Available ...
	Type ConditionType `json:"type"`
	// status of the condition, one of True, False, Unknown.
	Status ConditionStatus `json:"status"`
	//
	// observedGeneration represents the .metadata.generation that the condition was set based upon.
	// For instance, if .metadata.generation is currently 12, but the .status.conditions[x].observedGeneration is 9, the condition is out of date
	// with respect to the current state of the instance.
	// +optional
	// +kubebuilder:validation:Minimum=0
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
	// lastTransitionTime is the last time the condition transitioned from one status to another.
	// This should be when the underlying condition changed.  If that is not known, then using the time when the API field changed is acceptable.
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Format=date-time
	LastTransitionTime metav1.Time `json:"lastTransitionTime"`
	// reason contains a programmatic identifier indicating the reason for the condition's last transition.
	// +required
	Reason ConditionReason `json:"reason"`
	// message is a human readable message indicating details about the transition.
	// This may be an empty string.
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MaxLength=32768
	Message string `json:"message"`
}

// ExporterStatus defines the observed state of Kepler Exporter
type ExporterStatus struct {
	// The number of nodes that are running at least 1 kepler pod and are
	// supposed to run the kepler pod.
	CurrentNumberScheduled int32 `json:"currentNumberScheduled"`

	// The number of nodes that are running the kepler pod, but are not supposed
	// to run the kepler pod.
	NumberMisscheduled int32 `json:"numberMisscheduled"`

	// The total number of nodes that should be running the kepler
	// pod (including nodes correctly running the kepler pod).
	DesiredNumberScheduled int32 `json:"desiredNumberScheduled"`

	// numberReady is the number of nodes that should be running the kepler pod
	// and have one or more of the kepler pod running with a Ready Condition.
	NumberReady int32 `json:"numberReady"`

	// The total number of nodes that are running updated kepler pod
	// +optional
	UpdatedNumberScheduled int32 `json:"updatedNumberScheduled,omitempty"`

	// The number of nodes that should be running the kepler pod and have one or
	// more of the kepler pod running and available
	// +optional
	NumberAvailable int32 `json:"numberAvailable,omitempty"`

	// The number of nodes that should be running the
	// kepler pod and have none of the kepler pod running and available
	// +optional
	NumberUnavailable int32 `json:"numberUnavailable,omitempty"`

	// conditions represent the latest available observations of the kepler-exporter
	// +operator-sdk:csv:customresourcedefinitions:type=status,xDescriptors="urn:alm:descriptor:com.tectonic.ui:conditions"
	// +listType=atomic
	Conditions []Condition `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:scope="Cluster"
//+kubebuilder:subresource:status

// +kubebuilder:printcolumn:name="Port",type=integer,JSONPath=`.spec.exporter.deployment.port`
// +kubebuilder:printcolumn:name="Desired",type=integer,JSONPath=`.status.exporter.desiredNumberScheduled`
// +kubebuilder:printcolumn:name="Current",type=integer,JSONPath=`.status.exporter.currentNumberScheduled`
// +kubebuilder:printcolumn:name="Ready",type=integer,JSONPath=`.status.exporter.numberReady`
// +kubebuilder:printcolumn:name="Up-to-date",type=integer,JSONPath=`.status.exporter.updatedNumberScheduled`
// +kubebuilder:printcolumn:name="Available",type=integer,JSONPath=`.status.exporter.numberAvailable`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="Node-Selector",type=string,JSONPath=`.spec.exporter.deployment.nodeSelector`,priority=10
// +kubebuilder:printcolumn:name="Tolerations",type=string,JSONPath=`.spec.exporter.deployment.tolerations`,priority=10
//
// Kepler is deprecated and will be removed in a future release. Please use PowerMonitor instead.
type Kepler struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KeplerSpec   `json:"spec,omitempty"`
	Status KeplerStatus `json:"status,omitempty"`
}

// KeplerStatus defines the observed state of Kepler
type KeplerStatus struct {
	Exporter ExporterStatus `json:"exporter,omitempty"`
	// conditions represent the latest available observations of kepler-internal
	// +operator-sdk:csv:customresourcedefinitions:type=status,xDescriptors="urn:alm:descriptor:com.tectonic.ui:conditions"
	// +listType=atomic
	Conditions []Condition `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true

// KeplerList contains a list of Kepler
type KeplerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Kepler `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Kepler{}, &KeplerList{})
}
