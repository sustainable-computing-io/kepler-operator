package reconciler

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Action int

const (
	Continue Action = iota
	Requeue
	Stop
)

func (a Action) String() string {
	return [...]string{"Continue", "Requeue", "Stop"}[a]
}

// Result represents the result of reconciliation. Zero value indicates reconciliation ran fine
type Result struct {
	Action Action
	Error  error
}

type Reconciler interface {
	Reconcile(context.Context, client.Client, *runtime.Scheme) Result
}
