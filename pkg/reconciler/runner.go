// SPDX-FileCopyrightText: 2025 The Kepler Authors
// SPDX-License-Identifier: Apache-2.0

package reconciler

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Runner struct {
	Reconcilers []Reconciler
	Client      client.Client
	Scheme      *runtime.Scheme
	Logger      logr.Logger
}

// TODO: make sure that model server container (deployment) is ready before creating kepler daemonset
func (runner Runner) Run(ctx context.Context) (ctrl.Result, error) {
	var err error

	for _, r := range runner.Reconcilers {
		runner.Logger.V(6).Info("reconciler.run ...")
		result := r.Reconcile(ctx, runner.Client, runner.Scheme)

		if result.Error != nil {
			err = result.Error
		}

		switch result.Action {
		case Continue:
			if result.Error != nil {
				runner.Logger.V(3).Info("continue reconciliation despite error", "error", err)
			}
		case Stop:
			runner.Logger.V(3).Info("stopping further reconciliation as requested")
			return ctrl.Result{
				Requeue: err == nil, // requeue if err is nil
			}, err

		case Requeue:
			if err != nil {
				runner.Logger.V(3).Info("requeue reconciliation despite error", "error", err)
			} else {
				runner.Logger.V(3).Info("requeue reconciliation; no error so far")
			}
			return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
		}
	}
	return ctrl.Result{}, err
}
