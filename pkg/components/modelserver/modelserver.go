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
	"strconv"
	"strings"

	"github.com/sustainable.computing.io/kepler-operator/pkg/api/v1alpha1"
	"github.com/sustainable.computing.io/kepler-operator/pkg/components"
	"github.com/sustainable.computing.io/kepler-operator/pkg/utils/k8s"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	resource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	prefix = "model-server-"
	// exported
	PVCName        = prefix + "pvc"
	ConfigmapName  = prefix + "cm"
	ServiceName    = prefix + "svc"
	DeploymentName = prefix + "deploy"
)

const (
	defaultPromInterval = 20
	defaultPromStep     = 3
	defaultPromServer   = "http://prometheus-k8s." + components.Namespace + ".svc.cluster.local:9090/"
	defaultModelServer  = "http://" + ServiceName + "." + components.Namespace + ".svc.cluster.local:%d"
	StableImage         = "quay.io/sustainable_computing_io/kepler_model_server:v0.6"
)

// Config that will be set from outside
var (
	Config = struct {
		Image string
	}{
		Image: StableImage,
	}
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

func NewDeployment(ms *v1alpha1.ModelServerSpec) *appsv1.Deployment {
	volumes := []corev1.Volume{
		k8s.VolumeFromPVC("mnt", PVCName),
		k8s.VolumeFromConfigMap("cfm", ConfigmapName),
	}

	mounts := []corev1.VolumeMount{{
		Name:      "cfm",
		MountPath: "/etc/kepler/kepler.config",
		ReadOnly:  true,
	}, {
		Name:      "mnt",
		MountPath: "/mnt",
	}}

	port := ms.Port
	containers := []corev1.Container{{
		Image:           Config.Image,
		ImagePullPolicy: corev1.PullAlways,
		Name:            "server-api",
		Ports: []corev1.ContainerPort{{
			ContainerPort: int32(port),
			Name:          "http",
		}},
		VolumeMounts: mounts,
		Command:      []string{"python3.8"},
		Args:         []string{"-u", "src/server/model_server.py"},
	}}

	if ms.Trainer != nil {
		containers = append(containers, corev1.Container{
			Image:           Config.Image,
			ImagePullPolicy: corev1.PullIfNotPresent,
			Name:            "online-trainer",
			VolumeMounts:    mounts,
			Command:         []string{"python3.8"},
			Args:            []string{"-u", "src/train/online_trainer.py"},
		})
	}

	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: appsv1.SchemeGroupVersion.String(),
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      DeploymentName,
			Namespace: components.Namespace,
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

func NewService(ms *v1alpha1.ModelServerSpec) *corev1.Service {
	port := ms.Port
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      ServiceName,
			Namespace: components.Namespace,
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

func ModelServerConfigForClient(ms *v1alpha1.ModelServerSpec) k8s.StringMap {
	msConfig := k8s.StringMap{
		"MODEL_SERVER_URL": defaultIfEmpty(ms.URL, serverUrl(*ms)),
	}
	msConfig = msConfig.AddIfNotEmpty("MODEL_SERVER_REQ_PATH", ms.RequestPath)
	msConfig = msConfig.AddIfNotEmpty("MODEL_SERVER_MODEL_LIST_PATH", ms.ListPath)

	return msConfig
}

func NewConfigMap(d components.Detail, ms *v1alpha1.ModelServerSpec) *corev1.ConfigMap {
	if d == components.Metadata {
		return &corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				APIVersion: corev1.SchemeGroupVersion.String(),
				Kind:       "ConfigMap",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      ConfigmapName,
				Namespace: components.Namespace,
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

	if ms.Trainer != nil {
		trainerSettings := settingsForTrainer(ms.Trainer)
		msConfig = msConfig.Merge(trainerSettings)
	}

	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      ConfigmapName,
			Namespace: components.Namespace,
			Labels:    labels,
		},
		Data: msConfig.ToMap(),
	}
}

func settingsForTrainer(trainer *v1alpha1.ModelServerTrainerSpec) k8s.StringMap {
	prom := trainer.Prom
	if prom == nil {
		return k8s.StringMap{}
	}

	promHeaders := strings.Builder{}
	interval := defaultPromInterval
	step := defaultPromStep
	// TODO: ensure trailing , is accepted
	// iterate through available headers and append to promHeaders (this string will be converted to dict in the image)
	for _, h := range prom.Headers {
		promHeaders.WriteString(h.Key)
		promHeaders.WriteString(":")
		promHeaders.WriteString(h.Value)
		promHeaders.WriteString(",")
	}
	if prom.QueryInterval > 0 {
		interval = prom.QueryInterval
	}

	if prom.QueryStep > 0 {
		step = prom.QueryStep
	}
	msConfig := k8s.StringMap{
		"PROM_SERVER":         defaultIfEmpty(prom.Server, defaultPromServer),
		"PROM_SSL_DISABLE":    strconv.FormatBool(prom.SSLDisable),
		"PROM_QUERY_INTERVAL": strconv.Itoa(interval),
		"PROM_QUERY_STEP":     strconv.Itoa(step),
	}

	msConfig = msConfig.AddIfNotEmpty("PROM_HEADERS", promHeaders.String())

	return msConfig
}

func NewPVC() *corev1.PersistentVolumeClaim {
	return &corev1.PersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "PersistentVolumeClaim",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      PVCName,
			Namespace: components.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceName(corev1.ResourceStorage): resource.MustParse("5Gi"),
				},
			},
		},
	}
}

func defaultIfEmpty(given, defaultVal string) string {
	if given != "" {
		return given
	}
	return defaultVal
}

func serverUrl(ms v1alpha1.ModelServerSpec) string {
	port := ms.Port
	return fmt.Sprintf(defaultModelServer, port)
}
