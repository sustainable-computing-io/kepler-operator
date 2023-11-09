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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type Updater struct {
	Owner    metav1.Object
	Resource client.Object
	OnError  Action
	Logger   logr.Logger
}

func (r Updater) Reconcile(ctx context.Context, c client.Client, scheme *runtime.Scheme) Result {
	ownerNs := r.Owner.GetNamespace()
	resourceNs := r.Resource.GetNamespace()

	if ownerNs == "" || ownerNs == resourceNs {
		if err := ctrlutil.SetControllerReference(r.Owner, r.Resource, scheme); err != nil {

			return Result{
				Action: Stop,
				Error:  r.error("setting controller reference failed", err),
			}
		}
	}

	r.Logger.V(8).Info("updating resource", "resource", k8s.GVKName(r.Resource))

	if err := c.Patch(ctx, r.Resource, client.Apply, client.ForceOwnership, client.FieldOwner("kepler-operator")); err != nil {
		if errors.IsConflict(err) || errors.IsAlreadyExists(err) {
			// the cache may be stale; requests a Reconcile
			r.Logger.V(3).Error(err, "patch failed")
			return Result{
				Action: Requeue,
				Error:  nil, // suppress the error
			}
		}

		return Result{
			Action: r.OnError,
			Error:  r.error("patch failed", err),
		}
	}
	return Result{}
}

func (r Updater) error(msg string, err error) error {
	return fmt.Errorf("%s: updater: %s : %w", k8s.GVKName(r.Resource), msg, err)
}
