/*
Copyright 2023.

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
