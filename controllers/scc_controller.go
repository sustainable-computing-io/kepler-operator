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

package controllers

import (
	"context"
	"fmt"
	"strings"

	securityv1 "github.com/openshift/api/security/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
)

func (r *collectorReconciler) ensureSCC(l klog.Logger) (bool, error) {

	logger := l.WithValues("securityContextConstraints", types.NamespacedName{Name: "kepler-scc", Namespace: r.Instance.Namespace})

	var labels = make(map[string]string)
	labels["sustainable-computing.io/app"] = "kepler"

	scc := securityv1.SecurityContextConstraints{
		TypeMeta: metav1.TypeMeta{
			Kind:       "SecurityContextConstraints",
			APIVersion: "security.openshift.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			//Name:      "kepler-scc",
			Name:      SCCObjectNameSuffix,
			Labels:    labels,
			Namespace: r.Instance.Namespace + SCCObjectNameSpaceSuffix,
		},
		AllowPrivilegedContainer: true,
		AllowHostDirVolumePlugin: true,
		AllowHostIPC:             true,
		AllowHostNetwork:         true,
		AllowHostPID:             true,
		AllowHostPorts:           true,
		DefaultAddCapabilities:   []corev1.Capability{corev1.Capability("SYS_ADMIN")},
		FSGroup: securityv1.FSGroupStrategyOptions{
			Type: securityv1.FSGroupStrategyRunAsAny,
		},
		ReadOnlyRootFilesystem: true,
		RunAsUser: securityv1.RunAsUserStrategyOptions{
			Type: securityv1.RunAsUserStrategyRunAsAny,
		},
		SELinuxContext: securityv1.SELinuxContextStrategyOptions{
			Type: securityv1.SELinuxStrategyRunAsAny,
		},
		//Users: []string{"kepler", "system:serviceaccount:" + r.Instance.Namespace + ":kepler-sa"},
		Users:   []string{"kepler", "system:serviceaccount:" + r.Instance.Namespace + ":" + r.Instance.Name + ServiceAccountNameSuffix},
		Volumes: []securityv1.FSType{securityv1.FSType("configMap"), securityv1.FSType("projected"), securityv1.FSType("emptyDir"), securityv1.FSType("hostPath")},
	}

	found := &securityv1.SecurityContextConstraints{}

	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: SCCObjectNameSuffix, Namespace: r.Instance.Namespace + SCCObjectNameSpaceSuffix}, found)
	if err != nil {
		if strings.Contains(err.Error(), "no matches for kind") {
			fmt.Printf("resulting error not a timeout: %s", err)
			logger.V(1).Info("Not OpenShift skipping MachineConfig")
			return true, nil
		}
	}
	if err != nil && !apierrors.IsNotFound(err) {
		return false, err

	}
	if apierrors.IsNotFound(err) {
		err = r.Client.Create(context.TODO(), &scc)
		if err != nil {
			return false, err
		}
	}
	logger.V(1).Info("SecurityContextConstraints", "SecurityContextConstraints", scc)

	return true, nil
}

// allowHostDirVolumePlugin: true
// allowHostIPC: true
// allowHostNetwork: true
// allowHostPID: true
// allowHostPorts: true
// allowPrivilegedContainer: true
// apiVersion: security.openshift.io/v1
// defaultAddCapabilities:
// - SYS_ADMIN
// fsGroup:
//   type: RunAsAny
// kind: SecurityContextConstraints
// metadata:
//   labels:
//     sustainable-computing.io/app: kepler
//   name: kepler-scc
//   namespace: kepler
// readOnlyRootFilesystem: true
// runAsUser:
//   type: RunAsAny
// seLinuxContext:
//   type: RunAsAny
// users:
// - kepler
// - system:serviceaccount:kepler:kepler-sa
// volumes:
// - configMap
// - projected
// - emptyDir
// - hostPath
