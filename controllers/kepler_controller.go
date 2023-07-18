/*
Copyright 2022.

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

package controllers

import (
	"context"
	"strconv"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	keplerv1alpha1 "github.com/sustainable.computing.io/kepler-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var (
	// CollectotContainerImage is the container image name of the collector
	CollectotContainerImage string
	// EstimatorContainerImage is the container image name of the estimator
	EstimatorContainerImage string
	// ModelServerContainerImage is the container image name of the model-server
	ModelServerContainerImage = "quay.io/sustainable_computing_io/kepler_model_server:latest"
	// SCCName is the name of the kepler security context constraint
	SCCName string
)

// KeplerReconciler reconciles a Kepler object
type KeplerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

func (r *KeplerReconciler) removePVC(logger logr.Logger, inst *keplerv1alpha1.Kepler, ctx context.Context) error {
	msPVCResult := &corev1.PersistentVolumeClaim{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: ModelServerPersistentVolumeClaimNameSuffix, Namespace: inst.Namespace}, msPVCResult)
	if err != nil {
		if kerrors.IsNotFound(err) {
			logger.Info("PVC has already been deleted")
			return nil
		} else {
			logger.Error(err, "failed to get PVC")
			return err
		}
	}
	// PVC has been retrieved
	err = r.Client.Delete(ctx, msPVCResult)
	if err != nil {
		logger.Error(err, "failed to delete PVC")
		return err
	}

	logger.Info("Successfully Removed PVC")
	return nil
}

//+kubebuilder:rbac:groups=kepler.system.sustainable.computing.io,resources=keplers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=kepler.system.sustainable.computing.io,resources=keplers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=kepler.system.sustainable.computing.io,resources=keplers/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=services;configmaps;serviceaccounts;persistentvolumes;persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments;daemonsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=*,verbs=*
//+kubebuilder:rbac:groups=core,resources=nodes/metrics;nodes/proxy;nodes/stats,verbs=get;list;watch
//+kubebuilder:rbac:groups=security.openshift.io,resources=securitycontextconstraints,verbs=get;list;watch;create;update;patch;delete;use

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Kepler object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *KeplerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	logger := log.Log.WithValues("kepler", req.NamespacedName)
	r.Log = logger

	inst := &keplerv1alpha1.Kepler{}
	if err := r.Client.Get(ctx, req.NamespacedName, inst); err != nil {
		if kerrors.IsNotFound(err) {
			klog.Error(err, "Failed to get namespace")
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	//Remove Finalizer if it exists and perform Clean-up
	if inst.GetDeletionTimestamp() != nil {
		if ctrlutil.ContainsFinalizer(inst, keplerFinalizer) {
			errorPVC := r.removePVC(logger, inst, ctx)
			if errorPVC != nil {
				return ctrl.Result{}, errorPVC
			}
			// Remove finalizer
			ctrlutil.RemoveFinalizer(inst, keplerFinalizer)
			updateError := r.Client.Update(ctx, inst)
			if updateError != nil {
				return ctrl.Result{}, updateError
			}
		}
		return ctrl.Result{}, nil
	}

	//Apply Finalizer to Kepler Object

	if inst.Spec.ModelServerFeatures != nil && inst.Spec.ModelServerFeatures.IncludePVandPVCFinalizer {
		if !ctrlutil.ContainsFinalizer(inst, keplerFinalizer) {
			ctrlutil.AddFinalizer(inst, keplerFinalizer)
			updateErr := r.Client.Update(ctx, inst)
			if updateErr != nil {
				return ctrl.Result{}, updateErr
			}
		}
	}

	var result ctrl.Result
	var err error
	if inst.Spec.Collector != nil {
		result, err = CollectorReconciler(ctx, inst, r, logger)
	}

	if inst.Spec.ModelServerExporter != nil {
		result, err = ModelServerReconciler(ctx, inst, r, logger)
	}

	if inst.Spec.Collector == nil && inst.Spec.Estimator == nil && inst.Spec.ModelServerExporter == nil {
		return result, nil
	}

	// Set reconcile status condition
	if err == nil {
		inst.Status.Conditions = metav1.Condition{
			Type:               keplerv1alpha1.ConditionReconciled,
			Status:             metav1.ConditionTrue,
			Reason:             keplerv1alpha1.ReconciledReasonComplete,
			Message:            "Reconcile complete",
			LastTransitionTime: inst.CreationTimestamp,
		}
	} else {
		inst.Status.Conditions = metav1.Condition{
			Type:               keplerv1alpha1.ConditionReconciled,
			Status:             metav1.ConditionTrue,
			Reason:             keplerv1alpha1.ReconciledReasonError,
			Message:            err.Error(),
			LastTransitionTime: inst.CreationTimestamp,
		}
	}
	// Update instance status
	statusErr := r.Client.Status().Update(ctx, inst)
	if err == nil { // Don't mask previous error
		err = statusErr
	}
	return result, err
}

//nolint:dupl
func CollectorReconciler(ctx context.Context, instance *keplerv1alpha1.Kepler, kr *KeplerReconciler, logger klog.Logger) (ctrl.Result, error) {
	r := collectorReconciler{
		Ctx:              ctx,
		Instance:         instance,
		KeplerReconciler: *kr,
	}

	l := logger.WithValues("method", "Collector")
	_, err := reconcileBatch(l,

		r.ensureConfigMap,
		r.ensureService,
		r.ensureServiceAccount,
		r.ensureSCC,
		r.ensureDaemonSet,
	)

	return ctrl.Result{}, err

}

type modelServerReconciler struct {
	KeplerReconciler
	Ctx      context.Context
	Instance *keplerv1alpha1.Kepler
}

func (mser *modelServerReconciler) ensureModelServer(l klog.Logger) (bool, error) {
	modelServerDeployment := ModelServerDeployment{
		Context:  mser.Ctx,
		Instance: mser.Instance,
		Image:    ModelServerContainerImage,
		Client:   mser.Client,
		Scheme:   mser.Scheme,
	}
	return modelServerDeployment.Reconcile(l)
}

func ModelServerReconciler(ctx context.Context, instance *keplerv1alpha1.Kepler, kr *KeplerReconciler, logger klog.Logger) (ctrl.Result, error) {
	modelServerReconciler := modelServerReconciler{
		KeplerReconciler: *kr,
		Ctx:              ctx,
		Instance:         instance,
	}
	log := logger.WithValues("method", "Model Server")
	_, err := reconcileBatch(
		log,
		modelServerReconciler.ensureModelServer,
	)
	return ctrl.Result{}, err

}

// SetupWithManager sets up the controller with the Manager.
func (r *KeplerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&keplerv1alpha1.Kepler{}).
		Owns(&corev1.PersistentVolume{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Service{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&rbacv1.Role{}).
		Owns(&rbacv1.ClusterRole{}).
		Owns(&rbacv1.ClusterRoleBinding{}).
		Owns(&rbacv1.RoleBinding{}).
		Complete(r)
}

type collectorReconciler struct {
	KeplerReconciler
	serviceAccount *corev1.ServiceAccount
	Instance       *keplerv1alpha1.Kepler
	Ctx            context.Context
	daemonSet      *appsv1.DaemonSet
	service        *corev1.Service
	configMap      *corev1.ConfigMap
}

func (r *collectorReconciler) ensureServiceAccount(l klog.Logger) (bool, error) {
	r.serviceAccount = &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			Kind: "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.Instance.Name + ServiceAccountNameSuffix,
			Namespace: r.Instance.Namespace + ServiceAccountNameSpaceSuffix,
		},
	}
	kepDesc := keplerSADescription{
		Context: r.Ctx,
		Client:  r.Client,
		Scheme:  r.Scheme,
		SA:      r.serviceAccount,
		Owner:   r.Instance,
	}
	return kepDesc.Reconcile(l)
}

func (r *collectorReconciler) ensureConfigMap(l klog.Logger) (bool, error) {
	// strconv.Itoa(r.Instance.spe)
	bindAddress := "0.0.0.0:" + strconv.Itoa(r.Instance.Spec.Collector.CollectorPort)
	cmName := types.NamespacedName{
		Name:      r.Instance.Name + CollectorConfigMapNameSuffix,
		Namespace: r.Instance.Namespace + CollectorConfigMapNameSpaceSuffix,
	}
	logger := l.WithValues("configmap", cmName)
	r.configMap = &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cmName.Name,
			Namespace: cmName.Namespace,
		},
	}
	op, err := ctrlutil.CreateOrUpdate(r.Ctx, r.Client, r.configMap, func() error {
		if err := ctrl.SetControllerReference(r.Instance, r.configMap, r.Scheme); err != nil {
			logger.Error(err, "unable to set controller reference")
			return err
		}
		r.configMap.ObjectMeta.Name = cmName.Name
		r.configMap.ObjectMeta.Namespace = cmName.Namespace

		var data_map = make(map[string]string)

		data_map["KEPLER_NAMESPACE"] = r.Instance.Namespace
		data_map["KEPLER_LOG_LEVEL"] = "5"
		data_map["METRIC_PATH"] = "/metrics"
		data_map["BIND_ADDRESS"] = bindAddress
		data_map["ENABLE_GPU"] = "true"
		data_map["ENABLE_EBPF_CGROUPID"] = "true"
		data_map["CPU_ARCH_OVERRIDE"] = ""
		data_map["CGROUP_METRICS"] = "*"
		data_map["MODEL_CONFIG"] = "| CONTAINER_COMPONENTS_ESTIMATOR=false CONTAINER_COMPONENTS_INIT_URL=https://raw.githubusercontent.com/sustainable-computing-io/kepler-model-server/main/tests/test_models/DynComponentModelWeight/CgroupOnly/ScikitMixed/ScikitMixed.json"

		data_map["EXPOSE_HW_COUNTER_METRICS"] = "true"
		data_map["EXPOSE_CGROUP_METRICS"] = "true"
		r.configMap.Data = data_map

		return nil

	})
	if err != nil {
		logger.Error(err, "ConfigMap Reconcilation failed", "OperationResult: ", op)
		return false, err
	}
	logger.V(1).Info("ConfigMap reconciled", "OperationResult", op)
	return true, nil

}

func (r *collectorReconciler) ensureDaemonSet(l klog.Logger) (bool, error) {
	collectorPort := int32(r.Instance.Spec.Collector.CollectorPort)
	bindAddress := "0.0.0.0:" + strconv.Itoa(r.Instance.Spec.Collector.CollectorPort)

	dsName := types.NamespacedName{
		Name:      r.Instance.Name + DaemonSetNameSuffix,
		Namespace: r.Instance.Namespace + DaemonSetNameSpaceSuffix,
	}
	logger := l.WithValues("daemonSet", dsName)

	r.daemonSet = &appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			Kind: "DaemonSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      dsName.Name,
			Namespace: dsName.Namespace,
		},
	}

	op, err := ctrlutil.CreateOrUpdate(r.Ctx, r.Client, r.daemonSet, func() error {
		if err := ctrl.SetControllerReference(r.Instance, r.daemonSet, r.Scheme); err != nil {
			logger.Error(err, "unable to set controller reference")
			return err
		}

		var scc_value bool = true

		r.daemonSet.Spec.Template.Spec.DNSPolicy = corev1.DNSPolicy(corev1.DNSClusterFirstWithHostNet)

		tolerations := []corev1.Toleration{
			{
				Effect: corev1.TaintEffectNoSchedule,
				Key:    "node-role.kubernetes.io/master",
			}}

		r.daemonSet.Spec.Template.Spec.Tolerations = tolerations

		r.daemonSet.Spec.Template.ObjectMeta.Name = dsName.Name
		r.daemonSet.Spec.Template.Spec.ServiceAccountName = r.serviceAccount.Name
		image := r.Instance.Spec.Collector.Image
		logger.V(1).Info("DaemonSet Image", "image", image)
		r.daemonSet.Spec.Template.Spec.Containers = []corev1.Container{{
			Name: "kepler-exporter",
			SecurityContext: &corev1.SecurityContext{
				Privileged: &scc_value,
			},
			Image:   image,
			Command: []string{"/usr/bin/kepler", "-address", bindAddress, "-enable-gpu=true", "-enable-cgroup-id=true", "-v=1", "-kernel-source-dir=/usr/share/kepler/kernel_sources"},
			Ports: []corev1.ContainerPort{{
				ContainerPort: collectorPort,
				Name:          "http",
			}},
		}}
		httpget := corev1.HTTPGetAction{
			Path:   "/healthz",
			Port:   intstr.IntOrString{Type: intstr.Int, IntVal: collectorPort},
			Scheme: "HTTP",
		}

		probeHandler := corev1.ProbeHandler{
			HTTPGet: &httpget,
		}

		probe := corev1.Probe{
			ProbeHandler:        probeHandler,
			FailureThreshold:    5,
			InitialDelaySeconds: 10,
			PeriodSeconds:       60,
			SuccessThreshold:    1,
			TimeoutSeconds:      10,
		}
		r.daemonSet.Spec.Template.Spec.Containers[0].LivenessProbe = &probe

		envFromSource := corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				FieldPath: "status.hostIP",
			},
		}
		envFromSourceNode := corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				FieldPath: "spec.nodeName",
			},
		}
		r.daemonSet.Spec.Template.Spec.Containers[0].Env = []corev1.EnvVar{
			{Name: "NODE_IP", ValueFrom: &envFromSource},
			{Name: "NODE_NAME", ValueFrom: &envFromSourceNode},
		}

		r.daemonSet.Spec.Template.Spec.Containers[0].VolumeMounts = []corev1.VolumeMount{
			{Name: "lib-modules", MountPath: "/lib/modules"},
			{Name: "tracing", MountPath: "/sys"},
			{Name: "kernel-src", MountPath: "/usr/src/kernels"},
			{Name: "kernel-debug", MountPath: "/sys/kernel/debug"},
			{Name: "proc", MountPath: "/proc"},
			{Name: "cfm", MountPath: "/etc/kepler/kepler.config"},
		}

		r.daemonSet.Spec.Template.Spec.Volumes = []corev1.Volume{
			{Name: "lib-modules",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{
						Path: "/lib/modules",
					}},
			},
			{Name: "tracing",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{
						Path: "/sys",
					},
				}},
			{Name: "proc",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{
						Path: "/proc",
					},
				}},
			{Name: "kernel-debug",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{
						Path: "/sys/kernel/debug",
					},
				}},
			{Name: "kernel-src",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{
						Path: "/usr/src/kernels",
					},
				}},
			{Name: "cfm",
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							//Name: "kepler-exporter-cfm",
							Name: r.Instance.Name + CollectorConfigMapNameSuffix,
						},
					}}},
		}
		var matchLabels = make(map[string]string)

		matchLabels["app.kubernetes.io/component"] = "exporter"
		matchLabels["app.kubernetes.io/name"] = "kepler-exporter"
		matchLabels["sustainable-computing.io/app"] = "kepler"

		r.daemonSet.Spec.Selector = &metav1.LabelSelector{
			MatchLabels: matchLabels,
		}
		r.daemonSet.Spec.Template.ObjectMeta = metav1.ObjectMeta{
			Labels: matchLabels,
			Name:   dsName.Name,
		}

		return nil
	})

	if err != nil {
		logger.Error(err, "Daemonset Reconcilation failed", "OperationResult: ", op)
		return false, err
	}
	logger.V(1).Info("Daemonset reconciled", "OperationResult", op)

	return true, nil
}

func (r *collectorReconciler) ensureService(l logr.Logger) (bool, error) {

	collectorPort := int32(r.Instance.Spec.Collector.CollectorPort)

	serviceName := types.NamespacedName{
		Name:      r.Instance.Name + ServiceNameSuffix,
		Namespace: r.Instance.Namespace + ServiceNameSpaceSuffix,
	}
	logger := l.WithValues("Service", serviceName)
	r.service = &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind: "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName.Name,
			Namespace: serviceName.Namespace,
		},
	}
	op, err := ctrlutil.CreateOrUpdate(r.Ctx, r.Client, r.service, func() error {
		if err := ctrl.SetControllerReference(r.Instance, r.service, r.Scheme); err != nil {
			logger.Error(err, "unable to set controller reference")
			return err
		}

		r.service.ObjectMeta.Name = serviceName.Name
		r.service.ObjectMeta.Namespace = serviceName.Namespace

		if r.service.ObjectMeta.Labels == nil {
			r.service.ObjectMeta.Labels = map[string]string{}
		}
		r.service.ObjectMeta.Labels["app.kubernetes.io/component"] = "exporter"
		r.service.ObjectMeta.Labels["app.kubernetes.io/name"] = "kepler-exporter"
		r.service.Spec.ClusterIP = "None"
		if r.service.Spec.Selector == nil {
			r.service.Spec.Selector = map[string]string{}
		}
		r.service.Spec.Selector["app.kubernetes.io/component"] = "exporter"
		r.service.Spec.Selector["app.kubernetes.io/name"] = "kepler-exporter"

		r.service.Spec.Ports = []corev1.ServicePort{
			{
				Name: "http",
				Port: collectorPort,
				TargetPort: intstr.IntOrString{
					Type:   intstr.Int,
					IntVal: collectorPort},
			}}

		return nil
	})

	if err != nil {
		logger.Error(err, "Service Reconcilation failed", "OperationResult: ", op)
		return false, err
	}
	logger.V(1).Info("kepler-exporter service reconciled", "OperationResult", op)

	return true, nil
}
