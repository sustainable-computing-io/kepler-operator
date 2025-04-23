// SPDX-FileCopyrightText: 2025 The Kepler Authors
// SPDX-License-Identifier: Apache-2.0

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
