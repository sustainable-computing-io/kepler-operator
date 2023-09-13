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
		"app.kubernetes.io/part-of":    "kepler",
	}
)

const (
	Namespace = "openshift-kepler-operator"
)

func NewKeplerNamespace() *corev1.Namespace {
	return &corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Namespace",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: Namespace,
			Labels: CommonLabels.Merge(k8s.StringMap{
				// NOTE: Fixes the following error On Openshift 4.14
				//   Warning  FailedCreate  daemonset-controller
				//   Error creating: pods "kepler-exporter-ds-d6f28" is forbidden:
				//   violates PodSecurity "restricted:latest":
				//   privileged (container "kepler-exporter" must not set securityContext.privileged=true),
				//   allowPrivilegeEscalation != false (container "kepler-exporter" must set
				//   securityContext.allowPrivilegeEscalation=false),
				"pod-security.kubernetes.io/audit":   "privileged",
				"pod-security.kubernetes.io/enforce": "privileged",
				"pod-security.kubernetes.io/warn":    "privileged",
			}),
			//TODO: ensure in-cluster monitoring ignores this ns
		},
	}
}
