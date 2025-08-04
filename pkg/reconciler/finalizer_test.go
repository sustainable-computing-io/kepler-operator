// SPDX-FileCopyrightText: 2025 The Kepler Authors
// SPDX-License-Identifier: Apache-2.0

package reconciler

import (
	"context"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/sustainable.computing.io/kepler-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	ctrlutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const testFinalizerName = "test.finalizer/test"

// Mock clients for error testing

// mockClientWithGetError simulates Get operation failures
type mockClientWithGetError struct {
	client.Client
}

func (m *mockClientWithGetError) Get(ctx context.Context, key types.NamespacedName, obj client.Object, opts ...client.GetOption) error {
	return errors.NewInternalError(assert.AnError)
}

// mockClientWithUpdateError simulates Update operation failures
type mockClientWithUpdateError struct {
	client.Client
}

func (m *mockClientWithUpdateError) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	return errors.NewInternalError(assert.AnError)
}

// createTestScheme creates and returns a runtime scheme with required types
func createTestScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)
	_ = v1alpha1.AddToScheme(scheme)
	return scheme
}

// createTestConfigMap creates a test ConfigMap with the given name and namespace
func createTestConfigMap(name, namespace string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}

// createTestPowerMonitor creates a test PowerMonitorInternal with the given name and namespace
func createTestPowerMonitor(name, namespace string) *v1alpha1.PowerMonitorInternal {
	return &v1alpha1.PowerMonitorInternal{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}

func TestFinalizerReconcile(t *testing.T) {
	scheme := createTestScheme()

	tests := []struct {
		name            string
		setupObject     func() client.Object
		expectedAction  Action
		expectedError   bool
		expectFinalizer bool
	}{
		{
			name: "add finalizer to new ConfigMap",
			setupObject: func() client.Object {
				return createTestConfigMap("test-configmap", "test-ns")
			},
			expectedAction:  Stop,
			expectedError:   false,
			expectFinalizer: true,
		},
		{
			name: "add finalizer to new PowerMonitorInternal",
			setupObject: func() client.Object {
				return createTestPowerMonitor("test-pmi", "test-ns")
			},
			expectedAction:  Stop,
			expectedError:   false,
			expectFinalizer: true,
		},
		{
			name: "object already has finalizer - no action needed",
			setupObject: func() client.Object {
				obj := createTestConfigMap("test-configmap", "test-ns")
				ctrlutil.AddFinalizer(obj, testFinalizerName)
				return obj
			},
			expectedAction:  Continue,
			expectedError:   false,
			expectFinalizer: true,
		},
		{
			name: "remove finalizer from deleted object",
			setupObject: func() client.Object {
				obj := createTestPowerMonitor("test-pmi", "test-ns")
				now := metav1.Time{Time: time.Now()}
				obj.DeletionTimestamp = &now
				obj.Finalizers = []string{testFinalizerName}
				return obj
			},
			expectedAction:  Stop,
			expectedError:   false,
			expectFinalizer: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := tt.setupObject()
			testClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(obj).Build()

			finalizer := Finalizer{
				Resource:  obj.DeepCopyObject().(client.Object),
				Finalizer: testFinalizerName,
				Logger:    logr.Discard(),
			}

			result := finalizer.Reconcile(context.TODO(), testClient, scheme)

			assert.Equal(t, tt.expectedAction, result.Action)
			if tt.expectedError {
				assert.Error(t, result.Error)
			} else {
				assert.NoError(t, result.Error)
			}

			// Verify finalizer state in the cluster
			// Note: For deleted objects, after finalizer removal, the object might be gone from the fake client
			retrieved := obj.DeepCopyObject().(client.Object)
			key := types.NamespacedName{
				Name:      obj.GetName(),
				Namespace: obj.GetNamespace(),
			}
			err := testClient.Get(context.TODO(), key, retrieved)

			// Handle deleted objects that may no longer exist in the fake client
			if err != nil && errors.IsNotFound(err) && !obj.GetDeletionTimestamp().IsZero() {
				// Object was deleted after finalizer removal - this is expected behavior
				assert.False(t, tt.expectFinalizer, "object should be deleted when finalizer is removed")
			} else {
				assert.NoError(t, err)
				hasFinalizer := ctrlutil.ContainsFinalizer(retrieved, testFinalizerName)
				assert.Equal(t, tt.expectFinalizer, hasFinalizer)
			}
		})
	}
}

func TestFinalizerEdgeCases(t *testing.T) {
	scheme := createTestScheme()

	tests := []struct {
		name           string
		setupClient    func() client.Client
		setupObject    func() client.Object
		expectedAction Action
		expectedError  bool
		errorContains  string
	}{
		{
			name: "object not found in cluster",
			setupClient: func() client.Client {
				return fake.NewClientBuilder().WithScheme(scheme).Build() // Empty client
			},
			setupObject: func() client.Object {
				return createTestConfigMap("non-existent", "test-ns")
			},
			expectedAction: Continue,
			expectedError:  false,
		},
		{
			name: "Get operation fails with internal error",
			setupClient: func() client.Client {
				return &mockClientWithGetError{
					Client: fake.NewClientBuilder().WithScheme(scheme).Build(),
				}
			},
			setupObject: func() client.Object {
				return createTestConfigMap("test-configmap", "test-ns")
			},
			expectedAction: Requeue,
			expectedError:  true,
			errorContains:  "failed to refresh",
		},
		{
			name: "Update operation fails when adding finalizer",
			setupClient: func() client.Client {
				obj := createTestConfigMap("test-configmap", "test-ns")
				return &mockClientWithUpdateError{
					Client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(obj).Build(),
				}
			},
			setupObject: func() client.Object {
				return createTestConfigMap("test-configmap", "test-ns")
			},
			expectedAction: Stop,
			expectedError:  true,
		},
		{
			name: "Update operation fails when removing finalizer",
			setupClient: func() client.Client {
				obj := createTestPowerMonitor("test-pmi", "test-ns")
				now := metav1.Time{Time: time.Now()}
				obj.DeletionTimestamp = &now
				obj.Finalizers = []string{testFinalizerName}
				return &mockClientWithUpdateError{
					Client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(obj).Build(),
				}
			},
			setupObject: func() client.Object {
				obj := createTestPowerMonitor("test-pmi", "test-ns")
				now := metav1.Time{Time: time.Now()}
				obj.DeletionTimestamp = &now
				obj.Finalizers = []string{testFinalizerName}
				return obj
			},
			expectedAction: Stop,
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testClient := tt.setupClient()
			obj := tt.setupObject()

			finalizer := Finalizer{
				Resource:  obj,
				Finalizer: testFinalizerName,
				Logger:    logr.Discard(),
			}

			result := finalizer.Reconcile(context.TODO(), testClient, scheme)

			assert.Equal(t, tt.expectedAction, result.Action)
			if tt.expectedError {
				assert.Error(t, result.Error)
				if tt.errorContains != "" {
					assert.Contains(t, result.Error.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, result.Error)
			}
		})
	}
}
