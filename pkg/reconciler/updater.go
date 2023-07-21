package reconciler

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
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
	ns := r.Resource.GetNamespace()
	name := r.Resource.GetName()
	gvk := r.Resource.GetObjectKind().GroupVersionKind().String()
	return fmt.Errorf("%s/%s (%s): updater: %s : %w", ns, name, gvk, msg, err)
}
