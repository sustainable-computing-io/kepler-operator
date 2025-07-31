// SPDX-FileCopyrightText: 2025 The Kepler Authors
// SPDX-License-Identifier: Apache-2.0

package reconciler

import (
	"context"

	monv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/sustainable.computing.io/kepler-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PowerMonitorServiceMonitorReconciler struct {
	Pmi        *v1alpha1.PowerMonitorInternal
	Sm         *monv1.ServiceMonitor
	EnableRBAC bool
	EnableUWM  bool
}

func (r PowerMonitorServiceMonitorReconciler) Reconcile(ctx context.Context, cli client.Client, s *runtime.Scheme) Result {
	if r.EnableRBAC && !r.EnableUWM {
		return Deleter{Resource: r.Sm}.Reconcile(ctx, cli, s)
	}

	return Updater{Owner: r.Pmi, Resource: r.Sm}.Reconcile(ctx, cli, s)
}
