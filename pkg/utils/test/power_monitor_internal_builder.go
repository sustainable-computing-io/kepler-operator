// SPDX-FileCopyrightText: 2025 The Kepler Authors
// SPDX-License-Identifier: Apache-2.0

package test

import (
	"github.com/sustainable.computing.io/kepler-operator/api/v1alpha1"
	"github.com/sustainable.computing.io/kepler-operator/pkg/utils/k8s"
)

type PowerMonitorInternalBuilder struct{}

func (PowerMonitorInternalBuilder) WithNamespace(ns string) func(pmi *v1alpha1.PowerMonitorInternal) {
	return func(pmi *v1alpha1.PowerMonitorInternal) {
		pmi.Spec.Kepler.Deployment.Namespace = ns
	}
}

func (PowerMonitorInternalBuilder) WithKeplerImage(img string) func(pmi *v1alpha1.PowerMonitorInternal) {
	return func(pmi *v1alpha1.PowerMonitorInternal) {
		// k.Spec.Exporter.Deployment.Image = img
		pmi.Spec.Kepler.Deployment.Image = img
	}
}

func (PowerMonitorInternalBuilder) WithCluster(c k8s.Cluster) func(pmi *v1alpha1.PowerMonitorInternal) {
	return func(pmi *v1alpha1.PowerMonitorInternal) {
		pmi.Spec.OpenShift = v1alpha1.PowerMonitorInternalOpenShiftSpec{
			Enabled: c == k8s.OpenShift,
			Dashboard: v1alpha1.PowerMonitorInternalDashboardSpec{
				Enabled: c == k8s.OpenShift,
			},
		}
	}
}

func (PowerMonitorInternalBuilder) WithAdditionalConfigMaps(configMapNames []string) func(pmi *v1alpha1.PowerMonitorInternal) {
	return func(pmi *v1alpha1.PowerMonitorInternal) {
		var configMapRefs []v1alpha1.ConfigMapRef
		for _, name := range configMapNames {
			configMapRefs = append(configMapRefs, v1alpha1.ConfigMapRef{Name: name})
		}
		pmi.Spec.Kepler.Config.AdditionalConfigMaps = configMapRefs
	}
}
