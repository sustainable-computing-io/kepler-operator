package e2e

import (
	"testing"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	"github.com/stretchr/testify/assert"
	"github.com/sustainable.computing.io/kepler-operator/pkg/utils/test"

	"k8s.io/apimachinery/pkg/runtime"
)

func k8sClient(scheme *runtime.Scheme) (client.Client, error) {
	cfg := config.GetConfigOrDie()
	c, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		return nil, err
	}

	return c, nil
}

func TestKepler_Create(t *testing.T) {
	f := test.NewFramework(t)
	k8sClient, err := k8sClient(f.Scheme())
	assert.NoError(t, err)
	assert.NotNil(t, k8sClient)
}
