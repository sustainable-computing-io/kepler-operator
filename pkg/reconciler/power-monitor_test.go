// SPDX-FileCopyrightText: 2025 The Kepler Authors
// SPDX-License-Identifier: Apache-2.0

package reconciler

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/sustainable.computing.io/kepler-operator/api/v1alpha1"
	powermonitor "github.com/sustainable.computing.io/kepler-operator/pkg/components/power-monitor"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// Test helpers

func testScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)
	_ = v1alpha1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)
	return scheme
}

type testMockClient struct {
	client.Client
	getErrors map[string]error
}

func (m *testMockClient) Get(ctx context.Context, key types.NamespacedName, obj client.Object, opts ...client.GetOption) error {
	errorKey := fmt.Sprintf("%s/%s/%s", obj.GetObjectKind().GroupVersionKind().String(), key.Namespace, key.Name)
	if err, exists := m.getErrors[errorKey]; exists {
		return err
	}
	return m.Client.Get(ctx, key, obj, opts...)
}

func (m *testMockClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	// Simulate server-side apply by creating the object
	return m.Create(ctx, obj)
}

func TestPowerMonitorDeployer_Reconcile(t *testing.T) {
	scheme := testScheme()

	tests := []struct {
		name           string
		pmi            *v1alpha1.PowerMonitorInternal
		setupClient    func() client.Client
		expectedAction Action
		expectedError  bool
		errorContains  string
	}{
		{
			name: "successful reconciliation with no additional configs",
			pmi: &v1alpha1.PowerMonitorInternal{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pmi",
					Namespace: "test-ns",
				},
				Spec: v1alpha1.PowerMonitorInternalSpec{
					Kepler: v1alpha1.PowerMonitorInternalKeplerSpec{
						Config: v1alpha1.PowerMonitorInternalKeplerConfigSpec{
							LogLevel:             "info",
							AdditionalConfigMaps: []v1alpha1.ConfigMapRef{},
						},
						Deployment: v1alpha1.PowerMonitorInternalKeplerDeploymentSpec{
							Image:     "test-image:latest",
							Namespace: "test-ns",
						},
					},
				},
			},
			setupClient: func() client.Client {
				return &testMockClient{
					Client:    fake.NewClientBuilder().WithScheme(scheme).Build(),
					getErrors: make(map[string]error),
				}
			},
			expectedAction: Continue,
			expectedError:  false,
		},
		{
			name: "successful reconciliation with additional configs",
			pmi: &v1alpha1.PowerMonitorInternal{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pmi",
					Namespace: "test-ns",
				},
				Spec: v1alpha1.PowerMonitorInternalSpec{
					Kepler: v1alpha1.PowerMonitorInternalKeplerSpec{
						Config: v1alpha1.PowerMonitorInternalKeplerConfigSpec{
							LogLevel: "info",
							AdditionalConfigMaps: []v1alpha1.ConfigMapRef{
								{Name: "config-1"},
							},
						},
						Deployment: v1alpha1.PowerMonitorInternalKeplerDeploymentSpec{
							Image:     "test-image:latest",
							Namespace: "test-ns",
						},
					},
				},
			},
			setupClient: func() client.Client {
				yamlConfig1 := `
log:
  level: debug
  format: json
monitor:
  interval: 3s
`

				cfm1 := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{Name: "config-1", Namespace: "test-ns"},
					Data: map[string]string{
						powermonitor.KeplerConfigFile: yamlConfig1,
					},
				}
				return &testMockClient{
					Client:    fake.NewClientBuilder().WithScheme(scheme).WithObjects(cfm1).Build(),
					getErrors: make(map[string]error),
				}
			},
			expectedAction: Continue,
			expectedError:  false,
		},
		{
			name: "ignores invalid additional configs",
			pmi: &v1alpha1.PowerMonitorInternal{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pmi",
					Namespace: "test-ns",
				},
				Spec: v1alpha1.PowerMonitorInternalSpec{
					Kepler: v1alpha1.PowerMonitorInternalKeplerSpec{
						Config: v1alpha1.PowerMonitorInternalKeplerConfigSpec{
							LogLevel: "info",
							AdditionalConfigMaps: []v1alpha1.ConfigMapRef{
								{Name: "invalid-config"},
								{Name: "empty-config"},
							},
						},
						Deployment: v1alpha1.PowerMonitorInternalKeplerDeploymentSpec{
							Image:     "test-image:latest",
							Namespace: "test-ns",
						},
					},
				},
			},
			setupClient: func() client.Client {
				invalidCfm := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{Name: "invalid-config", Namespace: "test-ns"},
					Data: map[string]string{
						"other-file.yaml": "some other content",
					},
				}
				emptyCfm := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{Name: "empty-config", Namespace: "test-ns"},
					Data: map[string]string{
						powermonitor.KeplerConfigFile: "",
					},
				}
				return &testMockClient{
					Client:    fake.NewClientBuilder().WithScheme(scheme).WithObjects(invalidCfm, emptyCfm).Build(),
					getErrors: make(map[string]error),
				}
			},
			expectedAction: Continue,
			expectedError:  false,
		},
		{
			name: "fails when additional configmap not found",
			pmi: &v1alpha1.PowerMonitorInternal{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pmi",
					Namespace: "test-ns",
				},
				Spec: v1alpha1.PowerMonitorInternalSpec{
					Kepler: v1alpha1.PowerMonitorInternalKeplerSpec{
						Config: v1alpha1.PowerMonitorInternalKeplerConfigSpec{
							LogLevel: "info",
							AdditionalConfigMaps: []v1alpha1.ConfigMapRef{
								{Name: "missing-config"},
							},
						},
						Deployment: v1alpha1.PowerMonitorInternalKeplerDeploymentSpec{
							Image:     "test-image:latest",
							Namespace: "test-ns",
						},
					},
				},
			},
			setupClient: func() client.Client {
				return &testMockClient{
					Client:    fake.NewClientBuilder().WithScheme(scheme).Build(),
					getErrors: make(map[string]error),
				}
			},
			expectedAction: Stop,
			expectedError:  true,
			errorContains:  "configMap missing-config not found in test-ns namespace",
		},
		{
			name: "fails when client get operation fails",
			pmi: &v1alpha1.PowerMonitorInternal{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pmi",
					Namespace: "test-ns",
				},
				Spec: v1alpha1.PowerMonitorInternalSpec{
					Kepler: v1alpha1.PowerMonitorInternalKeplerSpec{
						Config: v1alpha1.PowerMonitorInternalKeplerConfigSpec{
							LogLevel: "info",
							AdditionalConfigMaps: []v1alpha1.ConfigMapRef{
								{Name: "failing-config"},
							},
						},
						Deployment: v1alpha1.PowerMonitorInternalKeplerDeploymentSpec{
							Image:     "test-image:latest",
							Namespace: "test-ns",
						},
					},
				},
			},
			setupClient: func() client.Client {
				yamlFailingConfig := `
kube:
  enabled: true
  nodeName: "test-node"
`
				cfm := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{Name: "failing-config", Namespace: "test-ns"},
					Data: map[string]string{
						powermonitor.KeplerConfigFile: yamlFailingConfig,
					},
				}
				testClient := &testMockClient{
					Client:    fake.NewClientBuilder().WithScheme(scheme).WithObjects(cfm).Build(),
					getErrors: make(map[string]error),
				}
				testClient.getErrors["/, Kind=/test-ns/failing-config"] = errors.NewInternalError(fmt.Errorf("get failed"))
				return testClient
			},
			expectedAction: Stop,
			expectedError:  true,
			errorContains:  "failed to get ConfigMap failing-config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := tt.setupClient()
			ds := &appsv1.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      tt.pmi.DaemonsetName(),
					Namespace: tt.pmi.Namespace(),
				},
				Spec: appsv1.DaemonSetSpec{
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: make(map[string]string),
						},
					},
				},
			}

			deployer := PowerMonitorDeployer{
				Pmi: tt.pmi,
				Ds:  ds,
			}

			result := deployer.Reconcile(context.Background(), client, scheme)

			assert.Equal(t, tt.expectedAction, result.Action)
			if tt.expectedError {
				assert.Error(t, result.Error)
				if tt.errorContains != "" {
					assert.Contains(t, result.Error.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, result.Error)

				// Verify ConfigMap was created for successful cases
				cfm := &corev1.ConfigMap{}
				key := types.NamespacedName{
					Name:      tt.pmi.Name,
					Namespace: tt.pmi.Namespace(),
				}
				err := client.Get(context.Background(), key, cfm)
				assert.NoError(t, err)
				assert.Contains(t, cfm.Data, powermonitor.KeplerConfigFile)
			}
		})
	}
}

func TestPowerMonitorDeployer_readAdditionalConfigs(t *testing.T) {
	scheme := testScheme()

	tests := []struct {
		name            string
		pmi             *v1alpha1.PowerMonitorInternal
		setupClient     func() client.Client
		expectedConfigs []string
		expectedError   bool
		errorContains   string
	}{
		{
			name: "no additional configs",
			pmi: &v1alpha1.PowerMonitorInternal{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pmi",
					Namespace: "test-ns",
				},
				Spec: v1alpha1.PowerMonitorInternalSpec{
					Kepler: v1alpha1.PowerMonitorInternalKeplerSpec{
						Config: v1alpha1.PowerMonitorInternalKeplerConfigSpec{
							AdditionalConfigMaps: []v1alpha1.ConfigMapRef{},
						},
						Deployment: v1alpha1.PowerMonitorInternalKeplerDeploymentSpec{
							Namespace: "test-ns",
						},
					},
				},
			},
			setupClient: func() client.Client {
				return fake.NewClientBuilder().WithScheme(scheme).Build()
			},
			expectedConfigs: nil,
			expectedError:   false,
		},
		{
			name: "single additional config",
			pmi: &v1alpha1.PowerMonitorInternal{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pmi",
					Namespace: "test-ns",
				},
				Spec: v1alpha1.PowerMonitorInternalSpec{
					Kepler: v1alpha1.PowerMonitorInternalKeplerSpec{
						Config: v1alpha1.PowerMonitorInternalKeplerConfigSpec{
							AdditionalConfigMaps: []v1alpha1.ConfigMapRef{
								{Name: "config-1"},
							},
						},
						Deployment: v1alpha1.PowerMonitorInternalKeplerDeploymentSpec{
							Namespace: "test-ns",
						},
					},
				},
			},
			setupClient: func() client.Client {
				yamlConfig := `
log:
  level: warn
  format: text
monitor:
  maxTerminated: 100
`
				cfm := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{Name: "config-1", Namespace: "test-ns"},
					Data: map[string]string{
						powermonitor.KeplerConfigFile: yamlConfig,
					},
				}
				return fake.NewClientBuilder().WithScheme(scheme).WithObjects(cfm).Build()
			},
			expectedConfigs: []string{`
log:
  level: warn
  format: text
monitor:
  maxTerminated: 100
`},
			expectedError: false,
		},
		{
			name: "multiple additional configs",
			pmi: &v1alpha1.PowerMonitorInternal{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pmi",
					Namespace: "test-ns",
				},
				Spec: v1alpha1.PowerMonitorInternalSpec{
					Kepler: v1alpha1.PowerMonitorInternalKeplerSpec{
						Config: v1alpha1.PowerMonitorInternalKeplerConfigSpec{
							AdditionalConfigMaps: []v1alpha1.ConfigMapRef{
								{Name: "config-1"},
								{Name: "config-2"},
							},
						},
						Deployment: v1alpha1.PowerMonitorInternalKeplerDeploymentSpec{
							Namespace: "test-ns",
						},
					},
				},
			},
			setupClient: func() client.Client {
				yamlConfig1 := `
debug:
  pprof:
    enabled: true
rapl:
  zones: ["package", "dram"]
`
				yamlConfig2 := `
web:
  listenAddresses: [":9090"]
host:
  sysfs: "/custom/sys"
  procfs: "/custom/proc"
`
				cfm1 := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{Name: "config-1", Namespace: "test-ns"},
					Data: map[string]string{
						powermonitor.KeplerConfigFile: yamlConfig1,
					},
				}
				cfm2 := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{Name: "config-2", Namespace: "test-ns"},
					Data: map[string]string{
						powermonitor.KeplerConfigFile: yamlConfig2,
					},
				}
				return fake.NewClientBuilder().WithScheme(scheme).WithObjects(cfm1, cfm2).Build()
			},
			expectedConfigs: []string{`
debug:
  pprof:
    enabled: true
rapl:
  zones: ["package", "dram"]
`, `
web:
  listenAddresses: [":9090"]
host:
  sysfs: "/custom/sys"
  procfs: "/custom/proc"
`},
			expectedError: false,
		},
		{
			name: "filters invalid configs",
			pmi: &v1alpha1.PowerMonitorInternal{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pmi",
					Namespace: "test-ns",
				},
				Spec: v1alpha1.PowerMonitorInternalSpec{
					Kepler: v1alpha1.PowerMonitorInternalKeplerSpec{
						Config: v1alpha1.PowerMonitorInternalKeplerConfigSpec{
							AdditionalConfigMaps: []v1alpha1.ConfigMapRef{
								{Name: "valid"},
								{Name: "invalid"},
								{Name: "empty"},
								{Name: "nil-data"},
							},
						},
						Deployment: v1alpha1.PowerMonitorInternalKeplerDeploymentSpec{
							Namespace: "test-ns",
						},
					},
				},
			},
			setupClient: func() client.Client {
				yamlValidConfig := `
log:
  level: error
exporter:
  stdout:
    enabled: false
  prometheus:
    enabled: true
    debugCollectors: ["go", "process"]
`
				validCfm := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{Name: "valid", Namespace: "test-ns"},
					Data: map[string]string{
						powermonitor.KeplerConfigFile: yamlValidConfig,
					},
				}
				invalidCfm := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{Name: "invalid", Namespace: "test-ns"},
					Data: map[string]string{
						"other-file.yaml": "invalid content",
					},
				}
				emptyCfm := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{Name: "empty", Namespace: "test-ns"},
					Data: map[string]string{
						powermonitor.KeplerConfigFile: "",
					},
				}
				nilDataCfm := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{Name: "nil-data", Namespace: "test-ns"},
					Data:       nil,
				}
				return fake.NewClientBuilder().WithScheme(scheme).WithObjects(validCfm, invalidCfm, emptyCfm, nilDataCfm).Build()
			},
			expectedConfigs: []string{`
log:
  level: error
exporter:
  stdout:
    enabled: false
  prometheus:
    enabled: true
    debugCollectors: ["go", "process"]
`},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := tt.setupClient()
			ds := &appsv1.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      tt.pmi.DaemonsetName(),
					Namespace: tt.pmi.Namespace(),
				},
			}

			deployer := PowerMonitorDeployer{
				Pmi: tt.pmi,
				Ds:  ds,
			}

			configs, err := deployer.readAdditionalConfigs(context.Background(), client)

			if tt.expectedError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				assert.Nil(t, configs)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedConfigs, configs)
			}
		})
	}
}

func TestSecretMounter_Reconcile(t *testing.T) {
	scheme := testScheme()

	tests := []struct {
		name           string
		pmi            *v1alpha1.PowerMonitorInternal
		setupClient    func() client.Client
		expectedAction Action
		expectedError  bool
		errorType      string
		errorContains  string
	}{
		{
			name: "no secrets referenced - should continue",
			pmi: &v1alpha1.PowerMonitorInternal{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pmi",
					Namespace: "test-ns",
				},
				Spec: v1alpha1.PowerMonitorInternalSpec{
					Kepler: v1alpha1.PowerMonitorInternalKeplerSpec{
						Deployment: v1alpha1.PowerMonitorInternalKeplerDeploymentSpec{
							PowerMonitorKeplerDeploymentSpec: v1alpha1.PowerMonitorKeplerDeploymentSpec{
								Secrets: []v1alpha1.SecretRef{}, // No secrets
							},
							Namespace: "test-ns",
						},
					},
				},
			},
			setupClient: func() client.Client {
				return fake.NewClientBuilder().WithScheme(scheme).Build()
			},
			expectedAction: Continue,
			expectedError:  false,
		},
		{
			name: "all secrets exist - should continue",
			pmi: &v1alpha1.PowerMonitorInternal{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pmi",
					Namespace: "test-ns",
				},
				Spec: v1alpha1.PowerMonitorInternalSpec{
					Kepler: v1alpha1.PowerMonitorInternalKeplerSpec{
						Deployment: v1alpha1.PowerMonitorInternalKeplerDeploymentSpec{
							PowerMonitorKeplerDeploymentSpec: v1alpha1.PowerMonitorKeplerDeploymentSpec{
								Secrets: []v1alpha1.SecretRef{
									{Name: "secret-1"},
									{Name: "secret-2"},
								},
							},
							Namespace: "test-ns",
						},
					},
				},
			},
			setupClient: func() client.Client {
				secret1 := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{Name: "secret-1", Namespace: "test-ns"},
					Data:       map[string][]byte{"key": []byte("value")},
				}
				secret2 := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{Name: "secret-2", Namespace: "test-ns"},
					Data:       map[string][]byte{"key": []byte("value")},
				}
				return fake.NewClientBuilder().WithScheme(scheme).WithObjects(secret1, secret2).Build()
			},
			expectedAction: Continue,
			expectedError:  false,
		},
		{
			name: "one secret missing - should continue with SecretNotFoundError",
			pmi: &v1alpha1.PowerMonitorInternal{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pmi",
					Namespace: "test-ns",
				},
				Spec: v1alpha1.PowerMonitorInternalSpec{
					Kepler: v1alpha1.PowerMonitorInternalKeplerSpec{
						Deployment: v1alpha1.PowerMonitorInternalKeplerDeploymentSpec{
							PowerMonitorKeplerDeploymentSpec: v1alpha1.PowerMonitorKeplerDeploymentSpec{
								Secrets: []v1alpha1.SecretRef{
									{Name: "secret-1"},
									{Name: "missing-secret"},
								},
							},
							Namespace: "test-ns",
						},
					},
				},
			},
			setupClient: func() client.Client {
				secret1 := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{Name: "secret-1", Namespace: "test-ns"},
					Data:       map[string][]byte{"key": []byte("value")},
				}
				return fake.NewClientBuilder().WithScheme(scheme).WithObjects(secret1).Build()
			},
			expectedAction: Continue,
			expectedError:  true,
			errorType:      "SecretNotFoundError",
			errorContains:  "missing-secret not found in test-ns namespace",
		},
		{
			name: "multiple secrets missing - should continue with SecretNotFoundError",
			pmi: &v1alpha1.PowerMonitorInternal{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pmi",
					Namespace: "test-ns",
				},
				Spec: v1alpha1.PowerMonitorInternalSpec{
					Kepler: v1alpha1.PowerMonitorInternalKeplerSpec{
						Deployment: v1alpha1.PowerMonitorInternalKeplerDeploymentSpec{
							PowerMonitorKeplerDeploymentSpec: v1alpha1.PowerMonitorKeplerDeploymentSpec{
								Secrets: []v1alpha1.SecretRef{
									{Name: "missing-secret-1"},
									{Name: "missing-secret-2"},
								},
							},
							Namespace: "test-ns",
						},
					},
				},
			},
			setupClient: func() client.Client {
				return fake.NewClientBuilder().WithScheme(scheme).Build()
			},
			expectedAction: Continue,
			expectedError:  true,
			errorType:      "SecretNotFoundError",
			errorContains:  "test-ns namespace",
		},
		{
			name: "client error (not NotFound) - should stop",
			pmi: &v1alpha1.PowerMonitorInternal{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pmi",
					Namespace: "test-ns",
				},
				Spec: v1alpha1.PowerMonitorInternalSpec{
					Kepler: v1alpha1.PowerMonitorInternalKeplerSpec{
						Deployment: v1alpha1.PowerMonitorInternalKeplerDeploymentSpec{
							PowerMonitorKeplerDeploymentSpec: v1alpha1.PowerMonitorKeplerDeploymentSpec{
								Secrets: []v1alpha1.SecretRef{
									{Name: "failing-secret"},
								},
							},
							Namespace: "test-ns",
						},
					},
				},
			},
			setupClient: func() client.Client {
				mockClient := &testMockClient{
					Client:    fake.NewClientBuilder().WithScheme(scheme).Build(),
					getErrors: map[string]error{},
				}
				mockClient.getErrors["/, Kind=/test-ns/failing-secret"] = errors.NewInternalError(fmt.Errorf("internal server error"))
				return mockClient
			},
			expectedAction: Stop,
			expectedError:  true,
			errorContains:  "failed to get secret failing-secret",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := tt.setupClient()

			// Create a mock DaemonSet for SecretMounter
			ds := &appsv1.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-ds",
					Namespace: "test-ns",
				},
				Spec: appsv1.DaemonSetSpec{
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: make(map[string]string),
						},
					},
				},
			}

			mounter := SecretMounter{
				Pmi:    tt.pmi,
				Ds:     ds,
				Logger: logr.Discard(), // Use discard logger for tests
			}

			result := mounter.Reconcile(context.Background(), client, scheme)

			assert.Equal(t, tt.expectedAction, result.Action)
			if tt.expectedError {
				assert.Error(t, result.Error)
				if tt.errorContains != "" {
					assert.Contains(t, result.Error.Error(), tt.errorContains)
				}
				if tt.errorType == "SecretNotFoundError" {
					_, ok := result.Error.(*SecretNotFoundError)
					assert.True(t, ok, "Error should be of type SecretNotFoundError")
				}
			} else {
				assert.NoError(t, result.Error)
			}
		})
	}
}
