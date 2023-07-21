package reconciler

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

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
	return Result{}
}

func (r Deleter) error(msg string, err error) error {
	ns := r.Resource.GetNamespace()
	name := r.Resource.GetName()
	gvk := r.Resource.GetObjectKind().GroupVersionKind().String()
	return fmt.Errorf("%s/%s (%s): deleter: %s : %w", ns, name, gvk, msg, err)
}
