/*
Copyright 2022.

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

// NOTE: json tags are required. Any new fields you add must have json tags for
// the fields to be serialized.

type Cgroupv2 string

type RatioMetrics struct {
	Global string `json:"global,omitempty"`
	Core   string `json:"core,omitempty"`
	Uncore string `json:"uncore,omitempty"`
	Dram   string `json:"dram,omitempty"`
}

type Sources struct {
	Cgroupv2 Cgroupv2 `json:"cgroupv2,omitempty"`
	Bpf      string   `json:"bpf,omitempty"`
	Counters string   `json:"counters,omitempty"`
	Kubelet  string   `json:"kubelet,omitempty"`
}

type HTTPHeader struct {
	Key   string `json:"headerKey,omitempty"`
	Value string `json:"headerValue,omitempty"`
}

type ModelServerTrainerSpec struct {
	// TODO: consider namespacing all Prometheus related fields

	// +kubebuilder:default=20
	PromQueryInterval int `json:"promQueryInterval,omitempty"`

	// +kubebuilder:default=3
	PromQueryStep int `json:"promQueryStep,omitempty"`

	PromHeaders []HTTPHeader `json:"promHeaders,omitempty"`

	// +kubebuilder:default=true
	PromSSLDisable bool `json:"promSSLDisable,omitempty"`

	// +kubebuilder:default=""
	InitialModelsEndpoint string `json:"initialModelsEndpoint,omitempty"`

	// +kubebuilder:default=""
	InitialModelNames string `json:"initialModelNames,omitempty"`
}

type ModelServerSpec struct {

	// +kubebuilder:default=""
	URL string `json:"url,omitempty"`

	// +kubebuilder:default=8100
	Port int `json:"port,omitempty"`

	// +kubebuilder:default=""
	Path string `json:"path,omitempty"`

	// +kubebuilder:default=""
	RequiredPath string `json:"requiredPath,omitempty"`

	// +kubebuilder:default=""
	PromServer string `json:"promServer,omitempty"`

	Trainer *ModelServerTrainerSpec `json:"trainer,omitempty"`
}

type EstimatorSpec struct {
	ModelName        string `json:"modelName,omitempty"`
	FilterConditions string `json:"filterConditions,omitempty"`
	InitUrl          string `json:"initUrl,omitempty"`
}

type ExporterSpec struct {
	// TODO: fix the default version before dev-preview

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
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
}

// KeplerSpec defines the desired state of Kepler
type KeplerSpec struct {
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors="urn:alm:descriptor:com.tectonic.ui:exporterFields"
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
	ConditionTrue    ConditionStatus = "True"
	ConditionFalse   ConditionStatus = "False"
	ConditionUnknown ConditionStatus = "Unknown"
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

// KeplerStatus defines the observed state of Kepler
type KeplerStatus struct {
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

	// conditions represent the latest available observations of the kepler-system

	// +operator-sdk:csv:customresourcedefinitions:type=status,xDescriptors="urn:alm:descriptor:com.tectonic.ui:conditions"
	// +listType=atomic
	Conditions []Condition `json:"conditions"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:scope="Cluster"
//+kubebuilder:subresource:status

// +kubebuilder:printcolumn:name="Port",type=integer,JSONPath=`.spec.exporter.port`
// +kubebuilder:printcolumn:name="Desired",type=integer,JSONPath=`.status.desiredNumberScheduled`
// +kubebuilder:printcolumn:name="Current",type=integer,JSONPath=`.status.currentNumberScheduled`
// +kubebuilder:printcolumn:name="Ready",type=integer,JSONPath=`.status.numberReady`
// +kubebuilder:printcolumn:name="Up-to-date",type=integer,JSONPath=`.status.updatedNumberScheduled`
// +kubebuilder:printcolumn:name="Available",type=integer,JSONPath=`.status.numberAvailable`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="Node-Selector",type=string,JSONPath=`.spec.exporter.nodeSelector`,priority=10
// +kubebuilder:printcolumn:name="Tolerations",type=string,JSONPath=`.spec.exporter.tolerations`,priority=10
//
// Kepler is the Schema for the keplers API
type Kepler struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KeplerSpec   `json:"spec,omitempty"`
	Status KeplerStatus `json:"status,omitempty"`
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
