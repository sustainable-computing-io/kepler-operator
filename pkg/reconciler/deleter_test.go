// SPDX-FileCopyrightText: 2025 The Kepler Authors
// SPDX-License-Identifier: Apache-2.0

package reconciler

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sustainable.computing.io/kepler-operator/pkg/utils/k8s"
	"golang.org/x/net/context"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestDeleterReconcile(t *testing.T) {
	// Set up scheme with the types we need
	testScheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(testScheme))
	require.NoError(t, corev1.AddToScheme(testScheme))
	require.NoError(t, appsv1.AddToScheme(testScheme))

	dep := k8s.Deployment("ns", "name").Build()
	c := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(dep).Build()

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
			deleter := Deleter{Resource: tc.resource}
			result := deleter.Reconcile(context.TODO(), c, testScheme)
			assert.Exactly(t, Continue, result.Action)
			assert.NoError(t, result.Error)

			dummy := tc.resource.DeepCopyObject().(client.Object)
			err := c.Get(context.TODO(), client.ObjectKeyFromObject(tc.resource), dummy)
			assert.ErrorContains(t, err, fmt.Sprintf(`"%s" not found`, tc.resource.GetName()))
		})
	}
}
