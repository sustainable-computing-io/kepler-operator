/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/sustainable.computing.io/kepler-operator/pkg/api/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	secv1 "github.com/openshift/api/security/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
)

// default ForeverTestTimeout is 30, some test fail because they take more than 30s
// change to custom in order to let the test finish withouth errors
const TestTimeout = 40 * time.Second

type Framework struct {
	T      *testing.T
	client client.Client
}
type frameworkFn func(*Framework)

func NewFramework(t *testing.T, fns ...frameworkFn) *Framework {
	t.Helper()
	f := Framework{T: t}
	for _, fn := range fns {
		fn(&f)
	}
	if f.client == nil {
		f.client = f.NewClient(f.Scheme())
	}

	return &f
}

func (f Framework) WithT(t *testing.T) Framework {
	dup := f
	dup.T = t
	return dup
}

func WithClient(c client.Client) frameworkFn {
	return func(f *Framework) {
		f.client = c
	}
}

func (f Framework) Client() client.Client {
	return f.client
}

func (f Framework) Scheme() *runtime.Scheme {
	f.T.Helper()
	scheme := runtime.NewScheme()
	assert.NoError(f.T, corev1.AddToScheme(scheme))
	assert.NoError(f.T, appsv1.AddToScheme(scheme))
	assert.NoError(f.T, secv1.AddToScheme(scheme))
	assert.NoError(f.T, rbacv1.AddToScheme(scheme))
	assert.NoError(f.T, v1alpha1.AddToScheme(scheme))
	return scheme
}

var once sync.Once

func (f Framework) NewClient(scheme *runtime.Scheme) client.Client {
	f.T.Helper()
	once.Do(func() {
		ctrl.SetLogger(zap.New())
	})
	cfg := config.GetConfigOrDie()
	c, err := client.New(cfg, client.Options{Scheme: scheme})
	assert.NoError(f.T, err)
	return c
}

type keplerFn func(*v1alpha1.Kepler)

func WithExporterPort(port int32) keplerFn {
	return func(k *v1alpha1.Kepler) {
		k.Spec.Exporter.Port = port
	}
}

func (f Framework) GetKepler(name string) *v1alpha1.Kepler {
	kepler := v1alpha1.Kepler{}
	f.AssertResourceExits(name, "", &kepler)
	return &kepler
}

func (f Framework) CreateKepler(name string, fns ...keplerFn) *v1alpha1.Kepler {
	kepler := v1alpha1.Kepler{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.GroupVersion.String(),
			Kind:       "Kepler",
		},

		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1alpha1.KeplerSpec{},
	}

	for _, fn := range fns {
		fn(&kepler)
	}

	f.T.Cleanup(func() {
		f.client.Delete(context.TODO(), &kepler)
	})

	err := f.client.Patch(context.TODO(), &kepler, client.Apply,
		client.ForceOwnership, client.FieldOwner("e2e-test"),
	)
	assert.NoError(f.T, err, "failed to create kepler")

	return &kepler
}

func (f Framework) DeleteKepler(name string) {
	f.T.Helper()
	k := v1alpha1.Kepler{}

	err := f.client.Get(context.TODO(), client.ObjectKey{Name: name}, &k)
	if errors.IsNotFound(err) {
		return
	}
	assert.NoError(f.T, err, "failed to get kepler :%s", name)

	err = f.client.Delete(context.Background(), &k)
	assert.NoError(f.T, err, "failed to delete kepler :%s", name)

	f.AssertNoResourceExits(name, "", &k)
}
