// SPDX-FileCopyrightText: 2025 The Kepler Authors
// SPDX-License-Identifier: Apache-2.0

package reconciler

import (
	"context"

	"github.com/sustainable.computing.io/kepler-operator/api/v1alpha1"
	"github.com/sustainable.computing.io/kepler-operator/pkg/components"
	powermonitor "github.com/sustainable.computing.io/kepler-operator/pkg/components/power-monitor"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PowerMonitorServiceMonitorReconciler struct {
	Pmi        *v1alpha1.PowerMonitorInternal
	EnableRBAC bool
	EnableUWM  bool
}

func (r PowerMonitorServiceMonitorReconciler) Reconcile(ctx context.Context, cli client.Client, s *runtime.Scheme) Result {
	sm := powermonitor.NewPowerMonitorServiceMonitor(components.Full, r.Pmi)
	if r.EnableRBAC && !r.EnableUWM {
		return Deleter{Resource: sm}.Reconcile(ctx, cli, s)
	}

	return Updater{Owner: r.Pmi, Resource: sm}.Reconcile(ctx, cli, s)
}
