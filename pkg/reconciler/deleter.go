// SPDX-FileCopyrightText: 2025 The Kepler Authors
// SPDX-License-Identifier: Apache-2.0

package reconciler

import (
	"context"
	"fmt"

	"github.com/sustainable.computing.io/kepler-operator/pkg/utils/k8s"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Deleter requests deletion without blocking reconciliation.
type Deleter struct {
	// Resource is the object to delete.
	Resource client.Object
	// OnError determines how reconciliation proceeds after an API error.
	OnError Action
	// WaitForDeletion requeues while Resource still exists, delaying later reconcilers.
	WaitForDeletion bool
}

func (r Deleter) Reconcile(ctx context.Context, c client.Client, scheme *runtime.Scheme) Result {
	if err := c.Delete(ctx, r.Resource); client.IgnoreNotFound(err) != nil {
		return Result{
			Error:  r.error("failed to delete", err),
			Action: r.OnError,
		}
	}

	if r.WaitForDeletion {
		obj := r.Resource.DeepCopyObject().(client.Object)
		err := c.Get(ctx, client.ObjectKeyFromObject(r.Resource), obj)
		if client.IgnoreNotFound(err) != nil {
			return Result{Error: r.error("failed to check deletion", err), Action: r.OnError}
		}
		if err == nil {
			return Result{Action: Requeue}
		}
	}
	return Result{}
}

func (r Deleter) error(msg string, err error) error {
	return fmt.Errorf("%s: deleter: %s : %w", k8s.GVKName(r.Resource), msg, err)
}
