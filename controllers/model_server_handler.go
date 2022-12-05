package controllers

import (
	"context"
	"strconv"

	keplerv1alpha1 "github.com/sustainable.computing.io/kepler-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	resource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	PersistentVolumeName      = "kepler-model-server-pv"
	PersistentVolumeClaimName = "kepler-model-server-pvc"
	ConfigMapName             = "kepler-model-server-cfm"
	ModelServerServiceName    = "kepler-model-server"
	ModelServerDeploymentName = "kepler-model-server"
)

type ModelServerDeployment struct {
	Context               context.Context
	Instance              *keplerv1alpha1.Kepler
	Client                client.Client
	Scheme                *runtime.Scheme
	Image                 string
	ConfigMap             *corev1.ConfigMap
	Deployment            *appsv1.Deployment
	Service               *corev1.Service
	PersistentVolume      *corev1.PersistentVolume
	PersistentVolumeClaim *corev1.PersistentVolumeClaim
}

func (msd *ModelServerDeployment) Reconcile(l klog.Logger) (bool, error) {
	return reconcileBatch(l,
		msd.ensureModelServerPersistentVolume,
		msd.ensureModelServerPersistentVolumeClaim,
		//msd.ensureModelServerConfigMap,
		//msd.ensureModelServerService,
		//msd.ensureModelServerDeployment,
	)
}

func (msd *ModelServerDeployment) buildModelServerConfigMap() corev1.ConfigMap {
	dataPairing := make(map[string]string)
	modelServerExporter := msd.Instance.Spec.ModelServerExporter
	modelServerTrainer := msd.Instance.Spec.ModelServerTrainer
	dataPairing["MNT_PATH"] = "/mnt"
	if modelServerExporter != nil {
		dataPairing["PROM_SERVER"] = modelServerExporter.PromServer
		dataPairing["MODEL_PATH"] = modelServerExporter.ModelPath
		dataPairing["MODEL_SERVER_PORT"] = strconv.Itoa(modelServerExporter.Port)
	}
	if modelServerTrainer != nil {
		customHeaders := ""
		// iterate through available headers and append to customHeaders (this string will be converted to dict in the image)
		for _, customHeader := range modelServerTrainer.PromHeaders {
			customHeaders = customHeaders + customHeader.HeaderKey + ":" + customHeader.HeaderValue + ","
		}

		dataPairing["PROM_QUERY_INTERVAL"] = strconv.Itoa(modelServerTrainer.PromQueryInterval)
		dataPairing["PROM_QUERY_STEP"] = strconv.Itoa(modelServerTrainer.PromQueryStep)
		dataPairing["PROM_SSL_DISABLE"] = strconv.FormatBool(modelServerTrainer.PromSSLDisable)
		dataPairing["PROM_HEADERS"] = customHeaders
		dataPairing["INITIAL_MODELS_LOC"] = modelServerTrainer.InitialModelsEndpoint
		initialModelNames := modelServerTrainer.InitialModelNames
		if initialModelNames == "" {
			initialModelNames = `AbsComponentModelWeight=KerasCompWeightFullPipeline
			AbsComponentPower=KerasCompFullPipeline
			DynComponentPower=ScikitMixed`
		}
		dataPairing["INITIAL_MODEL_NAMES"] = initialModelNames

	}

	configMap := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ConfigMapName,
			Namespace: msd.Instance.Namespace,
		},
		Data: dataPairing,
	}
	return configMap
}

func (msd *ModelServerDeployment) buildModelServerPVC() corev1.PersistentVolumeClaim {
	storageClassName := "default"
	modelServerPVC := corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      PersistentVolumeClaimName,
			Namespace: msd.Instance.Namespace,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			StorageClassName: &storageClassName,
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteMany,
			},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceName(corev1.ResourceStorage): resource.MustParse("5Gi"),
				},
			},
		},
	}
	return modelServerPVC
}

func (msd *ModelServerDeployment) buildModelServerPV() corev1.PersistentVolume {
	labels := map[string]string{
		"type":                        "local",
		"app.kubernetes.io/component": PersistentVolumeName,
		"app.kubernetes.io/name":      PersistentVolumeName,
	}
	modelServerPV := corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:   PersistentVolumeName,
			Labels: labels,
		},
		Spec: corev1.PersistentVolumeSpec{
			StorageClassName: "default",
			Capacity: corev1.ResourceList{
				corev1.ResourceName(corev1.ResourceStorage): resource.MustParse("5Gi"),
			},
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteMany,
			},
			PersistentVolumeSource: corev1.PersistentVolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/mnt/data",
				},
			},
		},
	}
	return modelServerPV
}

func (msd *ModelServerDeployment) buildModelServerService() corev1.Service {
	labels := map[string]string{
		"app.kubernetes.io/component": "model-server",
		"app.kubernetes.io/name":      "kepler-model-server",
	}
	modelServerService := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ModelServerServiceName,
			Namespace: msd.Instance.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: "None",
			Selector:  labels,
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       int32(msd.Instance.Spec.ModelServerExporter.Port),
					TargetPort: intstr.FromString("http"),
				},
			},
		},
	}
	return modelServerService
}

func (msd *ModelServerDeployment) buildModelServerDeployment() appsv1.Deployment {
	labels := map[string]string{
		"app.kubernetes.io/component": "model-server",
		"app.kubernetes.io/name":      "kepler-model-server",
	}
	volumes := []corev1.Volume{
		{
			Name: "mnt",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: msd.PersistentVolumeClaim.Name,
				},
			},
		},
		{
			Name: "cfm",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: msd.ConfigMap.Name,
					},
				},
			},
		},
	}

	correspondingVolumeMounts := []corev1.VolumeMount{
		{
			Name:      "cfm",
			MountPath: "/etc/config",
			ReadOnly:  true,
		},
		{
			Name:      "mnt",
			MountPath: "/mnt",
		},
	}

	// exporter or trainer will be active
	modelServerContainers := make([]corev1.Container, 0)
	if msd.Instance.Spec.ModelServerExporter != nil {
		modelServerContainers = append(modelServerContainers, corev1.Container{
			Image:           msd.Image,
			ImagePullPolicy: corev1.PullAlways,
			Name:            "model-server-api",
			Ports: []corev1.ContainerPort{{
				ContainerPort: int32(msd.Instance.Spec.ModelServerExporter.Port),
				Name:          "http",
			}},
			VolumeMounts: correspondingVolumeMounts,
			Command:      []string{"python3", "model_server.py"},
		})
	}

	if msd.Instance.Spec.ModelServerTrainer != nil {
		modelServerContainers = append(modelServerContainers, corev1.Container{
			Image:           msd.Image,
			ImagePullPolicy: corev1.PullIfNotPresent,
			Name:            "online-trainer",
			VolumeMounts:    correspondingVolumeMounts,
			Command:         []string{"python3", "online_trainer.py"},
		})
	}

	deployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ModelServerDeploymentName,
			Namespace: msd.Instance.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: modelServerContainers,
					Volumes:    volumes,
				},
			},
		},
	}

	return deployment

}

func (msd *ModelServerDeployment) ensureModelServerPersistentVolumeClaim(l klog.Logger) (bool, error) { //(ReconciliationResult, error) {
	newMSPVC := msd.buildModelServerPVC()
	msd.PersistentVolumeClaim = &corev1.PersistentVolumeClaim{
		ObjectMeta: newMSPVC.ObjectMeta,
		Spec:       newMSPVC.Spec,
	}
	msPVCResult := &corev1.PersistentVolumeClaim{}
	logger := l.WithValues("PVC", nameFor(msd.PersistentVolumeClaim))
	err := msd.Client.Get(msd.Context, types.NamespacedName{Name: PersistentVolumeClaimName, Namespace: msd.Instance.Namespace}, msPVCResult)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("PVC does not exist. Creating...")
			err = msd.Client.Create(msd.Context, msd.PersistentVolumeClaim)
			if err != nil {
				logger.Error(err, "failed to create PVC")
				return false, err
			}
		} else {
			logger.Error(err, "error not related to missing PVC")
			return false, err
		}
	} else {
		logger.Info("PVC already exists")
	}

	logger.Info("PVC reconciled")
	return true, nil
}

func (msd *ModelServerDeployment) ensureModelServerPersistentVolume(l klog.Logger) (bool, error) { //(ReconciliationResult, error) {
	newMSPV := msd.buildModelServerPV()
	msd.PersistentVolume = &corev1.PersistentVolume{
		ObjectMeta: newMSPV.ObjectMeta,
		Spec:       newMSPV.Spec,
	}
	msPVResult := &corev1.PersistentVolume{}
	logger := l.WithValues("PV", nameFor(msd.PersistentVolume))
	err := msd.Client.Get(msd.Context, types.NamespacedName{Name: PersistentVolumeName}, msPVResult)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("PV does not exist. Creating...")
			err = msd.Client.Create(msd.Context, msd.PersistentVolume)
			if err != nil {
				logger.Error(err, "failed to create PV")
				return false, err
			}
		} else {
			logger.Error(err, "error not related to missing PV")
			return false, err
		}
	} else {
		logger.Info("PV already exists")
	}
	logger.Info("PV reconciled")
	return true, nil
}

/*
func (msd *ModelServerDeployment) ensureModelServerConfigMap(l klog.Logger) (bool, error) { //(ReconciliationResult, error) {
	msd.buildModelServerConfigMap()
	msCFM := msd.ConfigMap
	msCFMResult := &corev1.ConfigMap{}
	logger := l.WithValues("ConfigMap", nameFor(msCFM))
	err := msd.Client.Get(msd.Context, types.NamespacedName{Name: ConfigMapName, Namespace: msd.Instance.Namespace}, msCFMResult)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("ConfigMap does not exist. Creating...")
			err = ctrl.SetControllerReference(msd.Instance, msCFM, msd.Scheme)
			if err != nil {
				logger.Error(err, "failed to set controller reference")
				return false, err
			}
			err = msd.Client.Create(msd.Context, msCFM)
			if err != nil {
				logger.Error(err, "failed to create configmap")
				return false, err
			}
		} else {
			logger.Error(err, "error not related to missing configmap")
			return false, err
		}
	} else if !reflect.DeepEqual(msCFM, msCFMResult) {
		logger.Info("ConfigMap found. Updating...")
		err = ctrl.SetControllerReference(msd.Instance, msCFM, msd.Scheme)
		if err != nil {
			logger.Error(err, "failed to set controller reference")
			return false, err
		}
		err = msd.Client.Update(msd.Context, msCFM)
		if err != nil {
			logger.Error(err, "failed to update configmap")
			return false, err
		}
	}
	logger.Info("ConfigMap reconciled")
	return true, nil

}

func (msd *ModelServerDeployment) ensureModelServerService(l klog.Logger) (bool, error) { //(ReconciliationResult, error) {
	msd.buildModelServerService()
	msService := msd.Service
	msServiceResult := &corev1.Service{}
	logger := l.WithValues("ModelServerService", nameFor(msService))
	err := msd.Client.Get(msd.Context, types.NamespacedName{Name: ModelServerServiceName, Namespace: msd.Instance.Namespace}, msServiceResult)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("Service does not exist. Creating...")
			err = ctrl.SetControllerReference(msd.Instance, msService, msd.Scheme)
			if err != nil {
				logger.Error(err, "failed to set controller reference")
				return false, err
			}
			err = msd.Client.Create(msd.Context, msService)
			if err != nil {
				logger.Error(err, "failed to create service")
				return false, err
			}
		} else {
			logger.Error(err, "error not related to missing service")
			return false, err
		}
	} else if !reflect.DeepEqual(msService, msServiceResult) {
		logger.Info("Service found. Updating...")
		err = ctrl.SetControllerReference(msd.Instance, msService, msd.Scheme)
		if err != nil {
			logger.Error(err, "failed to set controller reference")
			return false, err
		}
		err = msd.Client.Update(msd.Context, msService)
		if err != nil {
			logger.Error(err, "failed to update service")
			return false, err
		}
	}
	logger.Info("Service Reconciled")
	return true, nil

}

func (msd *ModelServerDeployment) ensureModelServerDeployment(l klog.Logger) (bool, error) {
	msd.buildModelServerDeployment()
	msDeployment := msd.Deployment
	msDeploymentResult := &appsv1.Deployment{}
	logger := l.WithValues("ModelServerDeployment", nameFor(msDeployment))
	err := msd.Client.Get(msd.Context, types.NamespacedName{Name: ModelServerDeploymentName, Namespace: msd.Instance.Namespace}, msDeploymentResult)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("Deployment does not exist. Creating...")
			err = ctrl.SetControllerReference(msd.Instance, msDeployment, msd.Scheme)
			if err != nil {
				logger.Error(err, "failed to set controller reference")
				return false, err
			}
			err = msd.Client.Create(msd.Context, msDeployment)
			if err != nil {
				logger.Error(err, "failed to create deployment")
				return false, err
			}
		} else {
			logger.Error(err, "error not related to missing deployment")
			return false, err
		}
	} else if !reflect.DeepEqual(msDeployment, msDeploymentResult) {
		logger.Info("Deployment found. Updating...")
		err = ctrl.SetControllerReference(msd.Instance, msDeployment, msd.Scheme)
		if err != nil {
			logger.Error(err, "failed to set controller reference")
			return false, err
		}
		err = msd.Client.Update(msd.Context, msDeployment)
		if err != nil {
			logger.Error(err, "failed to update deployment")
			return false, err
		}
	}
	logger.Info("Deployment Reconciled")
	return true, nil
}
*/
