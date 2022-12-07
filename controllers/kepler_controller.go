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

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/go-logr/logr"
	monitoring "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	keplerv1alpha1 "github.com/sustainable.computing.io/kepler-operator/api/v1alpha1"

	corev1 "k8s.io/api/core/v1"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	rbacv1 "k8s.io/api/rbac/v1"

	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/klog/v2"
)

var (
	// CollectotContainerImage is the container image name of the collector
	CollectotContainerImage string
	// EstimatorContainerImage is the container image name of the estimator
	EstimatorContainerImage string
	// ModelServerContainerImage is the container image name of the model-server
	ModelServerContainerImage = "quay.io/sustainable_computing_io/kepler_model_server:latest"
	// SCCName is the name of the scribe security context constraint
	SCCName string
)

// KeplerReconciler reconciles a Kepler object
type KeplerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

//+kubebuilder:rbac:groups=kepler.system.sustainable.computing.io,resources=keplers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=kepler.system.sustainable.computing.io,resources=keplers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=kepler.system.sustainable.computing.io,resources=keplers/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=persistentvolumes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete

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
	var result ctrl.Result
	var err error

	if inst.Spec.Collector != nil {
		result, err = CollectorReconciler(ctx, inst, r, logger)
	} else if inst.Spec.Estimator != nil {
		// result, err = EstimatorReconciler(ctx, inst, r, logger)
	} else if inst.Spec.ModelServerExporter != nil || inst.Spec.ModelServerTrainer != nil {
		result, err = ModelServerReconciler(ctx, inst, r, logger)
	} else {

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
		r.ensureServiceAccount,
		r.ensureService,
		r.ensureDaemonSet,
		r.ensureServiceMonitor,

		// apply all resoucres here like service account, scc etc here eg r.applyServiceAccount

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

/*
func EstimatorReconciler(ctx context.Context, instance *keplerv1alpha1.Kepler, kr *KeplerReconciler, logger klog.Logger) (ctrl.Result, error) {

}*/

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
		Owns(&rbacv1.RoleBinding{}).
		Owns(&monitoring.ServiceMonitor{}).
		Complete(r)
}

type collectorReconciler struct {
	KeplerReconciler
	serviceAccount *corev1.ServiceAccount
	Instance       *keplerv1alpha1.Kepler
	Ctx            context.Context
	daemonSet      *appsv1.DaemonSet
	service        *corev1.Service
	serviceMonitor *monitoring.ServiceMonitor
}

func (r *collectorReconciler) ensureServiceAccount(l klog.Logger) (bool, error) {
	r.serviceAccount = &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.Instance.Name,
			Namespace: r.Instance.Namespace,
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

func (r *collectorReconciler) ensureDaemonSet(l klog.Logger) (bool, error) {

	dsName := types.NamespacedName{
		Name:      r.Instance.Name + "-exporter",
		Namespace: r.Instance.Namespace,
	}
	logger := l.WithValues("daemonSet", dsName)
	r.daemonSet = &appsv1.DaemonSet{
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

		r.daemonSet.Spec.Template.ObjectMeta.Name = dsName.Name

		r.daemonSet.Spec.Template.Spec.HostNetwork = true

		r.daemonSet.Spec.Template.Spec.ServiceAccountName = r.serviceAccount.Name
		r.daemonSet.Spec.Template.Spec.Containers = []corev1.Container{{
			Name:    "kepler-exporter",
			Image:   "quay.io/sustainable_computing_io/kepler:latest",
			Command: []string{"/usr/bin/kepler", "-address", "0.0.0.0:9102", "-enable-gpu=true", "enable-cgroup-id=true", "v=1"},
			Ports: []corev1.ContainerPort{{
				ContainerPort: 9102,
				HostPort:      9102,
				Name:          "http",
			}},
		}}

		httpget := corev1.HTTPGetAction{
			Path:   "/healthz",
			Port:   intstr.IntOrString{Type: intstr.Int, IntVal: int32(9102)},
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
				FieldPath: "spec.nodeName",
			},
		}
		r.daemonSet.Spec.Template.Spec.Containers[0].Env = []corev1.EnvVar{
			{Name: "NODE_NAME", ValueFrom: &envFromSource},
		}

		r.daemonSet.Spec.Template.Spec.Containers[0].VolumeMounts = []corev1.VolumeMount{
			{Name: "lib-modules", MountPath: "/lib/modules"},
			{Name: "tracing", MountPath: "/sys"},
			{Name: "proc", MountPath: "/proc"},
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
		}

		var matchLabels = make(map[string]string)

		matchLabels["app.kubernetes.io/component"] = "exporter"
		matchLabels["app.kubernetes.io/name"] = "kepler-exporter"

		r.daemonSet.Spec.Selector = &metav1.LabelSelector{
			MatchLabels: matchLabels,
		}

		r.daemonSet.Spec.Template.ObjectMeta = metav1.ObjectMeta{
			Labels: matchLabels,
		}

		return nil
	})

	if err != nil {
		logger.Error(err, "Daemonset Reconcilation failed", "OperationResult: ", op)
		return false, err
	}
	logger.Info("Daemonset reconciled", "OperationResult: ", op)

	return true, nil
}

func (r *collectorReconciler) ensureService(l logr.Logger) (bool, error) {

	serviceName := types.NamespacedName{
		Name:      r.Instance.Name + "-exporter",
		Namespace: r.Instance.Namespace,
	}
	logger := l.WithValues("Service", serviceName)
	r.service = &corev1.Service{
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
				Port: 9102,
				TargetPort: intstr.IntOrString{
					Type:   intstr.Int,
					IntVal: 9102},
			}}

		return nil
	})

	if err != nil {
		logger.Error(err, "Service Reconcilation failed", "OperationResult: ", op)
		return false, err
	}
	logger.Info("kepler-exporter service reconciled", "OperationResult: ", op)
	return true, nil
}
