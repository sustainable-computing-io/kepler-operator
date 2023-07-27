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

	// +kubebuilder:default="latest"
	Version string `json:"version,omitempty"`
	// +kubebuilder:default=9103
	Port int `json:"port,omitempty"`
}

// KeplerSpec defines the desired state of Kepler
type KeplerSpec struct {
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors="urn:alm:descriptor:com.tectonic.ui:exporterFields"
	Exporter ExporterSpec `json:"exporter,omitempty"`
}

// KeplerStatus defines the observed state of Kepler
type KeplerStatus struct {
	// conditions represent the latest available observations of the kepler-system

	// +operator-sdk:csv:customresourcedefinitions:type=status,xDescriptors="urn:alm:descriptor:com.tectonic.ui:conditions"
	// +listType=atomic
	Conditions []metav1.Condition `json:"conditions"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:scope="Cluster"
//+kubebuilder:subresource:status

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
