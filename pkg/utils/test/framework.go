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
	"bytes"
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"golang.org/x/exp/slices"

	"github.com/stretchr/testify/assert"
	"github.com/sustainable.computing.io/kepler-operator/api/v1alpha1"
	"github.com/sustainable.computing.io/kepler-operator/pkg/utils/k8s"
	"github.com/sustainable.computing.io/kepler-operator/pkg/utils/test/oc"
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

// default ForeverTestTimeout is 30 seconds, some tests fail because they take
// more than 30s, some times, close to 2 minutes, so we set it to 2 minutes.
//
// NOTE: if a specific test case requires more than 2 minutes, use Timeout()
// function to set the timeout for just that assertion.
const TestTimeout = 2 * time.Minute

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

type (
	internalFn func(*v1alpha1.KeplerInternal)
	keplerFn   func(*v1alpha1.Kepler)
)

func WithExporterPort(port int32) keplerFn {
	return func(k *v1alpha1.Kepler) {
		k.Spec.Exporter.Deployment.Port = port
	}
}

func (f Framework) GetKepler(name string) *v1alpha1.Kepler {
	kepler := v1alpha1.Kepler{}
	f.AssertResourceExists(name, "", &kepler)
	return &kepler
}

func (f Framework) NewKepler(name string, fns ...keplerFn) v1alpha1.Kepler {
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
	return kepler
}

func (f Framework) Patch(obj client.Object) error {
	f.T.Logf("%s: creating/updating object %s", time.Now().UTC().Format(time.RFC3339), obj.GetName())
	return f.client.Patch(context.TODO(), obj, client.Apply,
		client.ForceOwnership, client.FieldOwner("e2e-test"),
	)
}

func (f Framework) CreateKepler(name string, fns ...keplerFn) *v1alpha1.Kepler {
	kepler := f.NewKepler(name, fns...)
	f.T.Logf("%s: creating/updating kepler %s", time.Now().UTC().Format(time.RFC3339), name)
	err := f.client.Patch(context.TODO(), &kepler, client.Apply,
		client.ForceOwnership, client.FieldOwner("e2e-test"),
	)
	assert.NoError(f.T, err, "failed to create kepler")

	f.T.Cleanup(func() {
		f.DeleteKepler(name)
	})

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

	f.T.Logf("%s: deleting kepler %s", time.Now().UTC().Format(time.RFC3339), name)

	err = f.client.Delete(context.Background(), &k)
	if err != nil && !errors.IsNotFound(err) {
		f.T.Errorf("failed to delete kepler:%s :%v", name, err)
	}

	f.WaitUntil(fmt.Sprintf("kepler %s is deleted", name), func(ctx context.Context) (bool, error) {
		k := v1alpha1.Kepler{}
		err := f.client.Get(ctx, client.ObjectKey{Name: name}, &k)
		return errors.IsNotFound(err), nil
	})
}

func (f Framework) GetKeplerInternal(name string) *v1alpha1.KeplerInternal {
	kepler := v1alpha1.KeplerInternal{}
	f.AssertResourceExists(name, "", &kepler)
	return &kepler
}

func (f Framework) CreateInternal(name string, fns ...internalFn) *v1alpha1.KeplerInternal {
	ki := v1alpha1.KeplerInternal{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.GroupVersion.String(),
			Kind:       "KeplerInternal",
		},

		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1alpha1.KeplerInternalSpec{},
	}

	for _, fn := range fns {
		fn(&ki)
	}

	f.T.Logf("%s: creating/updating kepler-internal %s", time.Now().UTC().Format(time.RFC3339), name)
	err := f.client.Patch(context.TODO(), &ki, client.Apply,
		client.ForceOwnership, client.FieldOwner("e2e-test"),
	)
	assert.NoError(f.T, err, "failed to create kepler-internal")

	f.T.Cleanup(func() {
		f.DeleteInternal(name, Timeout(5*time.Minute))
	})

	return &ki
}

func (f Framework) DeleteInternal(name string, fns ...AssertOptionFn) {
	f.T.Helper()

	k := v1alpha1.KeplerInternal{}
	err := f.client.Get(context.TODO(), client.ObjectKey{Name: name}, &k)
	if errors.IsNotFound(err) {
		return
	}
	assert.NoError(f.T, err, "failed to get kepler-internal :%s", name)

	f.T.Logf("%s: deleting kepler-internal %s", time.Now().UTC().Format(time.RFC3339), name)

	err = f.client.Delete(context.Background(), &k)
	if err != nil && !errors.IsNotFound(err) {
		f.T.Errorf("failed to delete kepler-internal:%s :%v", name, err)
	}

	f.WaitUntil(fmt.Sprintf("kepler-internal %s is deleted", name), func(ctx context.Context) (bool, error) {
		k := v1alpha1.KeplerInternal{}
		err := f.client.Get(ctx, client.ObjectKey{Name: name}, &k)
		return errors.IsNotFound(err), nil
	}, fns...)
}

func (f Framework) WaitUntilInternalCondition(name string, t v1alpha1.ConditionType, s v1alpha1.ConditionStatus, fns ...AssertOptionFn) *v1alpha1.KeplerInternal {
	f.T.Helper()
	k := v1alpha1.KeplerInternal{}
	f.WaitUntil(fmt.Sprintf("kepler-internal %s is %s", name, t),
		func(ctx context.Context) (bool, error) {
			err := f.client.Get(ctx, client.ObjectKey{Name: name}, &k)
			if errors.IsNotFound(err) {
				return true, fmt.Errorf("kepler-internal %s is not found", name)
			}

			condition, _ := k8s.FindCondition(k.Status.Exporter.Conditions, t)
			return condition.Status == s, nil
		}, fns...)
	return &k
}

func (f Framework) WaitUntilKeplerCondition(name string, t v1alpha1.ConditionType, s v1alpha1.ConditionStatus) *v1alpha1.Kepler {
	f.T.Helper()
	k := v1alpha1.Kepler{}
	f.WaitUntil(fmt.Sprintf("kepler %s is %s", name, t),
		func(ctx context.Context) (bool, error) {
			err := f.client.Get(ctx, client.ObjectKey{Name: name}, &k)
			if errors.IsNotFound(err) {
				return true, fmt.Errorf("kepler %s is not found", name)
			}

			condition, _ := k8s.FindCondition(k.Status.Exporter.Conditions, t)
			return condition.Status == s, nil
		})
	return &k
}

func (f Framework) AddResourceLabels(kind, name string, l map[string]string) error {
	f.T.Helper()
	b := new(bytes.Buffer)
	for label, value := range l {
		fmt.Fprintf(b, "%s=%s ", label, value)
	}
	f.T.Cleanup(func() {
		err := f.RemoveResourceLabels(kind, name, []string{"e2e-test"})
		assert.NoError(f.T, err, "could not remove label from node")
	})
	return f.AddResourceLabelsStr(kind, name, b.String())
}

func (f Framework) AddResourceLabelsStr(kind, name, l string) error {
	f.T.Helper()
	_, err := oc.Literal().From("oc label %s %s %s", kind, name, l).Run()
	return err
}

func (f Framework) RemoveResourceLabels(kind, name string, l []string) error {
	f.T.Helper()
	b := new(bytes.Buffer)
	for _, label := range l {
		fmt.Fprintf(b, "%s- ", label)
	}
	_, err := oc.Literal().From("oc label %s %s %s", kind, name, b.String()).Run()
	return err
}

func (f Framework) WithNodeSelector(label map[string]string) func(k *v1alpha1.Kepler) {
	return func(k *v1alpha1.Kepler) {
		k.Spec.Exporter.Deployment.NodeSelector = label
	}
}

func (f Framework) TaintNode(node, taintStr string) error {
	f.T.Helper()
	_, err := oc.Literal().From("oc adm taint node %s %s", node, taintStr).Run()
	f.T.Cleanup(func() {
		// remove taint
		_, err := oc.Literal().From("oc adm taint node %s %s", node, fmt.Sprintf("%s-", taintStr)).Run()
		assert.NoError(f.T, err, "could not remove taint from node")
	})
	return err
}

func (f Framework) WithTolerations(taints []corev1.Taint) func(k *v1alpha1.Kepler) {
	return func(k *v1alpha1.Kepler) {
		k.Spec.Exporter.Deployment.Tolerations = tolerateTaints(taints)
	}
}

func (f Framework) GetSchedulableNodes() []corev1.Node {
	f.T.Helper()
	var nodes corev1.NodeList
	err := f.client.List(context.TODO(), &nodes)
	assert.NoError(f.T, err, "failed to get nodes")

	var ret []corev1.Node
	for _, n := range nodes.Items {
		if isSchedulableNode(n) {
			ret = append(ret, n)
		}
	}
	return ret
}

func tolerateTaints(taints []corev1.Taint) []corev1.Toleration {
	var to []corev1.Toleration
	for _, ta := range taints {
		to = append(to, corev1.Toleration{
			Key:      ta.Key,
			Value:    ta.Value,
			Operator: corev1.TolerationOpEqual,
			Effect:   ta.Effect,
		})
	}
	return to
}

func isSchedulableNode(n corev1.Node) bool {
	return slices.IndexFunc(n.Spec.Taints, func(t corev1.Taint) bool {
		return t.Effect == corev1.TaintEffectNoSchedule ||
			t.Effect == corev1.TaintEffectNoExecute
	}) == -1
}
