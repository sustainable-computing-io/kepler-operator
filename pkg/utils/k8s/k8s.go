// SPDX-FileCopyrightText: 2025 The Kepler Authors
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"fmt"

	secv1 "github.com/openshift/api/security/v1"
	"github.com/sustainable.computing.io/kepler-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Cluster int

const (
	Kubernetes Cluster = iota
	OpenShift
)

// ContainerIndex type represents the hard-coded index of Containers in a PodSpec
type ContainerIndex int

type StringMap map[string]string

type SCCAllows struct {
	AllowPrivilegedContainer bool
	AllowHostDirVolumePlugin bool
	AllowHostIPC             bool
	AllowHostNetwork         bool
	AllowHostPID             bool
	AllowHostPorts           bool
}

func (l StringMap) Merge(other StringMap) StringMap {
	ret := StringMap{}

	for k, v := range l {
		ret[k] = v
	}

	for k, v := range other {
		ret[k] = v
	}
	return ret
}

func (l StringMap) ToMap() map[string]string {
	return l
}

func (l StringMap) AddIfNotEmpty(k, v string) StringMap {
	if k != "" && v != "" {
		l[k] = v
	}
	return l
}

func VolumeFromHost(name, path string) corev1.Volume {
	return corev1.Volume{
		Name: name,
		VolumeSource: corev1.VolumeSource{
			HostPath: &corev1.HostPathVolumeSource{Path: path},
		},
	}
}

func VolumeFromConfigMap(name, cmName string) corev1.Volume {
	return corev1.Volume{
		Name: name,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: cmName,
				},
			},
		},
	}
}

func VolumeFromPVC(name, pvcName string) corev1.Volume {
	return corev1.Volume{
		Name: name,
		VolumeSource: corev1.VolumeSource{
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
				ClaimName: pvcName,
			},
		},
	}
}

func VolumeFromEmptyDir(name string) corev1.Volume {
	return corev1.Volume{
		Name: name,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}
}

func EnvFromField(path string) *corev1.EnvVarSource {
	return &corev1.EnvVarSource{
		FieldRef: &corev1.ObjectFieldSelector{FieldPath: path},
	}
}

func EnvFromConfigMap(key, cmName string) *corev1.EnvVarSource {
	return &corev1.EnvVarSource{
		ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
			Key: key,
			LocalObjectReference: corev1.LocalObjectReference{
				Name: cmName,
			},
		},
	}
}

func GVKName(o client.Object) string {
	ns := o.GetNamespace()
	name := o.GetName()
	gvk := o.GetObjectKind().GroupVersionKind().String()
	if ns != "" {
		return fmt.Sprintf("%s (%s)", name, gvk)
	}
	return fmt.Sprintf("%s/%s (%s)", ns, name, gvk)
}

func FindCondition(c []v1alpha1.Condition, t v1alpha1.ConditionType) (v1alpha1.Condition, error) {
	for _, cond := range c {
		if cond.Type == t {
			return cond, nil
		}
	}
	return v1alpha1.Condition{}, fmt.Errorf("condition %s not found", t)
}

func NodeSelectorFromDS(ds *appsv1.DaemonSet) map[string]string {
	return ds.Spec.Template.Spec.NodeSelector
}

func TolerationsFromDS(ds *appsv1.DaemonSet) []corev1.Toleration {
	return ds.Spec.Template.Spec.Tolerations
}

func HostPIDFromDS(ds *appsv1.DaemonSet) bool {
	return ds.Spec.Template.Spec.HostPID
}

func CommandFromDS(ds *appsv1.DaemonSet, index ContainerIndex) []string {
	return ds.Spec.Template.Spec.Containers[index].Command
}

func AnnotationFromDS(ds *appsv1.DaemonSet) map[string]string {
	return ds.Spec.Template.Annotations
}

func VolumeMountsFromDS(ds *appsv1.DaemonSet, index ContainerIndex) []corev1.VolumeMount {
	return ds.Spec.Template.Spec.Containers[index].VolumeMounts
}

func VolumesFromDS(ds *appsv1.DaemonSet) []corev1.Volume {
	return ds.Spec.Template.Spec.Volumes
}

func VolumeFromSecret(name, secretName string) corev1.Volume {
	return corev1.Volume{
		Name: name,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName:  secretName,
				Optional:    pointer.Bool(true),
				DefaultMode: pointer.Int32(420),
			},
		},
	}
}

func AllowsFromSCC(SCC *secv1.SecurityContextConstraints) SCCAllows {
	return SCCAllows{
		AllowPrivilegedContainer: SCC.AllowPrivilegedContainer,
		AllowHostDirVolumePlugin: SCC.AllowHostDirVolumePlugin,
		AllowHostIPC:             SCC.AllowHostIPC,
		AllowHostNetwork:         SCC.AllowHostNetwork,
		AllowHostPID:             SCC.AllowHostPID,
		AllowHostPorts:           SCC.AllowHostPorts,
	}
}
