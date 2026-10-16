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

// Deleter issues a non-blocking delete and returns Continue, allowing
// multiple resources to be deleted in a single reconciliation pass.
// Guaranteed cleanup is provided by owner references and Kubernetes GC.
type Deleter struct {
	Resource client.Object
	OnError  Action
}

func (r Deleter) Reconcile(ctx context.Context, c client.Client, scheme *runtime.Scheme) Result {
	if err := c.Delete(ctx, r.Resource); client.IgnoreNotFound(err) != nil {
		return Result{
			Error:  r.error("failed to delete", err),
			Action: r.OnError,
		}
	}

	return Result{Action: Continue}
}

func (r Deleter) error(msg string, err error) error {
	return fmt.Errorf("%s: deleter: %s : %w", k8s.GVKName(r.Resource), msg, err)
}
