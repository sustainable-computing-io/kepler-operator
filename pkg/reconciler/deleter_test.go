// SPDX-FileCopyrightText: 2025 The Kepler Authors
// SPDX-License-Identifier: Apache-2.0

package reconciler

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sustainable.computing.io/kepler-operator/pkg/utils/k8s"
	"k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestDeleterReconcile(t *testing.T) {
	tests := []struct {
		name            string
		exists          bool
		waitForDeletion bool
	}{
		{name: "existing resource", exists: true},
		{name: "missing resource"},
		{name: "existing resource with completion check", exists: true, waitForDeletion: true},
		{name: "missing resource with completion check", waitForDeletion: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			scheme, c := testSetup(t)
			dep := k8s.Deployment("ns", "name").Build()
			if tt.exists {
				require.NoError(t, c.Create(ctx, dep))
			}
			deleter := Deleter{Resource: dep, WaitForDeletion: tt.waitForDeletion}
			result := deleter.Reconcile(ctx, c, scheme)
			assert.Equal(t, Continue, result.Action)
			assert.NoError(t, result.Error)
			assert.True(t, errors.IsNotFound(c.Get(ctx, client.ObjectKeyFromObject(dep), dep)))
		})
	}
}

func TestDeleterPendingDeletion(t *testing.T) {
	tests := []struct {
		name            string
		waitForDeletion bool
		wantAction      Action
	}{
		{name: "continue after requesting deletion", wantAction: Continue},
		{name: "requeue until deletion completes", waitForDeletion: true, wantAction: Requeue},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			scheme, c := testSetup(t)
			dep := k8s.Deployment("ns", "name").Build()
			dep.Finalizers = []string{"test.example.com/finalizer"}
			require.NoError(t, c.Create(ctx, dep))
			deleter := Deleter{Resource: dep, WaitForDeletion: tt.waitForDeletion}

			result := deleter.Reconcile(ctx, c, scheme)
			assert.Equal(t, tt.wantAction, result.Action)
			assert.NoError(t, result.Error)
			require.NoError(t, c.Get(ctx, client.ObjectKeyFromObject(dep), dep))
			assert.False(t, dep.DeletionTimestamp.IsZero())

			dep.Finalizers = nil
			require.NoError(t, c.Update(ctx, dep))
			result = deleter.Reconcile(ctx, c, scheme)
			assert.Equal(t, Continue, result.Action)
			assert.NoError(t, result.Error)
		})
	}
}

func TestDeleterErrors(t *testing.T) {
	tests := []struct {
		name  string
		funcs interceptor.Funcs
	}{
		{
			name: "failed to delete",
			funcs: interceptor.Funcs{Delete: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.DeleteOption) error {
				return assert.AnError
			}},
		},
		{
			name: "failed to check deletion",
			funcs: interceptor.Funcs{Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
				return assert.AnError
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme, c := testSetup(t)
			dep := k8s.Deployment("ns", "name").Build()
			require.NoError(t, c.Create(context.Background(), dep))
			deleter := Deleter{Resource: dep, OnError: Requeue, WaitForDeletion: true}
			result := deleter.Reconcile(context.Background(), interceptor.NewClient(c.(client.WithWatch), tt.funcs), scheme)
			assert.Equal(t, Requeue, result.Action)
			assert.ErrorIs(t, result.Error, assert.AnError)
			assert.ErrorContains(t, result.Error, tt.name)
		})
	}
}

func TestDeletersRunInSinglePass(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	scheme, c := testSetup(t)
	dep1 := k8s.Deployment("ns", "dep1").Build()
	dep2 := k8s.Deployment("ns", "dep2").Build()
	for _, dep := range []client.Object{dep1, dep2} {
		dep.SetFinalizers([]string{"test.example.com/finalizer"})
		require.NoError(t, c.Create(ctx, dep))
	}
	runner := createRunner([]Reconciler{Deleter{Resource: dep1}, Deleter{Resource: dep2}}, c, scheme)
	result, err := runner.Run(ctx)
	assert.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)
	for _, dep := range []client.Object{dep1, dep2} {
		require.NoError(t, c.Get(ctx, client.ObjectKeyFromObject(dep), dep))
		assert.False(t, dep.GetDeletionTimestamp().IsZero())
	}
}
