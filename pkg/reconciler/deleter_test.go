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

	t.Run("returns Continue when deleting existing resource", func(t *testing.T) {
		dep := k8s.Deployment("ns", "existing").Build()
		c := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(dep).Build()

		deleter := Deleter{Resource: dep}
		result := deleter.Reconcile(context.TODO(), c, testScheme)

		// Non-blocking: returns Continue to proceed with next reconciler
		assert.Exactly(t, Continue, result.Action)
		assert.NoError(t, result.Error)

		// Resource should be marked for deletion (fake client deletes immediately)
		dummy := dep.DeepCopyObject().(client.Object)
		err := c.Get(context.TODO(), client.ObjectKeyFromObject(dep), dummy)
		assert.ErrorContains(t, err, fmt.Sprintf(`"%s" not found`, dep.GetName()))
	})

	t.Run("returns Continue when resource already deleted", func(t *testing.T) {
		nonExistent := k8s.Deployment("ns", "non-existent").Build()
		c := fake.NewClientBuilder().WithScheme(testScheme).Build()

		deleter := Deleter{Resource: nonExistent}
		result := deleter.Reconcile(context.TODO(), c, testScheme)

		// Resource already gone, no requeue needed
		assert.Exactly(t, Continue, result.Action)
		assert.NoError(t, result.Error)
	})

	t.Run("multiple deleters execute in sequence", func(t *testing.T) {
		dep1 := k8s.Deployment("ns", "dep1").Build()
		dep2 := k8s.Deployment("ns", "dep2").Build()
		c := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(dep1, dep2).Build()

		// Both deleters return Continue, allowing sequential execution
		deleter1 := Deleter{Resource: dep1}
		result := deleter1.Reconcile(context.TODO(), c, testScheme)
		assert.Exactly(t, Continue, result.Action)
		assert.NoError(t, result.Error)

		deleter2 := Deleter{Resource: dep2}
		result = deleter2.Reconcile(context.TODO(), c, testScheme)
		assert.Exactly(t, Continue, result.Action)
		assert.NoError(t, result.Error)
	})
}
