/*
Copyright 2023.

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

// NOTE: all internal types can depend on public types
// e.g. kepler-internal.spec.exporter can reuse ExporterSpec because the API is
// considered stable but not vice-versa.

type InternalExporterDeploymentSpec struct {
	ExporterDeploymentSpec `json:",inline"`
	Image                  string `json:"image,omitempty"`
}

type InternalExporterSpec struct {
	Deployment InternalExporterDeploymentSpec `json:"deployment,omitempty"`
}

// KeplerInternalSpec defines the desired state of Kepler
type KeplerInternalSpec struct {
	Exporter InternalExporterSpec `json:"exporter,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:scope="Cluster"
//+kubebuilder:subresource:status

// +kubebuilder:printcolumn:name="Port",type=integer,JSONPath=`.spec.exporter.deployment.port`
// +kubebuilder:printcolumn:name="Desired",type=integer,JSONPath=`.status.desiredNumberScheduled`
// +kubebuilder:printcolumn:name="Current",type=integer,JSONPath=`.status.currentNumberScheduled`
// +kubebuilder:printcolumn:name="Ready",type=integer,JSONPath=`.status.numberReady`
// +kubebuilder:printcolumn:name="Up-to-date",type=integer,JSONPath=`.status.updatedNumberScheduled`
// +kubebuilder:printcolumn:name="Available",type=integer,JSONPath=`.status.numberAvailable`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="Image",type=string,JSONPath=`.spec.exporter.deployment.image`
// +kubebuilder:printcolumn:name="Node-Selector",type=string,JSONPath=`.spec.exporter.deployment.nodeSelector`,priority=10
// +kubebuilder:printcolumn:name="Tolerations",type=string,JSONPath=`.spec.exporter.deployment.tolerations`,priority=10
//
// KeplerInternal is the Schema for the keplers internal API
type KeplerInternal struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KeplerInternalSpec `json:"spec,omitempty"`
	Status KeplerStatus       `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// KeplerInternalList contains a list of Kepler
type KeplerInternalList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KeplerInternal `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KeplerInternal{}, &KeplerInternalList{})
}
