package reconciler

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/sustainable.computing.io/kepler-operator/pkg/utils/k8s"
	"github.com/sustainable.computing.io/kepler-operator/pkg/utils/test"
	"golang.org/x/net/context"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestDeleterReconcile(t *testing.T) {
	dep := k8s.Deployment("ns", "name").Build()
	c := fake.NewFakeClient(dep)

	tt := []struct {
		scenario string
		resource client.Object
	}{
		{"deletes existing resources", dep},
		{"deletes non-existent resources", k8s.Deployment("ns", "non-existent").Build()},
	}

	for _, tc := range tt {
		tc := tc
		t.Run(tc.scenario, func(t *testing.T) {
			f := test.NewFramework(t, test.WithClient(c))
			deleter := Deleter{Resource: tc.resource}
			result := deleter.Reconcile(context.TODO(), c, f.Scheme())
			assert.Exactly(t, Continue, result.Action)
			assert.NoError(t, result.Error)

			dummy := tc.resource.DeepCopyObject().(client.Object)
			err := c.Get(context.TODO(), client.ObjectKeyFromObject(tc.resource), dummy)
			assert.ErrorContains(t, err, fmt.Sprintf(`"%s" not found`, tc.resource.GetName()))
		})
	}
}
