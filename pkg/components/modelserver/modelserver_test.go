package modelserver

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/sustainable.computing.io/kepler-operator/pkg/api/v1alpha1"
	"github.com/sustainable.computing.io/kepler-operator/pkg/components"
)

func TestConfigMap(t *testing.T) {

	tt := []struct {
		spec     *v1alpha1.ModelServerSpec
		data     map[string]string
		scenario string
	}{
		{
			spec: &v1alpha1.ModelServerSpec{},
			data: map[string]string{
				"MODEL_PATH": "/mnt/models",
			},
			scenario: "default case",
		},
		{
			spec: &v1alpha1.ModelServerSpec{
				URL:         "fake-url",
				Path:        "fake-model-path",
				RequestPath: "fake-request-path",
				ListPath:    "fake-model-list-path",
				PipelineURL: "fake-pipeline",
				ErrorKey:    "fake-error-key",
			},
			data: map[string]string{
				"MODEL_PATH":                   "fake-model-path",
				"MODEL_SERVER_REQ_PATH":        "fake-request-path",
				"MODEL_SERVER_MODEL_LIST_PATH": "fake-model-list-path",
				"INITIAL_PIPELINE_URL":         "fake-pipeline",
				"ERROR_KEY":                    "fake-error-key",
			},
			scenario: "user defined server-api config",
		},
	}

	for _, tc := range tt {
		tc := tc
		t.Run(tc.scenario, func(t *testing.T) {
			t.Parallel()
			k := v1alpha1.Kepler{
				Spec: v1alpha1.KeplerSpec{
					ModelServer: tc.spec,
				},
			}
			actual := NewConfigMap(components.Full, k.Spec.ModelServer)
			assert.Equal(t, len(tc.data), len(actual.Data))
			for k, v := range tc.data {
				assert.Equal(t, v, actual.Data[k])
			}
		})
	}

}

func TestService(t *testing.T) {

	tt := []struct {
		spec        v1alpha1.ModelServerSpec
		servicePort int32
		scenario    string
	}{
		{
			spec: v1alpha1.ModelServerSpec{
				Port: 8080,
			},
			servicePort: 8080,
			scenario:    "user defined port",
		},
	}

	for _, tc := range tt {
		tc := tc
		t.Run(tc.scenario, func(t *testing.T) {
			t.Parallel()
			k := v1alpha1.Kepler{
				Spec: v1alpha1.KeplerSpec{
					ModelServer: &tc.spec,
				},
			}
			actual := NewService(k.Spec.ModelServer)
			assert.Equal(t, actual.Spec.Ports[0].Port, tc.servicePort)
		})
	}

}

func TestServerAPIContainer(t *testing.T) {

	tt := []struct {
		spec        v1alpha1.ModelServerSpec
		servicePort int32
		scenario    string
	}{
		{
			spec: v1alpha1.ModelServerSpec{
				Port: 8080,
			},
			servicePort: 8080,
			scenario:    "user defined port",
		},
	}

	for _, tc := range tt {
		tc := tc
		t.Run(tc.scenario, func(t *testing.T) {
			t.Parallel()
			k := v1alpha1.Kepler{
				Spec: v1alpha1.KeplerSpec{
					ModelServer: &tc.spec,
				},
			}
			actual := NewDeployment(k.Spec.ModelServer)
			containers := actual.Spec.Template.Spec.Containers
			assert.Equal(t, len(containers), 1)
			assert.Equal(t, containers[0].Ports[0].ContainerPort, tc.servicePort)
		})
	}

}

func TestTrainerContainer(t *testing.T) {

	tt := []struct {
		spec     v1alpha1.ModelServerSpec
		data     map[string]string
		args     []string
		scenario string
	}{
		{
			spec: v1alpha1.ModelServerSpec{
				Trainer: &v1alpha1.ModelServerTrainerSpec{
					Prom: &v1alpha1.PrometheusSpec{SSLDisable: true},
				},
			},
			data: map[string]string{
				"MODEL_PATH":          "/mnt/models",
				"PROM_SERVER":         "http://prometheus-k8s.openshift-kepler-operator.svc.cluster.local:9090/",
				"PROM_SSL_DISABLE":    "true",
				"PROM_QUERY_INTERVAL": "20",
				"PROM_QUERY_STEP":     "3",
			},
			args:     []string{"src/server/model_server.py", "src/train/online_trainer.py"},
			scenario: "default case",
		},
		// TODO: set PROM_HEADERS case
	}

	for _, tc := range tt {
		tc := tc
		t.Run(tc.scenario, func(t *testing.T) {
			t.Parallel()
			k := v1alpha1.Kepler{
				Spec: v1alpha1.KeplerSpec{
					ModelServer: &tc.spec,
				},
			}
			actualConfigmap := NewConfigMap(components.Full, k.Spec.ModelServer)
			assert.Equal(t, len(tc.data), len(actualConfigmap.Data))
			for k, v := range tc.data {
				assert.Equal(t, v, actualConfigmap.Data[k])
			}

			actualDeployment := NewDeployment(k.Spec.ModelServer)
			containers := actualDeployment.Spec.Template.Spec.Containers
			assert.Equal(t, len(containers), 2)
			assert.Equal(t, containers[0].Args[1], tc.args[0])
			assert.Equal(t, containers[1].Args[1], tc.args[1])
		})
	}

}
