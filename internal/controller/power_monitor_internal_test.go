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
	"github.com/sustainable.computing.io/kepler-operator/pkg/components"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestPowerMonitorInternalDeletionWaitsForNamespace(t *testing.T) {
	ctx := context.Background()
	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, v1alpha1.AddToScheme(scheme))
	now := metav1.Now()
	pmi := &v1alpha1.PowerMonitorInternal{
		ObjectMeta: metav1.ObjectMeta{
			Name: "power-monitor", DeletionTimestamp: &now, Finalizers: []string{Finalizer},
		},
		Spec: v1alpha1.PowerMonitorInternalSpec{Kepler: v1alpha1.PowerMonitorInternalKeplerSpec{
			Deployment: v1alpha1.PowerMonitorInternalKeplerDeploymentSpec{Namespace: "power-monitor"},
		}},
	}
	ns := components.NewNamespace(pmi.Namespace())
	ns.Finalizers = []string{"test.example.com/finalizer"}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(pmi, ns).Build()
	r := PowerMonitorInternalReconciler{Client: c, Scheme: scheme, logger: logr.Discard()}

	result, err := r.runPowerMonitorReconcilers(ctx, pmi)
	require.NoError(t, err)
	assert.Equal(t, 5*time.Second, result.RequeueAfter)
	require.NoError(t, c.Get(ctx, client.ObjectKeyFromObject(pmi), pmi))
	assert.Contains(t, pmi.Finalizers, Finalizer)
	require.NoError(t, c.Get(ctx, client.ObjectKeyFromObject(ns), ns))
	assert.False(t, ns.DeletionTimestamp.IsZero())

	ns.Finalizers = nil
	require.NoError(t, c.Update(ctx, ns))
	_, err = r.runPowerMonitorReconcilers(ctx, pmi)
	require.NoError(t, err)
	assert.True(t, errors.IsNotFound(c.Get(ctx, client.ObjectKeyFromObject(pmi), pmi)))
}
