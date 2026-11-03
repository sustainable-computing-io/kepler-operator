// SPDX-FileCopyrightText: 2025 The Kepler Authors
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sustainable.computing.io/kepler-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestPowerMonitorDeletionWaitsForInternal(t *testing.T) {
	ctx := context.Background()
	scheme := runtime.NewScheme()
	require.NoError(t, v1alpha1.AddToScheme(scheme))
	tests := []struct {
		name  string
		funcs interceptor.Funcs
	}{
		{name: "internal deletion pending"},
		{name: "delete failed", funcs: interceptor.Funcs{
			Delete: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.DeleteOption) error {
				return assert.AnError
			},
		}},
		{name: "completion check failed", funcs: interceptor.Funcs{
			Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
				if _, ok := obj.(*v1alpha1.PowerMonitorInternal); ok {
					return assert.AnError
				}
				return c.Get(ctx, key, obj, opts...)
			},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := metav1.Now()
			pm := &v1alpha1.PowerMonitor{ObjectMeta: metav1.ObjectMeta{
				Name: "power-monitor", DeletionTimestamp: &now, Finalizers: []string{Finalizer},
			}}
			pmi := &v1alpha1.PowerMonitorInternal{ObjectMeta: metav1.ObjectMeta{
				Name: pm.Name, Finalizers: []string{Finalizer},
			}}
			c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(pm, pmi).Build()
			r := PowerMonitorReconciler{Client: interceptor.NewClient(c, tt.funcs), Scheme: scheme, logger: logr.Discard()}

			result, err := r.runPowerMonitorReconcilers(ctx, pm)
			require.NoError(t, err)
			assert.Equal(t, 5*time.Second, result.RequeueAfter)
			require.NoError(t, c.Get(ctx, client.ObjectKeyFromObject(pm), pm))
			assert.Contains(t, pm.Finalizers, Finalizer)
			require.NoError(t, c.Get(ctx, client.ObjectKeyFromObject(pmi), pmi))

			// Complete internal cleanup and clear API errors before reconciling again.
			pmi.Finalizers = nil
			require.NoError(t, c.Update(ctx, pmi))
			r.Client = c
			_, err = r.runPowerMonitorReconcilers(ctx, pm)
			require.NoError(t, err)
			assert.True(t, errors.IsNotFound(c.Get(ctx, client.ObjectKeyFromObject(pm), pm)))
		})
	}
}
