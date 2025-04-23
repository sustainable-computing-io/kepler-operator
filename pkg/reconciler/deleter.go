// SPDX-FileCopyrightText: 2025 The Kepler Authors
// SPDX-License-Identifier: Apache-2.0

package reconciler

import (
	"context"
	"fmt"
	"time"

	"github.com/sustainable.computing.io/kepler-operator/pkg/utils/k8s"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Deleter struct {
	Resource    client.Object
	OnError     Action
	WaitTimeout time.Duration
}

// TODO: replace with builtin max when moving to go 1.21
func maxDuration(a, b time.Duration) time.Duration {
	if a > b {
		return a
	}
	return b
}

func (r Deleter) Reconcile(ctx context.Context, c client.Client, scheme *runtime.Scheme) Result {
	objKey := client.ObjectKeyFromObject(r.Resource)

	if err := c.Delete(ctx, r.Resource); client.IgnoreNotFound(err) != nil {
		return Result{
			Error:  r.error("failed to delete", err),
			Action: r.OnError,
		}
	}

	dup := r.Resource.DeepCopyObject().(client.Object)

	timeout := maxDuration(r.WaitTimeout, 30*time.Second)
	err := wait.PollImmediateWithContext(ctx, 5*time.Second, timeout, func(ctx context.Context) (bool, error) {
		err := c.Get(ctx, objKey, dup)
		// repeat until object is not found
		return errors.IsNotFound(err), nil
	})

	if wait.Interrupted(err) {
		return Result{
			Error:  r.error("timed out waiting for deletion", err),
			Action: r.OnError,
		}
	}
	return Result{}
}

func (r Deleter) error(msg string, err error) error {
	return fmt.Errorf("%s: deleter: %s : %w", k8s.GVKName(r.Resource), msg, err)
}
