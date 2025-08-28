// SPDX-FileCopyrightText: 2025 The Kepler Authors
// SPDX-License-Identifier: Apache-2.0

package reconciler

import (
	"context"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// Test helpers and utilities

// testSetup creates a common test environment
func testSetup(t *testing.T) (*runtime.Scheme, client.Client) {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	client := fake.NewClientBuilder().WithScheme(scheme).Build()
	return scheme, client
}

// createRunner creates a Runner with the given reconcilers
func createRunner(reconcilers []Reconciler, client client.Client, scheme *runtime.Scheme) Runner {
	return Runner{
		Reconcilers: reconcilers,
		Client:      client,
		Scheme:      scheme,
		Logger:      logr.Discard(),
	}
}

// mockReconciler represents a test reconciler that returns a predefined result
type mockReconciler struct {
	result Result
	called bool
}

func (m *mockReconciler) Reconcile(ctx context.Context, c client.Client, scheme *runtime.Scheme) Result {
	m.called = true
	return m.result
}

// newMockReconciler creates a new mock reconciler with the given result
func newMockReconciler(action Action, err error) *mockReconciler {
	return &mockReconciler{
		result: Result{Action: action, Error: err},
	}
}

// Test cases

func TestRunner_Run(t *testing.T) {
	tests := []struct {
		name               string
		reconcilers        []Reconciler
		expectedResult     ctrl.Result
		expectedError      bool
		validateReconciler func(t *testing.T, reconcilers []Reconciler)
	}{
		{
			name:           "no reconcilers",
			reconcilers:    []Reconciler{},
			expectedResult: ctrl.Result{},
			expectedError:  false,
		},
		{
			name: "reconciler - continue without error",
			reconcilers: []Reconciler{
				newMockReconciler(Continue, nil),
			},
			expectedResult: ctrl.Result{},
			expectedError:  false,
			validateReconciler: func(t *testing.T, reconcilers []Reconciler) {
				mock := reconcilers[0].(*mockReconciler)
				assert.True(t, mock.called, "reconciler should be called")
			},
		},
		{
			name: "reconciler - continue with error",
			reconcilers: []Reconciler{
				newMockReconciler(Continue, assert.AnError),
			},
			expectedResult: ctrl.Result{},
			expectedError:  true,
			validateReconciler: func(t *testing.T, reconcilers []Reconciler) {
				mock := reconcilers[0].(*mockReconciler)
				assert.True(t, mock.called, "reconciler should be called")
			},
		},
		{
			name: "reconciler - stop without error (triggers requeue)",
			reconcilers: []Reconciler{
				newMockReconciler(Stop, nil),
			},
			expectedResult: ctrl.Result{Requeue: true},
			expectedError:  false,
		},
		{
			name: "reconciler - stop with error (no requeue)",
			reconcilers: []Reconciler{
				newMockReconciler(Stop, assert.AnError),
			},
			expectedResult: ctrl.Result{Requeue: false},
			expectedError:  true,
		},
		{
			name: "reconciler - requeue without error",
			reconcilers: []Reconciler{
				newMockReconciler(Requeue, nil),
			},
			expectedResult: ctrl.Result{RequeueAfter: 5 * time.Second},
			expectedError:  false,
		},
		{
			name: "reconciler - requeue with error",
			reconcilers: []Reconciler{
				newMockReconciler(Requeue, assert.AnError),
			},
			expectedResult: ctrl.Result{RequeueAfter: 5 * time.Second},
			expectedError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			scheme, client := testSetup(t)
			runner := createRunner(tt.reconcilers, client, scheme)

			result, err := runner.Run(context.TODO())

			assert.Equal(t, tt.expectedResult, result, "unexpected result")
			if tt.expectedError {
				assert.Error(t, err, "expected an error")
			} else {
				assert.NoError(t, err, "expected no error")
			}

			if tt.validateReconciler != nil {
				tt.validateReconciler(t, tt.reconcilers)
			}
		})
	}
}
