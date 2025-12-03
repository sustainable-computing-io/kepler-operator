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
				// NOTE: Kepler requires hostPID and hostPath volumes for /proc and /sys access
				// Only the "privileged" PSA level allows these host-level features
				// However, the Kepler container itself does NOT run as privileged (privileged: false)
				"pod-security.kubernetes.io/enforce": "privileged",
			}),
			//TODO: ensure in-cluster monitoring ignores this ns
		},
	}
}
