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

package modelserver

import (
	"fmt"
	"strings"

	"github.com/sustainable.computing.io/kepler-operator/api/v1alpha1"
	"github.com/sustainable.computing.io/kepler-operator/pkg/components"
	"github.com/sustainable.computing.io/kepler-operator/pkg/utils/k8s"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	PVCNameSuffix   = "-pvc"
	ConfigMapSuffix = "-cm"
	ServiceSuffix   = "-svc"
)

const (
	defaultModelServer        = "http://%s.%s.svc.cluster.local:%d"
	StableImage               = "quay.io/sustainable_computing_io/kepler_model_server:v0.7.11-2"
	waitForModelServerCommand = "until [[ \"$(curl -s -o /dev/null -w %%{http_code} %s/best-models)\" -eq 200 ]]; do sleep 1; done"
)

var (
	// common labels for all resources of modelserver
	labels = components.CommonLabels.Merge(k8s.StringMap{
		"app.kubernetes.io/component":  "model-server",
		"sustainable-computing.io/app": "model-server",
	})

	podSelector = labels.Merge(k8s.StringMap{
		"app.kubernetes.io/name": "model-server",
	})
)

var (
	shellCommand = []string{"/usr/bin/bash", "-c"}
)

func NewDeployment(deployName string, ms *v1alpha1.InternalModelServerSpec, namespace string) *appsv1.Deployment {
	pvcName := deployName + PVCNameSuffix
	configMapName := deployName + ConfigMapSuffix
	var storage corev1.Volume
	if ms.Storage.PersistentVolumeClaim == nil {
		storage = k8s.VolumeFromEmptyDir("mnt")
	} else {
		storage = k8s.VolumeFromPVC("mnt", pvcName)
	}
	volumes := []corev1.Volume{
		storage,
		k8s.VolumeFromConfigMap("cfm", configMapName),
		k8s.VolumeFromEmptyDir("resource"),
	}

	mounts := []corev1.VolumeMount{{
		Name:      "cfm",
		MountPath: "/etc/kepler/kepler.config",
		ReadOnly:  true,
	}, {
		Name:      "mnt",
		MountPath: "/mnt",
	}, {
		Name:      "resource",
		MountPath: "/usr/local/lib/python3.10/site-packages/resource",
	}}

	port := ms.Port
	containers := []corev1.Container{{
		Image:           ms.Image,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Name:            "server-api",
		Ports: []corev1.ContainerPort{{
			ContainerPort: int32(port),
			Name:          "http",
		}},
		VolumeMounts: mounts,
		Command:      []string{"model-server"},
		Args:         []string{"-l", "info"},
	}}

	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: appsv1.SchemeGroupVersion.String(),
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      deployName,
			Namespace: namespace,
			Labels:    labels,
		},

		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: podSelector,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: podSelector,
				},
				Spec: corev1.PodSpec{
					Containers: containers,
					Volumes:    volumes,
				},
			},
		},
	}
}

func NewService(deployName string, ms *v1alpha1.InternalModelServerSpec, namespace string) *corev1.Service {
	port := ms.Port
	serviceName := deployName + ServiceSuffix
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: "None",
			Selector:  labels,
			Ports: []corev1.ServicePort{{
				Name:       "http",
				Port:       int32(port),
				TargetPort: intstr.FromString("http"),
			}},
		},
	}
}

func ConfigForClient(deployName, deployNamespace string, ms *v1alpha1.InternalModelServerSpec) k8s.StringMap {
	msConfig := k8s.StringMap{
		"MODEL_SERVER_URL": defaultIfEmpty(ms.URL, serverUrl(deployName, deployNamespace, *ms)),
	}
	msConfig = msConfig.AddIfNotEmpty("MODEL_SERVER_REQ_PATH", ms.RequestPath)
	msConfig = msConfig.AddIfNotEmpty("MODEL_SERVER_MODEL_LIST_PATH", ms.ListPath)

	return msConfig
}

func NewConfigMap(deployName string, d components.Detail, ms *v1alpha1.InternalModelServerSpec, namespace string) *corev1.ConfigMap {
	configMapName := deployName + ConfigMapSuffix
	if d == components.Metadata {
		return &corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				APIVersion: corev1.SchemeGroupVersion.String(),
				Kind:       "ConfigMap",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      configMapName,
				Namespace: namespace,
				Labels:    labels,
			},
		}
	}
	msConfig := k8s.StringMap{
		"MODEL_PATH": defaultIfEmpty(ms.Path, "/mnt/models"),
	}
	msConfig = msConfig.AddIfNotEmpty("MODEL_SERVER_REQ_PATH", ms.RequestPath)
	msConfig = msConfig.AddIfNotEmpty("MODEL_SERVER_MODEL_LIST_PATH", ms.ListPath)
	msConfig = msConfig.AddIfNotEmpty("INITIAL_PIPELINE_URL", ms.PipelineURL)
	msConfig = msConfig.AddIfNotEmpty("ERROR_KEY", ms.ErrorKey)

	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: namespace,
			Labels:    labels,
		},
		Data: msConfig.ToMap(),
	}
}

func NewPVC(deployName string, namespace string, pvcSpec *corev1.PersistentVolumeClaimSpec) *corev1.PersistentVolumeClaim {
	pvcName := deployName + PVCNameSuffix
	return &corev1.PersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "PersistentVolumeClaim",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      pvcName,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: *pvcSpec,
	}
}

func defaultIfEmpty(given, defaultVal string) string {
	if given != "" {
		return given
	}
	return defaultVal
}

func serverUrl(deployName, deployNamespace string, ms v1alpha1.InternalModelServerSpec) string {
	port := ms.Port
	serviceName := deployName + ServiceSuffix
	return fmt.Sprintf(defaultModelServer, serviceName, deployNamespace, port)
}

func formatModelServerCommand(deployName, deployNamespace string, ms v1alpha1.InternalModelServerSpec) string {
	return fmt.Sprintf(waitForModelServerCommand, serverUrl(deployName, deployNamespace, ms))
}

func addModelServerWaitCmd(exporterContainer *corev1.Container, deployName, deployNamespace string, ms v1alpha1.InternalModelServerSpec) *corev1.Container {
	cmd := exporterContainer.Command
	exporterContainer.Command = shellCommand
	exporterContainer.Args = []string{fmt.Sprintf("%s && %s", formatModelServerCommand(deployName, deployNamespace, ms), strings.Join(cmd, " "))}
	return exporterContainer

}

func AddModelServerDependency(exporterContainer *corev1.Container, deployName, deployNamespace string, ms *v1alpha1.InternalModelServerSpec) *corev1.Container {
	exporterContainer = addModelServerWaitCmd(exporterContainer, deployName, deployNamespace, *ms)
	return exporterContainer
}
