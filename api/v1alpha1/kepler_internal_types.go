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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: all internal types can depend on public types
// e.g. kepler-internal.spec.exporter can reuse ExporterSpec because the API is
// considered stable but not vice-versa.

type InternalExporterDeploymentSpec struct {
	ExporterDeploymentSpec `json:",inline"`
	// Image of kepler-exporter to be deployed
	// +kubebuilder:validation:MinLength=3
	Image string `json:"image"`

	// Namespace where kepler-exporter will be deployed
	// +kubebuilder:validation:MinLength=1
	Namespace string `json:"namespace"`
}

type InternalExporterSpec struct {
	// +kubebuilder:validation:Required
	Deployment InternalExporterDeploymentSpec `json:"deployment"`

	Redfish *RedfishSpec `json:"redfish,omitempty"`
}

type DashboardSpec struct {
	// +kubebuilder:default=false
	Enabled bool `json:"enabled,omitempty"`
}

type OpenShiftSpec struct {
	// +kubebuilder:default=true
	Enabled   bool          `json:"enabled"`
	Dashboard DashboardSpec `json:"dashboard,omitempty"`
}

// KeplerInternalSpec defines the desired state of KeplerInternal
type KeplerInternalSpec struct {
	Exporter  InternalExporterSpec `json:"exporter"`
	OpenShift OpenShiftSpec        `json:"openshift,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:scope="Cluster"
//+kubebuilder:subresource:status

// +kubebuilder:printcolumn:name="Port",type=integer,JSONPath=`.spec.exporter.deployment.port`
// +kubebuilder:printcolumn:name="Desired",type=integer,JSONPath=`.status.exporter.desiredNumberScheduled`
// +kubebuilder:printcolumn:name="Current",type=integer,JSONPath=`.status.exporter.currentNumberScheduled`
// +kubebuilder:printcolumn:name="Up-to-date",type=integer,JSONPath=`.status.exporter.updatedNumberScheduled`
// +kubebuilder:printcolumn:name="Ready",type=integer,JSONPath=`.status.exporter.numberReady`
// +kubebuilder:printcolumn:name="Available",type=integer,JSONPath=`.status.exporter.numberAvailable`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="Image",type=string,JSONPath=`.spec.exporter.deployment.image`
// +kubebuilder:printcolumn:name="Node-Selector",type=string,JSONPath=`.spec.exporter.deployment.nodeSelector`,priority=10
// +kubebuilder:printcolumn:name="Tolerations",type=string,JSONPath=`.spec.exporter.deployment.tolerations`,priority=10
//
// KeplerInternal is the Schema for the keplers internal API
type KeplerInternal struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KeplerInternalSpec   `json:"spec,omitempty"`
	Status KeplerInternalStatus `json:"status,omitempty"`
}

type DeploymentStatus string

const (
	DeploymentNotInstalled DeploymentStatus = "NotInstalled"
	DeploymentNotReady     DeploymentStatus = "NotReady"
	DeploymentRunning      DeploymentStatus = "Running"
)

// KeplerInternalStatus represents status of KeplerInternal
type KeplerInternalStatus struct {
	Exporter ExporterStatus `json:"exporter,omitempty"`
}

func (ki KeplerInternal) Namespace() string {
	return ki.Spec.Exporter.Deployment.Namespace
}

func (ki KeplerInternal) DaemonsetName() string {
	return ki.Name
}

func (ki KeplerInternal) ServiceAccountName() string {
	return ki.Name
}

func (ki KeplerInternal) FQServiceAccountName() string {
	return "system:serviceaccount:" + ki.Namespace() + ":" + ki.Name
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
