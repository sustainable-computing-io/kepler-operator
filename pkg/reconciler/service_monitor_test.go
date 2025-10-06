// SPDX-FileCopyrightText: 2025 The Kepler Authors
// SPDX-License-Identifier: Apache-2.0

package reconciler

import (
	"context"
	"testing"

	monv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/stretchr/testify/assert"
	"github.com/sustainable.computing.io/kepler-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// serviceMonitorTestScheme creates a runtime scheme for ServiceMonitor testing
func serviceMonitorTestScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)
	_ = v1alpha1.AddToScheme(scheme)
	_ = monv1.AddToScheme(scheme)
	return scheme
}

// serviceMonitorTestPMI creates a minimal PowerMonitorInternal for testing
func serviceMonitorTestPMI() *v1alpha1.PowerMonitorInternal {
	return &v1alpha1.PowerMonitorInternal{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pmi",
			Namespace: "test-ns",
		},
		Spec: v1alpha1.PowerMonitorInternalSpec{
			Kepler: v1alpha1.PowerMonitorInternalKeplerSpec{
				Deployment: v1alpha1.PowerMonitorInternalKeplerDeploymentSpec{
					Namespace: "test-ns",
					Image:     "test-image",
				},
			},
		},
	}
}

// serviceMonitorMockClient simulates client behavior for testing
type serviceMonitorMockClient struct {
	client.Client
	failDelete bool
	failPatch  bool
}

func (m *serviceMonitorMockClient) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	if m.failDelete {
		return errors.NewInternalError(assert.AnError)
	}
	return nil
}

func (m *serviceMonitorMockClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	if m.failPatch {
		return errors.NewInternalError(assert.AnError)
	}
	// Simulate successful patch by creating the object
	return m.Create(ctx, obj)
}

func TestPowerMonitorServiceMonitorReconciler(t *testing.T) {
	scheme := serviceMonitorTestScheme()
	pmi := serviceMonitorTestPMI()

	// Create test ServiceMonitor
	testSM := &monv1.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sm",
			Namespace: "test-ns",
		},
	}

	tests := []struct {
		name       string
		enableRBAC bool
		enableUWM  bool
		wantDelete bool
	}{
		{"RBAC disabled, UWM enabled", false, true, false},
		{"RBAC enabled, UWM disabled", true, false, true},
		{"both disabled", false, false, false},
		{"both enabled", true, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseClient := fake.NewClientBuilder().WithScheme(scheme).Build()
			client := &serviceMonitorMockClient{Client: baseClient}

			reconciler := PowerMonitorServiceMonitorReconciler{
				Pmi:        pmi,
				Sm:         testSM,
				EnableRBAC: tt.enableRBAC,
				EnableUWM:  tt.enableUWM,
			}

			result := reconciler.Reconcile(context.TODO(), client, scheme)

			assert.Equal(t, Continue, result.Action)
			if tt.wantDelete {
				// Delete operations should succeed
				assert.NoError(t, result.Error)
			}
		})
	}
}
