//go:build !experiment
// +build !experiment

package experiment

import (
	"github.com/sustainable.computing.io/kepler-operator/pkg/utils/k8s"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type KeplerReconciler struct {
	client.Client
	Scheme  *runtime.Scheme
	Cluster k8s.Cluster
}

func (r *KeplerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	mgr.GetLogger().V(3).Info("Experimental API is disabled")
	return nil
}
