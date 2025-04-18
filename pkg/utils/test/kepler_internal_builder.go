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

package test

import (
	"github.com/sustainable.computing.io/kepler-operator/api/v1alpha1"
	"github.com/sustainable.computing.io/kepler-operator/pkg/utils/k8s"
)

type InternalBuilder struct{}

func (InternalBuilder) WithNamespace(ns string) func(k *v1alpha1.KeplerInternal) {
	return func(k *v1alpha1.KeplerInternal) {
		k.Spec.Exporter.Deployment.Namespace = ns
	}
}

func (InternalBuilder) WithExporterImage(img string) func(k *v1alpha1.KeplerInternal) {
	return func(k *v1alpha1.KeplerInternal) {
		k.Spec.Exporter.Deployment.Image = img
	}
}

func (InternalBuilder) WithExporterPort(p int) func(k *v1alpha1.KeplerInternal) {
	return func(k *v1alpha1.KeplerInternal) {
		k.Spec.Exporter.Deployment.Port = int32(p)
	}
}

func (InternalBuilder) WithCluster(c k8s.Cluster) func(k *v1alpha1.KeplerInternal) {
	return func(k *v1alpha1.KeplerInternal) {
		k.Spec.OpenShift = v1alpha1.OpenShiftSpec{
			Enabled: c == k8s.OpenShift,
		}
	}
}

func (InternalBuilder) WithRedfish(c k8s.Cluster, secretName string) func(k *v1alpha1.KeplerInternal) {
	return func(k *v1alpha1.KeplerInternal) {
		k.Spec.Exporter.Redfish = &v1alpha1.RedfishSpec{
			SecretRef: secretName,
		}
	}
}
