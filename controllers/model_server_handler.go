package controllers

import (
	"context"
	"reflect"
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
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	ctrlutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	//ctrl "sigs.k8s.io/controller-runtime"
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
		msd.ensureModelServerConfigMap,
		msd.ensureModelServerService,
		msd.ensureModelServerDeployment,
	)
}

func (msd *ModelServerDeployment) buildModelServerConfigMap() {
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
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: msd.Instance.APIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      ConfigMapName,
			Namespace: msd.Instance.Namespace,
		},
		Data: dataPairing,
	}
	msd.ConfigMap = &configMap
}

func (msd *ModelServerDeployment) buildModelServerPVC() {
	storageClassName := "default"
	modelServerPVC := corev1.PersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PersistentVolumeClaim",
			APIVersion: msd.Instance.APIVersion,
		},
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
	msd.PersistentVolumeClaim = &modelServerPVC
}

func (msd *ModelServerDeployment) buildModelServerPV() {
	labels := map[string]string{
		"type":                        "local",
		"app.kubernetes.io/component": PersistentVolumeName,
		"app.kubernetes.io/name":      PersistentVolumeName,
	}
	modelServerPV := corev1.PersistentVolume{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PersistentVolume",
			APIVersion: msd.Instance.APIVersion,
		},
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
	msd.PersistentVolume = &modelServerPV
}

func (msd *ModelServerDeployment) buildModelServerService() {
	labels := map[string]string{
		"app.kubernetes.io/component": "model-server",
		"app.kubernetes.io/name":      "kepler-model-server",
	}
	modelServerService := corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: msd.Instance.APIVersion,
		},
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
	msd.Service = &modelServerService
}

func (msd *ModelServerDeployment) buildModelServerDeployment() {
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
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: msd.Instance.APIVersion,
		},
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

	msd.Deployment = &deployment

}

func (msd *ModelServerDeployment) ensureModelServerPersistentVolumeClaim(l klog.Logger) (bool, error) { //(ReconciliationResult, error) {
	msd.buildModelServerPVC()
	msPVC := msd.PersistentVolumeClaim
	msPVCResult := &corev1.PersistentVolumeClaim{}
	logger := l.WithValues("PVC", nameFor(msPVC))
	err := msd.Client.Get(msd.Context, types.NamespacedName{Name: PersistentVolumeClaimName, Namespace: msd.Instance.Namespace}, msPVCResult)

	if err != nil && errors.IsNotFound(err) {
		if errors.IsNotFound(err) {
			logger.Info("PVC does not exist. creating...")
			ctrlutil.SetControllerReference(msd.Instance, msPVC, msd.Scheme)
			err = msd.Client.Create(msd.Context, msPVC)
			if err != nil {
				logger.Error(err, "failed to create PVC")
				return false, err
			}
		} else {
			logger.Error(err, "error is not a missing error")
			return false, err
		}
	}
	logger.Info("PVC reconciled")
	return true, nil
}

func (msd *ModelServerDeployment) ensureModelServerPersistentVolume(l klog.Logger) (bool, error) { //(ReconciliationResult, error) {
	msd.buildModelServerPV()
	msPV := msd.PersistentVolume
	msPVResult := &corev1.PersistentVolume{}
	logger := l.WithValues("PV", nameFor(msPV))
	err := msd.Client.Get(msd.Context, types.NamespacedName{Name: PersistentVolumeName}, msPVResult)

	if err != nil && errors.IsNotFound(err) {
		if errors.IsNotFound(err) {
			logger.Info("PV does not exist. Creating...")
			ctrlutil.SetControllerReference(msd.Instance, msPV, msd.Scheme)
			err = msd.Client.Create(msd.Context, msPV)
			if err != nil {
				logger.Error(err, "failed to create PV")
				return false, err
			}
		} else {
			logger.Error(err, "error not related to missing PV")
			return false, err
		}
	}
	logger.Info("PV reconciled")
	return true, nil
}

func (msd *ModelServerDeployment) ensureModelServerConfigMap(l klog.Logger) (bool, error) { //(ReconciliationResult, error) {
	msd.buildModelServerConfigMap()
	msCFM := msd.ConfigMap
	msCFMResult := &corev1.ConfigMap{}
	logger := l.WithValues("ConfigMap", nameFor(msCFM))
	err := msd.Client.Get(msd.Context, types.NamespacedName{Name: ConfigMapName, Namespace: msd.Instance.Namespace}, msCFMResult)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("ConfigMap does not exist. Creating...")
			ctrlutil.SetControllerReference(msd.Instance, msCFM, msd.Scheme)
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
		controllerutil.SetControllerReference(msd.Instance, msCFM, msd.Scheme)
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
			ctrlutil.SetControllerReference(msd.Instance, msService, msd.Scheme)
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
		controllerutil.SetControllerReference(msd.Instance, msService, msd.Scheme)
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
			ctrlutil.SetControllerReference(msd.Instance, msDeployment, msd.Scheme)
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
		controllerutil.SetControllerReference(msd.Instance, msDeployment, msd.Scheme)
		err = msd.Client.Update(msd.Context, msDeployment)
		if err != nil {
			logger.Error(err, "failed to update deployment")
			return false, err
		}
	}
	logger.Info("Deployment Reconciled")
	return true, nil
}
