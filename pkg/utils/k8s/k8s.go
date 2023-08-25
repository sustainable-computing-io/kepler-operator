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

package k8s

import (
	"github.com/sustainable.computing.io/kepler-operator/pkg/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

type StringMap map[string]string

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

func EnvFromField(path string) *corev1.EnvVarSource {
	return &corev1.EnvVarSource{
		FieldRef: &corev1.ObjectFieldSelector{FieldPath: path},
	}
}

func FindCondition(c []v1alpha1.Condition, t v1alpha1.ConditionType) (v1alpha1.Condition, error) {
	for _, cond := range c {
		if cond.Type == t {
			return cond, nil
		}
	}
	return v1alpha1.Condition{}, fmt.Errorf("condition %s not found", t)
}
