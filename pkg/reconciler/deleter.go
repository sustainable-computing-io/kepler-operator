// SPDX-FileCopyrightText: 2025 The Kepler Authors
// SPDX-License-Identifier: Apache-2.0

package reconciler

import (
	"context"
	"fmt"

	"github.com/sustainable.computing.io/kepler-operator/pkg/utils/k8s"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Deleter is a non-blocking reconciler that deletes a resource.
// It follows the check-and-requeue pattern:
// 1. Check if resource exists - if not, return success (already deleted)
// 2. Issue delete request (non-blocking)
// 3. Return Requeue to check completion on next reconciliation
//
// This pattern avoids blocking the reconciliation loop while waiting for
// Kubernetes garbage collection to complete.
type Deleter struct {
	Resource client.Object
	OnError  Action
}

func (r Deleter) Reconcile(ctx context.Context, c client.Client, scheme *runtime.Scheme) Result {
	objKey := client.ObjectKeyFromObject(r.Resource)

	// Check if resource still exists
	dup := r.Resource.DeepCopyObject().(client.Object)
	if err := c.Get(ctx, objKey, dup); err != nil {
		if errors.IsNotFound(err) {
			// Resource is already gone, nothing to do
			return Result{}
		}
		return Result{
			Error:  r.error("failed to check existence", err),
			Action: r.OnError,
		}
	}

	if err := c.Delete(ctx, r.Resource); client.IgnoreNotFound(err) != nil {
		return Result{
			Error:  r.error("failed to delete", err),
			Action: r.OnError,
		}
	}

	// Requeue to verify deletion completed on next reconciliation
	// This avoids blocking while waiting for Kubernetes GC
	return Result{Action: Requeue}
}

func (r Deleter) error(msg string, err error) error {
	return fmt.Errorf("%s: deleter: %s : %w", k8s.GVKName(r.Resource), msg, err)
}
