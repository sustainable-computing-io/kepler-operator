// SPDX-FileCopyrightText: 2025 The Kepler Authors
// SPDX-License-Identifier: Apache-2.0

package components

import (
	"github.com/sustainable.computing.io/kepler-operator/pkg/utils/k8s"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Detail int

const (
	Full     Detail = iota
	Metadata Detail = iota
)

var (
	CommonLabels = k8s.StringMap{
		"app.kubernetes.io/managed-by": "kepler-operator",
	}
)

func NewNamespace(ns string) *corev1.Namespace {
	return &corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Namespace",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: ns,
			Labels: CommonLabels.Merge(k8s.StringMap{
				// NOTE: Fixes the following error On Openshift 4.14
				//   Warning  FailedCreate  daemonset-controller
				//   Error creating: pods "kepler-exporter-ds-d6f28" is forbidden:
				//   violates PodSecurity "restricted:latest":
				//   privileged (container "kepler-exporter" must not set securityContext.privileged=true),
				//   allowPrivilegeEscalation != false (container "kepler-exporter" must set
				//   securityContext.allowPrivilegeEscalation=false),
				"pod-security.kubernetes.io/enforce": "privileged",
			}),
			//TODO: ensure in-cluster monitoring ignores this ns
		},
	}
}
