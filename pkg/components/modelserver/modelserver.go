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
	PVCName            = prefix + "pvc"
	ConfigmapName      = prefix + "cm"
	ServiceName        = prefix + "svc"
	DeploymentName     = prefix + "deploy"
	ServiceAccountName = prefix + "sa"
)

const (
	defaultModelURL = "https://raw.githubusercontent.com/sustainable-computing-io/kepler-model-server/main/tests/test_models"
	defaultModels   = `
	AbsComponentModelWeight=KerasCompWeightFullPipeline
	AbsComponentPower=KerasCompFullPipeline
	DynComponentPower=ScikitMixed
	`

	defaultPromServer  = "http://prometheus-k8s." + components.Namespace + ".svc.cluster.local:9090/"
	defaultModelServer = "http://kepler-model-server." + components.Namespace + ".cluster.local:%d"

	image = "quay.io/sustainable_computing_io/kepler_model_server:latest"
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

func NewServiceAccount() *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      ServiceAccountName,
			Namespace: components.Namespace,
			Labels:    labels,
		},
	}
}

func NewDeployment(k *v1alpha1.Kepler) *appsv1.Deployment {

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

	// exporter will always be active
	exporterPort := int32(k.Spec.Exporter.Port)
	ms := k.Spec.ModelServer

	containers := []corev1.Container{{
		Image:           image,
		ImagePullPolicy: corev1.PullAlways,
		Name:            "model-server-api",
		Ports: []corev1.ContainerPort{{
			ContainerPort: exporterPort,
			Name:          "http",
		}},
		VolumeMounts: mounts,
		Command:      []string{"python3.8", "model_server.py"},
	}}

	if ms.Trainer != nil {
		containers = append(containers, corev1.Container{
			Image:           image,
			ImagePullPolicy: corev1.PullIfNotPresent,
			Name:            "online-trainer",
			VolumeMounts:    mounts,
			Command:         []string{"python3.8", "online_trainer.py"},
		})
	}

	return &appsv1.Deployment{
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

func NewService(k *v1alpha1.Kepler) *corev1.Service {
	exporterPort := int32(k.Spec.Exporter.Port)

	return &corev1.Service{
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
				Port:       exporterPort,
				TargetPort: intstr.FromString("http"),
			}},
		},
	}
}

func NewConfigMap(d components.Detail, k *v1alpha1.Kepler) *corev1.ConfigMap {
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

	ms := k.Spec.ModelServer

	msConfig := k8s.StringMap{
		"MODEL_SERVER_ENABLE": "true",
		"PROM_SERVER":         defaultIfEmpty(ms.PromServer, defaultPromServer),

		"MODEL_SERVER_URL":      defaultIfEmpty(ms.URL, fmt.Sprintf(defaultModelServer, ms.Port)),
		"MODEL_PATH":            defaultIfEmpty(ms.Path, "models"),
		"MODEL_SERVER_PORT":     strconv.Itoa(ms.Port),
		"MODEL_SERVER_REQ_PATH": defaultIfEmpty(ms.RequiredPath, "/model"),

		"MNT_PATH": "/mnt",
	}

	trainerSettings := settingsForTrainer(ms.Trainer)

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
		Data: msConfig.Merge(trainerSettings).ToMap(),
	}
}

func settingsForTrainer(trainer *v1alpha1.ModelServerTrainerSpec) k8s.StringMap {
	if trainer == nil {
		return nil
	}

	// iterate through available headers and append to promHeaders (this string will be converted to dict in the image)
	promHeaders := strings.Builder{}
	for _, h := range trainer.PromHeaders {
		promHeaders.WriteString(h.Key)
		promHeaders.WriteString(":")
		promHeaders.WriteString(h.Value)
		promHeaders.WriteString(",")
	}

	return k8s.StringMap{
		"PROM_SSL_DISABLE":    strconv.FormatBool(trainer.PromSSLDisable),
		"PROM_HEADERS":        promHeaders.String(),
		"PROM_QUERY_INTERVAL": strconv.Itoa(trainer.PromQueryInterval),
		"PROM_QUERY_STEP":     strconv.Itoa(trainer.PromQueryStep),
		"INITIAL_MODELS_LOC":  defaultIfEmpty(trainer.InitialModelsEndpoint, defaultModelURL),
		"INITIAL_MODEL_NAMES": defaultIfEmpty(trainer.InitialModelNames, defaultModels),
	}
}

func NewPVC(k *v1alpha1.Kepler) *corev1.PersistentVolumeClaim {
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
