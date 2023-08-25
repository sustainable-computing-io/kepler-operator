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
	"fmt"

	"github.com/go-logr/logr"
	"github.com/sustainable.computing.io/kepler-operator/pkg/utils/k8s"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type Finalizer struct {
	Resource  client.Object
	Finalizer string
	Logger    logr.Logger
}

func (r Finalizer) Reconcile(ctx context.Context, c client.Client, s *runtime.Scheme) Result {
	logger := r.Logger.WithValues("reconciler", "finalizer")

	// NOTE: we can safely typecast since Resource is a client.Object

	// refresh the object before adding or removing Finalizer
	refreshed := r.Resource.DeepCopyObject().(client.Object)
	objKey := client.ObjectKeyFromObject(r.Resource)
	if err := c.Get(ctx, objKey, refreshed); err != nil {
		if errors.IsNotFound(err) {
			logger.V(3).Info("object is already deleted; no action taken", "key", objKey)
			return Result{}
		}
		return Result{Action: Requeue, Error: r.error("failed to refresh", err)}
	}

	deleted := !refreshed.GetDeletionTimestamp().IsZero()
	hasFinalizer := ctrlutil.ContainsFinalizer(refreshed, r.Finalizer)

	logger.V(3).Info("finalizer state", "deleted", deleted, "finalizer", hasFinalizer)

	if deleted && hasFinalizer {
		logger.V(3).Info("removing finalizer")

		ctrlutil.RemoveFinalizer(refreshed, r.Finalizer)
		err := c.Update(ctx, refreshed)
		return Result{Error: err, Action: Stop}
	}

	if !deleted && !hasFinalizer {
		logger.V(3).Info("no finalizer found; adding it")

		ctrlutil.AddFinalizer(refreshed, r.Finalizer)
		err := c.Update(ctx, refreshed)
		return Result{Error: err, Action: Stop}
	}

	return Result{}
}

func (r Finalizer) error(msg string, err error) error {
	return fmt.Errorf("%s: finalizer: %s : %w", k8s.GVKName(r.Resource), msg, err)
}
