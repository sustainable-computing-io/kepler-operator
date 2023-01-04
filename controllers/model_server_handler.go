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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	PersistentVolumeName      = "kepler-model-server-pv"
	PersistentVolumeClaimName = "kepler-model-server-pvc"
	ModelServerConfigMapName  = "kepler-model-server-cfm"
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

func (msd *ModelServerDeployment) buildModelServerConfigMap() corev1.ConfigMap {
	dataPairing := make(map[string]string)
	modelServerExporter := msd.Instance.Spec.ModelServerExporter
	modelServerTrainer := msd.Instance.Spec.ModelServerTrainer
	dataPairing["MNT_PATH"] = "/mnt"
	if modelServerExporter != nil {
		dataPairing["MODEL_SERVER_ENABLE"] = "true"
		dataPairing["PROM_SERVER"] = modelServerExporter.PromServer
		if dataPairing["PROM_SERVER"] == "" {
			dataPairing["PROM_SERVER"] = "http://prometheus-k8s." + msd.Instance.Namespace + ".svc.cluster.local:9090/"
		}
		dataPairing["MODEL_PATH"] = modelServerExporter.ModelPath
		dataPairing["MODEL_SERVER_PORT"] = strconv.Itoa(modelServerExporter.Port)
		dataPairing["MODEL_SERVER_URL"] = modelServerExporter.ModelServerURL
		if dataPairing["MODEL_SERVER_URL"] == "" {
			dataPairing["MODEL_SERVER_URL"] = "http://kepler-model-server." + msd.Instance.Namespace + ".cluster.local:" + strconv.Itoa(modelServerExporter.Port)
		}
		dataPairing["MODEL_SERVER_MODEL_REQ_PATH"] = modelServerExporter.ModelServerRequiredPath
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
			Name:      ModelServerConfigMapName,
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
			/*err = ctrl.SetControllerReference(msd.Instance, msd.PersistentVolumeClaim, msd.Scheme)
			if err != nil {
				logger.Error(err, "failed to set controller reference")
				return false, err
			}
			*/
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
		/*err = ctrl.SetControllerReference(msd.Instance, msd.PersistentVolumeClaim, msd.Scheme)
		if err != nil {
			logger.Error(err, "failed to set controller reference")
			return false, err
		}*/

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

func (msd *ModelServerDeployment) ensureModelServerConfigMap(l klog.Logger) (bool, error) {
	newMSCFM := msd.buildModelServerConfigMap()
	msd.ConfigMap = &corev1.ConfigMap{
		ObjectMeta: newMSCFM.ObjectMeta,
	}
	logger := l.WithValues("ConfigMap", nameFor(msd.ConfigMap))
	op, err := ctrlutil.CreateOrUpdate(msd.Context, msd.Client, msd.ConfigMap, func() error {
		err := ctrl.SetControllerReference(msd.Instance, msd.ConfigMap, msd.Scheme)
		if err != nil {
			logger.Error(err, "failed to set controller reference")
			return err
		}
		// regardless of whether I am creating or updating, data is to be changed
		// test to see if updates occur
		msd.ConfigMap.Data = newMSCFM.Data
		return nil
	})
	if err != nil {
		logger.Error(err, "create/update failed for ConfigMap")
		return false, err
	}
	logger.Info("ConfigMap reconciled", "Operation", op)
	return true, nil

}

func (msd *ModelServerDeployment) ensureModelServerService(l klog.Logger) (bool, error) {
	newMSS := msd.buildModelServerService()

	msd.Service = &corev1.Service{
		ObjectMeta: newMSS.ObjectMeta,
	}

	logger := l.WithValues("ModelServerService", nameFor(msd.Service))
	op, err := ctrlutil.CreateOrUpdate(msd.Context, msd.Client, msd.Service, func() error {
		err := ctrl.SetControllerReference(msd.Instance, msd.Service, msd.Scheme)
		if err != nil {
			logger.Error(err, "failed to set controller reference")
			return err
		}
		// for immutable fields
		if msd.Service.ObjectMeta.CreationTimestamp.IsZero() {
			msd.Service.Spec.ClusterIP = newMSS.Spec.ClusterIP
			msd.Service.Spec.Selector = newMSS.Spec.Selector
			msd.Service.Spec.Ports = newMSS.Spec.Ports
		}
		// for mutable fields
		if !reflect.DeepEqual(msd.Service.Spec.Ports[0].Port, newMSS.Spec.Ports[0].Port) {
			msd.Service.Spec.Ports[0].Port = newMSS.Spec.Ports[0].Port
		}
		return nil
	})
	if err != nil {
		logger.Error(err, "create/update failed for Service")
		return false, err
	}
	logger.Info("Service Reconciled", "Operation", op)
	return true, nil

}

func (msd *ModelServerDeployment) ensureModelServerDeployment(l klog.Logger) (bool, error) {
	newMSD := msd.buildModelServerDeployment()
	msd.Deployment = &appsv1.Deployment{
		ObjectMeta: newMSD.ObjectMeta,
	}
	logger := l.WithValues("ModelServerDeployment", nameFor(msd.Deployment))

	op, err := ctrlutil.CreateOrUpdate(msd.Context, msd.Client, msd.Deployment, func() error {
		err := ctrl.SetControllerReference(msd.Instance, msd.Deployment, msd.Scheme)
		if err != nil {
			logger.Error(err, "failed to set controller reference")
			return err
		}
		// for immutable fields
		if msd.Deployment.ObjectMeta.CreationTimestamp.IsZero() {
			msd.Deployment.Spec.Selector = newMSD.Spec.Selector
			msd.Deployment.Spec.Template.ObjectMeta = newMSD.Spec.Template.ObjectMeta
			msd.Deployment.Spec.Template.Spec.Volumes = newMSD.Spec.Template.Spec.Volumes
		}
		// for mutable fields

		//compare deepcopy to prevent unnecessary updates
		if !reflect.DeepEqual(msd.Deployment.Spec.Template.Spec.Containers, newMSD.Spec.Template.Spec.Containers) {
			msd.Deployment.Spec.Template.Spec.Containers = newMSD.Spec.Template.Spec.Containers
		}
		return nil
	})
	if err != nil {
		logger.Error(err, "create/update failed for Deployment")
		return false, err
	}
	logger.Info("Deployment Reconciled", "Operation", op)
	return true, nil

}
