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
