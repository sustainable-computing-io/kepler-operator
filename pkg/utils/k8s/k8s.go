package k8s

import (
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
