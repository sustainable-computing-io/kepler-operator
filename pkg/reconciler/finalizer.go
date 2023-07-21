package reconciler

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
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
	ns := r.Resource.GetNamespace()
	name := r.Resource.GetName()
	gvk := r.Resource.GetObjectKind().GroupVersionKind().String()
	return fmt.Errorf("%s/%s (%s): finalizer: %s : %w", ns, name, gvk, msg, err)
}
