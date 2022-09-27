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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// KeplerMachineConfigSpec defines the desired state of KeplerMachineConfig
type KeplerMachineConfigSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of KeplerMachineConfig. Edit keplermachineconfig_types.go to remove/update
	Foo string `json:"foo,omitempty"`
}

// KeplerMachineConfigStatus defines the observed state of KeplerMachineConfig
type KeplerMachineConfigStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// KeplerMachineConfig is the Schema for the keplermachineconfigs API
type KeplerMachineConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KeplerMachineConfigSpec   `json:"spec,omitempty"`
	Status KeplerMachineConfigStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// KeplerMachineConfigList contains a list of KeplerMachineConfig
type KeplerMachineConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KeplerMachineConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KeplerMachineConfig{}, &KeplerMachineConfigList{})
}
