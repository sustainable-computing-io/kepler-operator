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
	ctrlutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	//ctrl "sigs.k8s.io/controller-runtime"
)

var ModelServerEndpoint string

type ReconciliationResult byte

const (
	UnexpectedError ReconciliationResult = iota
	NoObjectExistedCreateNew
	ObjectExistedUpdatedorReplaced
	ObjectExistedIgnore
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
			Name:      "kepler-model-server-cfm",
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
			Name:      "kepler-model-server-pvc",
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
		"app.kubernetes.io/component": "kepler-model-server-pv",
		"app.kubernetes.io/name":      "kepler-model-server-pv",
	}
	modelServerPV := corev1.PersistentVolume{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PersistentVolume",
			APIVersion: msd.Instance.APIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   "kepler-model-server-pv",
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
			Name:      "kepler-model-server",
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
			Name:      "kepler-model-server",
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

	err := msd.Client.Get(msd.Context, types.NamespacedName{Name: "kepler-model-server-pvc", Namespace: msd.Instance.Namespace}, msPVCResult)

	if err != nil && errors.IsNotFound(err) {
		if errors.IsNotFound(err) {
			l.Info("PVC does not exist. Creating...")
			ctrlutil.SetControllerReference(msd.Instance, msPVC, msd.Scheme)
			err = msd.Client.Create(msd.Context, msPVC)
			if err != nil {
				//return UnexpectedError, err
				return false, err
			}
		} else {
			l.Error(err, "failed to get PVC")
			return false, err
		}
	}

	// TODO: if it does exist, only modify mutable fields when they become variable in future
	//return ObjectExistedIgnore, nil
	return true, nil
}

func (msd *ModelServerDeployment) ensureModelServerPersistentVolume(l klog.Logger) (bool, error) { //(ReconciliationResult, error) {
	// Create PV and PVC
	msd.buildModelServerPV()
	msPV := msd.PersistentVolume
	msPVResult := &corev1.PersistentVolume{}

	err := msd.Client.Get(msd.Context, types.NamespacedName{Name: "kepler-model-server-pv"}, msPVResult)

	if err != nil && errors.IsNotFound(err) {
		if errors.IsNotFound(err) {
			l.Info("PV does not exist. Creating...")
			ctrlutil.SetControllerReference(msd.Instance, msPV, msd.Scheme)
			err = msd.Client.Create(msd.Context, msPV)
			if err != nil {
				//return UnexpectedError, err
				return false, err
			}
		} else {
			l.Error(err, "failed to get PV")
			return false, err
		}
	}

	// TODO: if it does exist, only modify mutable fields when they become variable in future

	//return ObjectExistedIgnore, nil
	return true, nil
}

func (msd *ModelServerDeployment) ensureModelServerConfigMap(l klog.Logger) (bool, error) { //(ReconciliationResult, error) {

	msd.buildModelServerConfigMap()
	msCFM := msd.ConfigMap
	msCFMResult := &corev1.ConfigMap{}

	err := msd.Client.Get(msd.Context, types.NamespacedName{Name: "kepler-model-server-cfm", Namespace: msd.Instance.Namespace}, msCFMResult)
	if err != nil {
		if errors.IsNotFound(err) {
			l.Info("ConfigMap does not exist. Creating...")
			ctrlutil.SetControllerReference(msd.Instance, msCFM, msd.Scheme)
			err = msd.Client.Create(msd.Context, msCFM)
			if err != nil {
				l.Error(err, "failed to create Config Map")
				//return UnexpectedError, err
				return false, err
			}
		} else {
			l.Error(err, "failed to get Config Map")
			//return UnexpectedError, err
			return false, err
		}
	}
	return true, nil
	//return NoObjectExistedCreateNew, nil
	/*} else if !reflect.DeepEqual(msCFM.Data, msCFMResult.Data) {
		// If they are different, we must delete ConfigMap and Deployment (we then redeploy )
		// here we just delete and add new ConfigMap and return with the fact that object has existed and was changed
		// so the deployment must be updated!


		//if ConfigMap has changed, deployment must be reset to remount!
		//if deployment containerPort is different, new service must be made with updated port!
		return ObjectExistedUpdatedorReplaced, nil
	}*/

	// Note that only the ConfigMap data values can be changed
	// Operator CRD enforces certain restrictions (do not need to worry about metadata being changed like Name or Namespace by user)

}

func (msd *ModelServerDeployment) ensureModelServerService(l klog.Logger) (bool, error) { //(ReconciliationResult, error) {
	msd.buildModelServerService()

	msService := msd.Service
	msServiceResult := &corev1.Service{}

	err := msd.Client.Get(msd.Context, types.NamespacedName{Name: "kepler-model-server", Namespace: msd.Instance.Namespace}, msServiceResult)
	if err != nil {
		if errors.IsNotFound(err) {
			l.Info("Service does not exist. Creating...")
			ctrlutil.SetControllerReference(msd.Instance, msService, msd.Scheme)
			err = msd.Client.Create(msd.Context, msService)
			if err != nil {
				l.Error(err, "failed to create Service")
				//return UnexpectedError, err
				return false, err
			}
		} else {
			l.Error(err, "failed to get Service")
			//return UnexpectedError, err
			return false, err
		}
	}

	return true, nil

}

func (msd *ModelServerDeployment) ensureModelServerDeployment(l klog.Logger) (bool, error) {
	msd.buildModelServerDeployment()

	msDeployment := msd.Deployment
	msDeploymentResult := &appsv1.Deployment{}

	err := msd.Client.Get(msd.Context, types.NamespacedName{Name: "kepler-model-server", Namespace: msd.Instance.Namespace}, msDeploymentResult)
	if err != nil {
		if errors.IsNotFound(err) {
			l.Info("Deployment does not exist. Creating...")
			ctrlutil.SetControllerReference(msd.Instance, msDeployment, msd.Scheme)
			err = msd.Client.Create(msd.Context, msDeployment)
			if err != nil {
				l.Error(err, "failed to create Deployment")
				//return UnexpectedError, err
				return false, err
			}
		} else {
			l.Error(err, "failed to get Deployment")
			//return UnexpectedError, err
			return false, err
		}
	}
	if err == nil {
		l.Info("Deployment Name and Type", msDeploymentResult.TypeMeta.Kind, msDeploymentResult.Kind)
	}
	return true, nil

}

/*

func (msd *ModelServerDeployment) ensureModelServer(l klog.Logger) (bool, error) {

	return reconcileBatch(l,
		msd.ensureModelServerPersistentVolume,
		msd.ensureModelServerPersistentVolumeClaim,
		msd.ensureModelServerConfigMap,
		msd.ensureModelServerService,
		msd.ensureModelServerDeployment,
	)
	// Create PV and PVC if not exist

	// Create Or Update ConfigMap

	// Changes to ConfigMap will result in deletion of Deployment and Service

	// Launch Deployment and Service

}
*/
