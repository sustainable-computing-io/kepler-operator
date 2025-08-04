// SPDX-FileCopyrightText: 2025 The Kepler Authors
// SPDX-License-Identifier: Apache-2.0

package reconciler

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/sustainable.computing.io/kepler-operator/api/v1alpha1"
	"github.com/sustainable.computing.io/kepler-operator/pkg/utils/k8s"
	appsv1 "k8s.io/api/apps/v1"
	authv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// Test setup helpers

// createTestScheme creates a scheme with all necessary types for security tests
func createSecurityTestScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)
	_ = authv1.AddToScheme(scheme)
	_ = v1alpha1.AddToScheme(scheme)
	return scheme
}

// createTestPMI creates a standard test PowerMonitorInternal object
func createTestPMI() *v1alpha1.PowerMonitorInternal {
	return &v1alpha1.PowerMonitorInternal{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pmi",
			Namespace: "test-ns",
			UID:       "test-uid",
		},
		Spec: v1alpha1.PowerMonitorInternalSpec{
			Kepler: v1alpha1.PowerMonitorInternalKeplerSpec{
				Deployment: v1alpha1.PowerMonitorInternalKeplerDeploymentSpec{
					PowerMonitorKeplerDeploymentSpec: v1alpha1.PowerMonitorKeplerDeploymentSpec{
						Security: v1alpha1.PowerMonitorKeplerDeploymentSecuritySpec{
							AllowedSANames: []string{"test-sa"},
							Mode:           v1alpha1.SecurityModeRBAC,
						},
					},
					Namespace: "test-ns",
				},
			},
		},
	}
}

// Mock clients for different scenarios

// mockClientBuilder helps build mock clients for various scenarios
type mockClientBuilder struct {
	scheme       *runtime.Scheme
	objects      []client.Object
	getError     map[string]error // key format: "kind/namespace/name"
	createError  map[string]error
	updateError  map[string]error
	patchError   map[string]error
	tokenRequest *authv1.TokenRequest
}

func newMockClientBuilder() *mockClientBuilder {
	return &mockClientBuilder{
		scheme:      createSecurityTestScheme(),
		getError:    make(map[string]error),
		createError: make(map[string]error),
		updateError: make(map[string]error),
		patchError:  make(map[string]error),
	}
}

func (b *mockClientBuilder) withObjects(objs ...client.Object) *mockClientBuilder {
	b.objects = append(b.objects, objs...)
	return b
}

func (b *mockClientBuilder) withPatchError(kind, namespace, name string, err error) *mockClientBuilder {
	key := fmt.Sprintf("%s/%s/%s", kind, namespace, name)
	b.patchError[key] = err
	return b
}

func (b *mockClientBuilder) withTokenRequest(token string) *mockClientBuilder {
	b.tokenRequest = &authv1.TokenRequest{
		Status: authv1.TokenRequestStatus{
			Token: token,
		},
	}
	return b
}

func (b *mockClientBuilder) build() client.Client {
	baseClient := fake.NewClientBuilder().WithScheme(b.scheme).WithObjects(b.objects...).Build()
	return &mockClient{
		Client:       baseClient,
		getError:     b.getError,
		patchError:   b.patchError,
		tokenRequest: b.tokenRequest,
	}
}

type mockClient struct {
	client.Client
	getError     map[string]error
	patchError   map[string]error
	tokenRequest *authv1.TokenRequest
}

func (m *mockClient) Get(ctx context.Context, key types.NamespacedName, obj client.Object, opts ...client.GetOption) error {
	errorKey := fmt.Sprintf("%s/%s/%s", obj.GetObjectKind().GroupVersionKind().Kind, key.Namespace, key.Name)
	if err, exists := m.getError[errorKey]; exists {
		return err
	}
	return m.Client.Get(ctx, key, obj, opts...)
}

func (m *mockClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	// Determine object kind from the object type since GVK might not be set in tests
	var kind string
	switch obj.(type) {
	case *corev1.Secret:
		kind = "Secret"
	case *corev1.ConfigMap:
		kind = "ConfigMap"
	case *corev1.ServiceAccount:
		kind = "ServiceAccount"
	}

	errorKey := fmt.Sprintf("%s/%s/%s", kind, obj.GetNamespace(), obj.GetName())
	if err, exists := m.patchError[errorKey]; exists {
		return err
	}
	// For patch operations that would fail in fake client, return success for testing
	return nil
}

func (m *mockClient) SubResource(subresource string) client.SubResourceClient {
	return &mockSubResourceClient{
		SubResourceClient: m.Client.SubResource(subresource),
		tokenRequest:      m.tokenRequest,
	}
}

type mockSubResourceClient struct {
	client.SubResourceClient
	tokenRequest *authv1.TokenRequest
}

func (m *mockSubResourceClient) Create(ctx context.Context, obj client.Object, subResource client.Object, opts ...client.SubResourceCreateOption) error {
	if tokenReq, ok := subResource.(*authv1.TokenRequest); ok && m.tokenRequest != nil {
		tokenReq.Status = m.tokenRequest.Status
		return nil
	}
	return errors.NewBadRequest("token request failed")
}

// Test cases for reconcilers

func TestKubeRBACProxyConfigReconciler(t *testing.T) {
	tests := []struct {
		name           string
		enableRBAC     bool
		enableUWM      bool
		mockClient     func() client.Client
		expectedAction Action
		expectedError  bool
		errorContains  string
	}{
		{
			name:       "RBAC disabled - should call deleter",
			enableRBAC: false,
			enableUWM:  false,
			mockClient: func() client.Client {
				return newMockClientBuilder().build()
			},
			expectedAction: Continue,
			expectedError:  false,
		},
		{
			name:       "RBAC enabled - should call updater successfully",
			enableRBAC: true,
			enableUWM:  false,
			mockClient: func() client.Client {
				return newMockClientBuilder().build()
			},
			expectedAction: Continue,
			expectedError:  false,
		},
		{
			name:       "RBAC enabled but patch fails",
			enableRBAC: true,
			enableUWM:  false,
			mockClient: func() client.Client {
				return newMockClientBuilder().
					withPatchError("Secret", "test-ns", "power-monitor-kube-rbac-proxy-config", errors.NewInternalError(fmt.Errorf("patch failed"))).
					build()
			},
			expectedAction: Continue,
			expectedError:  true,
			errorContains:  "patch failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pmi := createTestPMI()
			client := tt.mockClient()
			scheme := createSecurityTestScheme()

			reconciler := KubeRBACProxyConfigReconciler{
				Pmi:        pmi,
				EnableRBAC: tt.enableRBAC,
				EnableUWM:  tt.enableUWM,
			}

			result := reconciler.Reconcile(context.TODO(), client, scheme)

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

func TestCABundleConfigReconciler(t *testing.T) {
	tests := []struct {
		name           string
		enableRBAC     bool
		enableUWM      bool
		mockClient     func() client.Client
		expectedAction Action
		expectedError  bool
		errorContains  string
	}{
		{
			name:       "RBAC disabled - should call deleter",
			enableRBAC: false,
			enableUWM:  true, // UWM enabled but RBAC disabled should still call deleter
			mockClient: func() client.Client {
				return newMockClientBuilder().build()
			},
			expectedAction: Continue,
			expectedError:  false,
		},
		{
			name:       "UWM disabled - should call deleter",
			enableRBAC: true, // RBAC enabled but UWM disabled should call deleter
			enableUWM:  false,
			mockClient: func() client.Client {
				return newMockClientBuilder().build()
			},
			expectedAction: Continue,
			expectedError:  false,
		},
		{
			name:       "Both RBAC and UWM disabled - should call deleter",
			enableRBAC: false,
			enableUWM:  false,
			mockClient: func() client.Client {
				return newMockClientBuilder().build()
			},
			expectedAction: Continue,
			expectedError:  false,
		},
		{
			name:       "RBAC and UWM enabled - should call updater successfully",
			enableRBAC: true,
			enableUWM:  true,
			mockClient: func() client.Client {
				return newMockClientBuilder().build()
			},
			expectedAction: Continue,
			expectedError:  false,
		},
		{
			name:       "RBAC and UWM enabled but patch fails",
			enableRBAC: true,
			enableUWM:  true,
			mockClient: func() client.Client {
				return newMockClientBuilder().
					withPatchError("ConfigMap", "test-ns", "power-monitor-serving-certs-ca-bundle", errors.NewInternalError(fmt.Errorf("patch failed"))).
					build()
			},
			expectedAction: Continue,
			expectedError:  true,
			errorContains:  "patch failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pmi := createTestPMI()
			client := tt.mockClient()
			scheme := createSecurityTestScheme()

			reconciler := CABundleConfigReconciler{
				Pmi:        pmi,
				EnableRBAC: tt.enableRBAC,
				EnableUWM:  tt.enableUWM,
			}

			result := reconciler.Reconcile(context.TODO(), client, scheme)

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

func TestUWMSecretTokenReconciler(t *testing.T) {
	// Create test ServiceAccount that UWM reconciler expects
	promSA := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "prometheus-user-workload",
			Namespace: "openshift-user-workload-monitoring",
		},
	}

	tests := []struct {
		name           string
		enableRBAC     bool
		enableUWM      bool
		cluster        k8s.Cluster
		mockClient     func() client.Client
		expectedAction Action
		expectedError  bool
		errorContains  string
	}{
		{
			name:       "RBAC disabled - should call deleter",
			enableRBAC: false,
			enableUWM:  false,
			cluster:    k8s.Kubernetes,
			mockClient: func() client.Client {
				return newMockClientBuilder().build()
			},
			expectedAction: Continue,
			expectedError:  false,
		},
		{
			name:       "UWM disabled - should call deleter",
			enableRBAC: true,
			enableUWM:  false,
			cluster:    k8s.Kubernetes,
			mockClient: func() client.Client {
				return newMockClientBuilder().build()
			},
			expectedAction: Continue,
			expectedError:  false,
		},
		{
			name:       "RBAC and UWM enabled with existing SA - should succeed",
			enableRBAC: true,
			enableUWM:  true,
			cluster:    k8s.Kubernetes,
			mockClient: func() client.Client {
				return newMockClientBuilder().
					withObjects(promSA).
					withTokenRequest("test-token").
					build()
			},
			expectedAction: Continue,
			expectedError:  false,
		},
		{
			name:       "UWM service account not found",
			enableRBAC: true,
			enableUWM:  true,
			cluster:    k8s.Kubernetes,
			mockClient: func() client.Client {
				return newMockClientBuilder().build() // No SA created
			},
			expectedAction: Stop,
			expectedError:  true,
			errorContains:  "missing \"prometheus-user-workload\" in \"openshift-user-workload-monitoring\" namespace yet",
		},
		{
			name:       "OpenShift cluster with longer timeout",
			enableRBAC: true,
			enableUWM:  true,
			cluster:    k8s.OpenShift,
			mockClient: func() client.Client {
				return newMockClientBuilder().
					withObjects(promSA).
					withTokenRequest("test-token").
					build()
			},
			expectedAction: Continue,
			expectedError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pmi := createTestPMI()
			client := tt.mockClient()
			scheme := createSecurityTestScheme()

			reconciler := UWMSecretTokenReconciler{
				Pmi:        pmi,
				Cluster:    tt.cluster,
				EnableRBAC: tt.enableRBAC,
				EnableUWM:  tt.enableUWM,
			}

			result := reconciler.Reconcile(context.TODO(), client, scheme)

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

func TestKubeRBACProxyObjectsChecker(t *testing.T) {
	// Save original timeouts and restore them after test
	originalOpenshiftTimeout := openshiftTimeout
	originalK8sTimeout := k8sTimeout
	originalRetryDelay := retryDelay

	// Use much shorter timeouts for testing
	openshiftTimeout = 100 * time.Millisecond
	k8sTimeout = 50 * time.Millisecond
	retryDelay = 10 * time.Millisecond

	defer func() {
		// Restore original timeouts
		openshiftTimeout = originalOpenshiftTimeout
		k8sTimeout = originalK8sTimeout
		retryDelay = originalRetryDelay
	}()

	testDS := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-ds",
			Namespace: "test-ns",
		},
	}

	// Test secrets that the checker looks for
	testSecrets := []*corev1.Secret{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "power-monitor-kube-rbac-proxy-config",
				Namespace: "test-ns",
			},
			Data: map[string][]byte{"config.yaml": []byte("test-config")},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "power-monitor-tls",
				Namespace: "test-ns",
			},
			Data: map[string][]byte{"tls.crt": []byte("cert"), "tls.key": []byte("key")},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "prometheus-user-workload-token",
				Namespace: "test-ns",
			},
			Data: map[string][]byte{"token": []byte("test-token")},
		},
	}

	testConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "power-monitor-serving-certs-ca-bundle",
			Namespace: "test-ns",
		},
		Data: map[string]string{"ca-bundle.crt": "test-ca-bundle"},
	}

	tests := []struct {
		name           string
		enableRBAC     bool
		enableUWM      bool
		cluster        k8s.Cluster
		setupObjects   []client.Object
		expectedAction Action
		expectedError  bool
		errorContains  string
	}{
		{
			name:           "RBAC disabled - should skip all checks",
			enableRBAC:     false,
			enableUWM:      false,
			cluster:        k8s.Kubernetes,
			setupObjects:   []client.Object{},
			expectedAction: Continue,
			expectedError:  false,
		},
		{
			name:       "RBAC enabled - all required objects present (Kubernetes)",
			enableRBAC: true,
			enableUWM:  false,
			cluster:    k8s.Kubernetes,
			setupObjects: []client.Object{
				testSecrets[0], // rbac config secret
				testSecrets[1], // tls secret
			},
			expectedAction: Continue,
			expectedError:  false,
		},
		{
			name:       "RBAC and UWM enabled - all objects present (OpenShift)",
			enableRBAC: true,
			enableUWM:  true,
			cluster:    k8s.OpenShift,
			setupObjects: []client.Object{
				testSecrets[0], // rbac config secret
				testSecrets[1], // tls secret
				testSecrets[2], // uwm token secret
				testConfigMap,  // ca bundle
			},
			expectedAction: Continue,
			expectedError:  false,
		},
		{
			name:           "RBAC config secret missing",
			enableRBAC:     true,
			enableUWM:      false,
			cluster:        k8s.Kubernetes,
			setupObjects:   []client.Object{testSecrets[1]}, // only tls secret
			expectedAction: Stop,
			expectedError:  true,
			errorContains:  "power-monitor-kube-rbac-proxy-config",
		},
		{
			name:           "TLS secret missing",
			enableRBAC:     true,
			enableUWM:      false,
			cluster:        k8s.Kubernetes,
			setupObjects:   []client.Object{testSecrets[0]}, // only rbac config secret
			expectedAction: Stop,
			expectedError:  true,
			errorContains:  "power-monitor-tls",
		},
		{
			name:       "UWM enabled but CA bundle missing",
			enableRBAC: true,
			enableUWM:  true,
			cluster:    k8s.OpenShift,
			setupObjects: []client.Object{
				testSecrets[0], // rbac config secret
				testSecrets[1], // tls secret
				testSecrets[2], // uwm token secret
				// missing ca bundle configmap
			},
			expectedAction: Stop,
			expectedError:  true,
			errorContains:  "power-monitor-serving-certs-ca-bundle",
		},
		{
			name:       "UWM enabled but token secret missing",
			enableRBAC: true,
			enableUWM:  true,
			cluster:    k8s.OpenShift,
			setupObjects: []client.Object{
				testSecrets[0], // rbac config secret
				testSecrets[1], // tls secret
				testConfigMap,  // ca bundle
				// missing uwm token secret
			},
			expectedAction: Stop,
			expectedError:  true,
			errorContains:  "prometheus-user-workload-token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pmi := createTestPMI()
			client := newMockClientBuilder().withObjects(tt.setupObjects...).build()
			scheme := createSecurityTestScheme()

			reconciler := KubeRBACProxyObjectsChecker{
				Pmi:        pmi,
				Cluster:    tt.cluster,
				Ds:         testDS,
				EnableRBAC: tt.enableRBAC,
				EnableUWM:  tt.enableUWM,
			}

			result := reconciler.Reconcile(context.TODO(), client, scheme)

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
