package controllers

import (
	monitoring "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *collectorReconciler) ensureServiceMonitor(l klog.Logger) (bool, error) {
	smName := types.NamespacedName{
		Name:      r.Instance.Name + "-exporter",
		Namespace: r.Instance.Namespace,
	}
	logger := l.WithValues("serviceMonitor", smName)
	r.serviceMonitor = &monitoring.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      smName.Name,
			Namespace: smName.Namespace,
		},
	}

	op, err := ctrlutil.CreateOrUpdate(r.Ctx, r.Client, r.serviceMonitor, func() error {
		if err := ctrl.SetControllerReference(r.Instance, r.serviceMonitor, r.Scheme); err != nil {
			logger.Error(err, "unable to set controller reference")
			return err
		}

		r.serviceMonitor.ObjectMeta.Name = smName.Name
		r.serviceMonitor.ObjectMeta.Namespace = smName.Namespace

		var matchLabels = make(map[string]string)

		matchLabels["app.kubernetes.io/component"] = "exporter"
		matchLabels["app.kubernetes.io/name"] = "kepler-exporter"

		r.serviceMonitor.ObjectMeta.Labels = matchLabels

		var relabelConfig = monitoring.RelabelConfig{
			Action:       "replace",
			Regex:        "(.*)",
			Replacement:  "$1",
			SourceLabels: []monitoring.LabelName{"__meta_kubernetes_pod_node_name"},
			TargetLabel:  "instance",
		}

		r.serviceMonitor.Spec.Endpoints = []monitoring.Endpoint{{
			Port:           "http",
			Interval:       monitoring.Duration("3s"),
			RelabelConfigs: []*monitoring.RelabelConfig{&relabelConfig},
			Scheme:         "http",
		}}

		r.serviceMonitor.Spec.Selector = metav1.LabelSelector{
			MatchLabels: matchLabels,
		}

		return nil
	})

	if err != nil {
		logger.Error(err, "ServiceMonitor Reconcilation failed", "OperationResult: ", op)
		return false, err
	}
	logger.Info("ServiceMonitor reconciled", "OperationResult: ", op)

	return true, nil
}
