// SPDX-FileCopyrightText: 2025 The Kepler Authors
// SPDX-License-Identifier: Apache-2.0

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

	certv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	certmetav1 "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	secv1 "github.com/openshift/api/security/v1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
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
	assert.NoError(f.T, certv1.AddToScheme(scheme))
	assert.NoError(f.T, batchv1.AddToScheme(scheme))
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
	internalFn             func(*v1alpha1.KeplerInternal)
	powermonitorinternalFn func(*v1alpha1.PowerMonitorInternal)
	keplerFn               func(*v1alpha1.Kepler)
	powermonitorFn         func(*v1alpha1.PowerMonitor)
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

func (f Framework) GetPowerMonitor(name string) *v1alpha1.PowerMonitor {
	pm := v1alpha1.PowerMonitor{}
	f.AssertResourceExists(name, "", &pm)
	return &pm
}

func (f Framework) NewPowerMonitor(name string, fns ...powermonitorFn) v1alpha1.PowerMonitor {
	pm := v1alpha1.PowerMonitor{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.GroupVersion.String(),
			Kind:       "PowerMonitor",
		},

		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1alpha1.PowerMonitorSpec{},
	}

	for _, fn := range fns {
		fn(&pm)
	}
	return pm
}

func (f Framework) CreatePowerMonitor(name string, fns ...powermonitorFn) *v1alpha1.PowerMonitor {
	pm := f.NewPowerMonitor(name, fns...)
	f.T.Logf("%s: creating/updating powermonitor %s", time.Now().UTC().Format(time.RFC3339), name)
	err := f.client.Patch(context.TODO(), &pm, client.Apply,
		client.ForceOwnership, client.FieldOwner("e2e-test"),
	)
	assert.NoError(f.T, err, "failed to create powermonitor")

	f.T.Cleanup(func() {
		f.DeletePowerMonitor(name)
	})

	return &pm
}

func (f Framework) DeletePowerMonitor(name string) {
	f.T.Helper()

	pm := v1alpha1.PowerMonitor{}
	err := f.client.Get(context.TODO(), client.ObjectKey{Name: name}, &pm)
	if errors.IsNotFound(err) {
		return
	}
	assert.NoError(f.T, err, "failed to get powermonitor :%s", name)

	f.T.Logf("%s: deleting powermonitor %s", time.Now().UTC().Format(time.RFC3339), name)

	err = f.client.Delete(context.Background(), &pm)
	if err != nil && !errors.IsNotFound(err) {
		f.T.Errorf("failed to delete powermonitor:%s :%v", name, err)
	}

	f.WaitUntil(fmt.Sprintf("powermonitor %s is deleted", name), func(ctx context.Context) (bool, error) {
		pm := v1alpha1.PowerMonitor{}
		err := f.client.Get(ctx, client.ObjectKey{Name: name}, &pm)
		return errors.IsNotFound(err), nil
	})
}

func (f Framework) GetPowerMonitorInternal(name string) *v1alpha1.PowerMonitorInternal {
	pmi := v1alpha1.PowerMonitorInternal{}
	f.AssertResourceExists(name, "", &pmi)
	return &pmi
}

func (f Framework) CreatePowerMonitorInternal(name string, fns ...powermonitorinternalFn) *v1alpha1.PowerMonitorInternal {
	pmi := v1alpha1.PowerMonitorInternal{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.GroupVersion.String(),
			Kind:       "PowerMonitorInternal",
		},

		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1alpha1.PowerMonitorInternalSpec{},
	}

	for _, fn := range fns {
		fn(&pmi)
	}

	f.T.Logf("%s: creating/updating power-monitor-internal %s", time.Now().UTC().Format(time.RFC3339), name)
	err := f.client.Patch(context.TODO(), &pmi, client.Apply,
		client.ForceOwnership, client.FieldOwner("e2e-test"),
	)
	assert.NoError(f.T, err, "failed to create power-monitor-internal")

	f.T.Cleanup(func() {
		f.DeletePowerMonitorInternal(name, Timeout(5*time.Minute))
	})

	return &pmi
}

func (f Framework) DeletePowerMonitorInternal(name string, fns ...AssertOptionFn) {
	f.T.Helper()

	pmi := v1alpha1.PowerMonitorInternal{}
	err := f.client.Get(context.TODO(), client.ObjectKey{Name: name}, &pmi)
	if errors.IsNotFound(err) {
		return
	}
	assert.NoError(f.T, err, "failed to get power-monitor-internal :%s", name)

	f.T.Logf("%s: deleting power-monitor-internal %s", time.Now().UTC().Format(time.RFC3339), name)

	err = f.client.Delete(context.Background(), &pmi)
	if err != nil && !errors.IsNotFound(err) {
		f.T.Errorf("failed to delete power-monitor-internal:%s :%v", name, err)
	}

	f.WaitUntil(fmt.Sprintf("power-monitor-internal %s is deleted", name), func(ctx context.Context) (bool, error) {
		pmi := v1alpha1.PowerMonitorInternal{}
		err := f.client.Get(ctx, client.ObjectKey{Name: name}, &pmi)
		return errors.IsNotFound(err), nil
	}, fns...)
}

func (f Framework) WaitUntilPowerMonitorInternalCondition(name string, t v1alpha1.ConditionType, s v1alpha1.ConditionStatus, fns ...AssertOptionFn) *v1alpha1.PowerMonitorInternal {
	f.T.Helper()
	pmi := v1alpha1.PowerMonitorInternal{}
	f.WaitUntil(fmt.Sprintf("power-monitor-internal %s is %s", name, t),
		func(ctx context.Context) (bool, error) {
			err := f.client.Get(ctx, client.ObjectKey{Name: name}, &pmi)
			if errors.IsNotFound(err) {
				return true, fmt.Errorf("power-monitor-internal %s is not found", name)
			}

			condition, _ := k8s.FindCondition(pmi.Status.Conditions, t)
			return condition.Status == s, nil
		}, fns...)
	return &pmi
}

func (f Framework) WaitUntilPowerMonitorCondition(name string, t v1alpha1.ConditionType, s v1alpha1.ConditionStatus) *v1alpha1.PowerMonitor {
	f.T.Helper()
	pm := v1alpha1.PowerMonitor{}
	f.WaitUntil(fmt.Sprintf("powermonitor %s is %s", name, t),
		func(ctx context.Context) (bool, error) {
			err := f.client.Get(ctx, client.ObjectKey{Name: name}, &pm)
			if errors.IsNotFound(err) {
				return true, fmt.Errorf("powermonitor %s is not found", name)
			}

			condition, _ := k8s.FindCondition(pm.Status.Conditions, t)
			return condition.Status == s, nil
		})
	return &pm
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

func (f Framework) WithPowerMonitorNodeSelector(label map[string]string) func(k *v1alpha1.PowerMonitor) {
	return func(pm *v1alpha1.PowerMonitor) {
		pm.Spec.Kepler.Deployment.NodeSelector = label
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

func (f Framework) WithPowerMonitorTolerations(taints []corev1.Taint) func(k *v1alpha1.PowerMonitor) {
	return func(pm *v1alpha1.PowerMonitor) {
		pm.Spec.Kepler.Deployment.Tolerations = tolerateTaints(taints)
	}
}

func (f Framework) WithPowerMonitorSecuritySet(mode v1alpha1.SecurityMode, allowedSANames []string) func(k *v1alpha1.PowerMonitor) {
	return func(pm *v1alpha1.PowerMonitor) {
		pm.Spec.Kepler.Deployment.Security.Mode = mode
		pm.Spec.Kepler.Deployment.Security.AllowedSANames = allowedSANames
	}
}

func (f Framework) WithAdditionalConfigMaps(configMapNames []string) func(k *v1alpha1.PowerMonitor) {
	return func(pm *v1alpha1.PowerMonitor) {
		var configMapRefs []v1alpha1.ConfigMapRef
		for _, name := range configMapNames {
			configMapRefs = append(configMapRefs, v1alpha1.ConfigMapRef{Name: name})
		}
		pm.Spec.Kepler.Config.AdditionalConfigMaps = configMapRefs
	}
}

func (f Framework) WithMaxTerminated(maxTerminated int32) func(k *v1alpha1.PowerMonitor) {
	return func(pm *v1alpha1.PowerMonitor) {
		pm.Spec.Kepler.Config.MaxTerminated = &maxTerminated
	}
}

func (f Framework) WithStaleness(staleness string) func(k *v1alpha1.PowerMonitor) {
	return func(pm *v1alpha1.PowerMonitor) {
		duration, _ := time.ParseDuration(staleness)
		pm.Spec.Kepler.Config.Staleness = &metav1.Duration{Duration: duration}
	}
}

func (f Framework) WithSampleRate(sampleRate string) func(k *v1alpha1.PowerMonitor) {
	return func(pm *v1alpha1.PowerMonitor) {
		duration, _ := time.ParseDuration(sampleRate)
		pm.Spec.Kepler.Config.SampleRate = &metav1.Duration{Duration: duration}
	}
}

func (f Framework) NewAdditionalConfigMap(configMapName, namespace, config string) *corev1.ConfigMap {
	cm := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: namespace,
		},
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "ConfigMap",
		},
		Data: map[string]string{
			"config.yaml": config,
		},
	}
	return &cm
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

func (f Framework) DeployOpenshiftCerts(serviceName, serviceNamespace, clusterIssuerName, caCertName, caCertSecretName, pmIssuerName, tlsCertName, tlsCertSecretName string) {
	f.T.Helper()

	f.InstallCertManager()

	f.CreateSelfSignedClusterIssuer(clusterIssuerName)

	caCert := f.CreateCACertificate(caCertName, caCertSecretName, serviceNamespace, clusterIssuerName)
	f.WaitUntil("ca certificate and secret are deployed", func(ctx context.Context) (bool, error) {
		err := f.client.Get(ctx, client.ObjectKeyFromObject(caCert), caCert)
		if err != nil {
			return false, err
		}
		for _, cond := range caCert.Status.Conditions {
			if cond.Type == certv1.CertificateConditionReady && cond.Status == certmetav1.ConditionTrue {
				return true, nil
			}
		}
		return false, err
	}, Timeout(5*time.Minute))

	f.CreateCAIssuer(pmIssuerName, caCertSecretName, serviceNamespace)

	tlsCert := f.CreateTLSCertificate(tlsCertName, tlsCertSecretName, serviceNamespace, pmIssuerName, []string{
		fmt.Sprintf("%s.%s.svc", serviceName, serviceNamespace)})
	f.WaitUntil("tls certificate and secret are deployed", func(ctx context.Context) (bool, error) {
		err := f.client.Get(ctx, client.ObjectKeyFromObject(tlsCert), tlsCert)
		if err != nil {
			return false, err
		}
		for _, cond := range tlsCert.Status.Conditions {
			if cond.Type == certv1.CertificateConditionReady && cond.Status == certmetav1.ConditionTrue {
				return true, nil
			}
		}
		return false, err
	}, Timeout(5*time.Minute))

}

func (f Framework) InstallCertManager() {
	f.T.Helper()

	_, err := oc.Literal().From("kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.2/cert-manager.yaml").Run()
	assert.NoError(f.T, err, "failed to install cert-manager")

	f.WaitUntil("cert-manager pods are running", func(ctx context.Context) (bool, error) {
		pods := corev1.PodList{}
		err := f.client.List(ctx, &pods, client.InNamespace("cert-manager"))
		if err != nil {
			return false, err
		}

		for _, pod := range pods.Items {
			if pod.Status.Phase != corev1.PodRunning {
				return false, nil
			}
		}
		return true, nil
	}, Timeout(5*time.Minute))

	f.T.Cleanup(func() {
		_, err := oc.Literal().From("kubectl delete -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.2/cert-manager.yaml").Run()
		assert.NoError(f.T, err, "failed to uninstall cert-manager")
	})
}

func (f Framework) CreateSelfSignedClusterIssuer(name string) *certv1.ClusterIssuer {
	issuer := certv1.ClusterIssuer{
		TypeMeta: metav1.TypeMeta{
			APIVersion: certv1.SchemeGroupVersion.String(),
			Kind:       "ClusterIssuer",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: certv1.IssuerSpec{
			IssuerConfig: certv1.IssuerConfig{
				SelfSigned: &certv1.SelfSignedIssuer{},
			},
		},
	}

	err := f.client.Patch(context.TODO(), &issuer, client.Apply, client.ForceOwnership, client.FieldOwner("e2e-test"))
	assert.NoError(f.T, err, "failed to create ClusterIssuer")

	f.T.Cleanup(func() {
		f.DeleteSelfSignedClusterIssuer(name, Timeout(5*time.Minute))
	})
	return &issuer
}

func (f Framework) DeleteSelfSignedClusterIssuer(name string, fns ...AssertOptionFn) {
	f.T.Helper()

	issuer := certv1.ClusterIssuer{}
	err := f.client.Get(context.TODO(), client.ObjectKey{Name: name}, &issuer)
	if errors.IsNotFound(err) {
		return
	}
	assert.NoError(f.T, err, "failed to get self signed cluster issuer :%s", name)

	f.T.Logf("%s: deleting self signed cluster issuer %s", time.Now().UTC().Format(time.RFC3339), name)

	err = f.client.Delete(context.Background(), &issuer)
	if err != nil && !errors.IsNotFound(err) {
		f.T.Errorf("failed to delete self signed cluster issuer:%s :%v", name, err)
	}

	f.WaitUntil(fmt.Sprintf("self signed cluster %s is deleted", name), func(ctx context.Context) (bool, error) {
		issuer := certv1.ClusterIssuer{}
		err := f.client.Get(ctx, client.ObjectKey{Name: name}, &issuer)
		return errors.IsNotFound(err), nil
	}, fns...)
}

func (f Framework) CreateCACertificate(name, secretName, ns, issuerName string) *certv1.Certificate {
	cert := certv1.Certificate{
		TypeMeta: metav1.TypeMeta{
			APIVersion: certv1.SchemeGroupVersion.String(),
			Kind:       "Certificate",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Spec: certv1.CertificateSpec{
			IsCA:       true,
			CommonName: name,
			SecretName: secretName,
			Duration:   &metav1.Duration{Duration: 8760 * time.Hour},
			PrivateKey: &certv1.CertificatePrivateKey{
				Algorithm: certv1.RSAKeyAlgorithm,
				Size:      2048,
			},
			IssuerRef: certmetav1.ObjectReference{
				Name: issuerName,
				Kind: "ClusterIssuer",
			},
		},
	}

	err := f.client.Patch(context.TODO(), &cert, client.Apply, client.ForceOwnership, client.FieldOwner("e2e-test"))
	assert.NoError(f.T, err, "failed to create CA Certificate")

	f.T.Cleanup(func() {
		f.DeleteCACertificate(name, name, ns, Timeout(5*time.Minute))
	})

	return &cert
}

func (f Framework) DeleteCACertificate(name, secretName, ns string, fns ...AssertOptionFn) {
	f.T.Helper()
	caCert := certv1.Certificate{}
	err := f.client.Get(context.TODO(), client.ObjectKey{Name: name, Namespace: ns}, &caCert)
	if errors.IsNotFound(err) {
		return
	}
	assert.NoError(f.T, err, "failed to get ca certificate :%s", name)

	caSecret := corev1.Secret{}
	err = f.client.Get(context.TODO(), client.ObjectKey{Name: secretName, Namespace: ns}, &caSecret)
	if errors.IsNotFound(err) {
		return
	}
	assert.NoError(f.T, err, "failed to get ca secret:%s", secretName)

	f.T.Logf("%s: deleting ca certificate %s", time.Now().UTC().Format(time.RFC3339), name)

	err = f.client.Delete(context.Background(), &caCert)
	if err != nil && !errors.IsNotFound(err) {
		f.T.Errorf("failed to delete ca certificate:%s :%v", name, err)
	}

	f.WaitUntil(fmt.Sprintf("ca certificate %s in %s is deleted", name, ns), func(ctx context.Context) (bool, error) {
		caCert := certv1.Certificate{}
		err := f.client.Get(ctx, client.ObjectKey{Name: name, Namespace: ns}, &caCert)
		return errors.IsNotFound(err), nil
	}, fns...)

	f.T.Logf("%s: deleting ca secret %s", time.Now().UTC().Format(time.RFC3339), secretName)

	err = f.client.Delete(context.Background(), &caSecret)
	if err != nil && !errors.IsNotFound(err) {
		f.T.Errorf("failed to delete ca secret:%s :%v", secretName, err)
	}

	f.WaitUntil(fmt.Sprintf("ca secret %s in %s is deleted", secretName, ns), func(ctx context.Context) (bool, error) {
		caSecret := corev1.Secret{}
		err := f.client.Get(ctx, client.ObjectKey{Name: secretName, Namespace: ns}, &caSecret)
		return errors.IsNotFound(err), nil
	}, fns...)
}

func (f Framework) CreateCAIssuer(name, secretName, ns string) *certv1.Issuer {
	issuer := certv1.Issuer{
		TypeMeta: metav1.TypeMeta{
			APIVersion: certv1.SchemeGroupVersion.String(),
			Kind:       "Issuer",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Spec: certv1.IssuerSpec{
			IssuerConfig: certv1.IssuerConfig{
				CA: &certv1.CAIssuer{
					SecretName: secretName,
				},
			},
		},
	}

	err := f.client.Patch(context.TODO(), &issuer, client.Apply, client.ForceOwnership, client.FieldOwner("e2e-test"))
	assert.NoError(f.T, err, "failed to create CA Issuer")

	f.T.Cleanup(func() {
		f.DeleteCAIssuer(name, ns, Timeout(5*time.Minute))
	})

	return &issuer
}

func (f Framework) DeleteCAIssuer(name, ns string, fns ...AssertOptionFn) {
	f.T.Helper()
	issuer := certv1.Issuer{}
	err := f.client.Get(context.TODO(), client.ObjectKey{Name: name, Namespace: ns}, &issuer)
	if errors.IsNotFound(err) {
		return
	}
	assert.NoError(f.T, err, "failed to get ca issuer:%s", name)

	f.T.Logf("%s: deleting ca issuer %s", time.Now().UTC().Format(time.RFC3339), name)

	err = f.client.Delete(context.Background(), &issuer)
	if err != nil && !errors.IsNotFound(err) {
		f.T.Errorf("failed to delete ca issuer:%s :%v", name, err)
	}

	f.WaitUntil(fmt.Sprintf("ca issuer %s in %s is deleted", name, ns), func(ctx context.Context) (bool, error) {
		issuer := certv1.Issuer{}
		err := f.client.Get(ctx, client.ObjectKey{Name: name, Namespace: ns}, &issuer)
		return errors.IsNotFound(err), nil
	}, fns...)
}

func (f Framework) CreateTLSCertificate(name, secretName, ns, issuerName string, dnsNames []string) *certv1.Certificate {
	cert := certv1.Certificate{
		TypeMeta: metav1.TypeMeta{
			APIVersion: certv1.SchemeGroupVersion.String(),
			Kind:       "Certificate",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Spec: certv1.CertificateSpec{
			SecretName: secretName,
			Duration:   &metav1.Duration{Duration: 8760 * time.Hour},
			DNSNames:   dnsNames,
			IssuerRef: certmetav1.ObjectReference{
				Name: issuerName,
				Kind: "Issuer",
			},
		},
	}

	err := f.client.Patch(context.TODO(), &cert, client.Apply, client.ForceOwnership, client.FieldOwner("e2e-test"))
	assert.NoError(f.T, err, "failed to create TLS Certificate")

	f.T.Cleanup(func() {
		f.DeleteTLSCertificate(name, name, ns, Timeout(5*time.Minute))
	})

	return &cert
}

func (f Framework) DeleteTLSCertificate(name, secretName, ns string, fns ...AssertOptionFn) {
	f.T.Helper()
	caCert := certv1.Certificate{}
	err := f.client.Get(context.TODO(), client.ObjectKey{Name: name, Namespace: ns}, &caCert)
	if errors.IsNotFound(err) {
		return
	}
	assert.NoError(f.T, err, "failed to get ca certificate :%s", name)

	caSecret := corev1.Secret{}
	err = f.client.Get(context.TODO(), client.ObjectKey{Name: secretName, Namespace: ns}, &caSecret)
	if errors.IsNotFound(err) {
		return
	}
	assert.NoError(f.T, err, "failed to get ca secret:%s", secretName)

	f.T.Logf("%s: deleting ca certificate %s", time.Now().UTC().Format(time.RFC3339), name)

	err = f.client.Delete(context.Background(), &caCert)
	if err != nil && !errors.IsNotFound(err) {
		f.T.Errorf("failed to delete ca certificate:%s :%v", name, err)
	}

	f.WaitUntil(fmt.Sprintf("ca certificate %s in %s is deleted", name, ns), func(ctx context.Context) (bool, error) {
		caCert := certv1.Certificate{}
		err := f.client.Get(ctx, client.ObjectKey{Name: name, Namespace: ns}, &caCert)
		return errors.IsNotFound(err), nil
	}, fns...)

	f.T.Logf("%s: deleting ca secret %s", time.Now().UTC().Format(time.RFC3339), secretName)

	err = f.client.Delete(context.Background(), &caSecret)
	if err != nil && !errors.IsNotFound(err) {
		f.T.Errorf("failed to delete ca secret:%s :%v", secretName, err)
	}

	f.WaitUntil(fmt.Sprintf("ca secret %s in %s is deleted", secretName, ns), func(ctx context.Context) (bool, error) {
		caSecret := corev1.Secret{}
		err := f.client.Get(ctx, client.ObjectKey{Name: secretName, Namespace: ns}, &caSecret)
		return errors.IsNotFound(err), nil
	}, fns...)
}

func (f Framework) CreateCurlPowerMonitorTestSuite(testJobName, testSAName, testNs, audience, serviceURL, caCertSecretName, caCertSecretNs string) string {
	f.CreateNamespace(testNs)

	f.CreateSA(testSAName, testNs)

	caConfigMapName := "test-monitoring-serving-certs-ca-bundle"
	f.CreateCAConfigMap(caConfigMapName, testNs, caCertSecretName, caCertSecretNs)

	curlJob := f.CreateCurlPowerMonitorJob(testJobName, testNs, testSAName, caConfigMapName, audience, serviceURL)
	f.WaitUntil(fmt.Sprintf("job %s in %s is complete", testJobName, testNs), func(ctx context.Context) (bool, error) {
		err := f.client.Get(ctx, client.ObjectKeyFromObject(curlJob), curlJob)
		if err != nil {
			return false, err
		}
		return curlJob.Status.Succeeded > 0, nil
	}, Timeout(5*time.Minute))

	logs, err := oc.Literal().From("oc logs job/%s -n %s", testJobName, testNs).Run()
	if err != nil {
		f.T.Errorf("failed to get job pod's logs: %s :%v", testJobName, err)
	}
	return logs
}

func (f Framework) CreateCAConfigMap(name, ns, caCertSecretName, caCertSecretNs string) *corev1.ConfigMap {
	newCAConfigMap := corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Data: map[string]string{},
	}
	caSecret := corev1.Secret{}
	f.AssertResourceExists(caCertSecretName, caCertSecretNs, &caSecret)
	caCertData := caSecret.Data["tls.crt"]
	newCAConfigMap.Data["service-ca.crt"] = string(caCertData)

	err := f.client.Patch(context.TODO(), &newCAConfigMap, client.Apply, client.ForceOwnership, client.FieldOwner("e2e-test"))
	assert.NoError(f.T, err, "failed to create ca bundle config map")

	f.T.Cleanup(func() {
		f.DeleteCAConfigMap(name, ns, Timeout(5*time.Minute))
	})

	return &newCAConfigMap
}

func (f Framework) DeleteCAConfigMap(name, ns string, fns ...AssertOptionFn) {
	f.T.Helper()
	caConfigMap := corev1.ConfigMap{}
	err := f.client.Get(context.TODO(), client.ObjectKey{Name: name, Namespace: ns}, &caConfigMap)
	if errors.IsNotFound(err) {
		return
	}
	assert.NoError(f.T, err, "failed to get ca bundle config map:%s", name)

	f.T.Logf("%s: deleting ca bundle config map %s", time.Now().UTC().Format(time.RFC3339), name)

	err = f.client.Delete(context.Background(), &caConfigMap)
	if err != nil && !errors.IsNotFound(err) {
		f.T.Errorf("failed to delete ca config map:%s :%v", name, err)
	}

	f.WaitUntil(fmt.Sprintf("ca bundle configmap %s in %s is deleted", name, ns), func(ctx context.Context) (bool, error) {
		caConfigMap := corev1.ConfigMap{}
		err := f.client.Get(ctx, client.ObjectKey{Name: name, Namespace: ns}, &caConfigMap)
		return errors.IsNotFound(err), nil
	}, fns...)
}

func (f Framework) CreateCurlPowerMonitorJob(name, ns, saName, caConfigMapName, audience, serviceURL string) *batchv1.Job {
	volumes := []corev1.Volume{
		k8s.VolumeFromConfigMap("ca-bundle", caConfigMapName),
		k8s.VolumeFromProjectedToken("token-vol", audience, "token"),
	}
	command := []string{
		"/bin/sh",
		"-c",
		fmt.Sprintf(
			`curl -v -s --cacert /var/run/secrets/ca/service-ca.crt -H "Authorization: Bearer $(cat /var/service-account/token)" %s`, serviceURL,
		),
	}
	jobContainers := []corev1.Container{
		{
			Name:    name,
			Image:   "curlimages/curl:latest",
			Command: command,
			VolumeMounts: []corev1.VolumeMount{
				{Name: "ca-bundle", MountPath: "/var/run/secrets/ca", ReadOnly: true},
				{Name: "token-vol", MountPath: "/var/service-account", ReadOnly: true},
			},
		},
	}
	curlJob := batchv1.Job{
		TypeMeta: metav1.TypeMeta{
			APIVersion: batchv1.SchemeGroupVersion.String(),
			Kind:       "Job",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: name,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: saName,
					RestartPolicy:      corev1.RestartPolicyNever,
					Containers:         jobContainers,
					Volumes:            volumes,
				},
			},
		},
	}

	err := f.client.Patch(context.TODO(), &curlJob, client.Apply, client.ForceOwnership, client.FieldOwner("e2e-test"))
	assert.NoError(f.T, err, "failed to create curl power monitor job")

	f.T.Cleanup(func() {
		f.DeleteCurlPowerMonitorJob(name, ns, Timeout(5*time.Minute))
	})

	return &curlJob
}

func (f Framework) DeleteCurlPowerMonitorJob(name, ns string, fns ...AssertOptionFn) {
	f.T.Helper()
	curlJob := batchv1.Job{}
	err := f.client.Get(context.TODO(), client.ObjectKey{Name: name, Namespace: ns}, &curlJob)
	if errors.IsNotFound(err) {
		return
	}
	assert.NoError(f.T, err, "failed to get curl power monitor job:%s", name)

	f.T.Logf("%s: deleting curl power monitor job %s", time.Now().UTC().Format(time.RFC3339), name)

	foregroundPolicy := metav1.DeletePropagationForeground
	err = f.client.Delete(context.Background(), &curlJob, &client.DeleteOptions{
		GracePeriodSeconds: ptr.To(int64(0)),
		PropagationPolicy:  &foregroundPolicy,
	})
	if err != nil && !errors.IsNotFound(err) {
		f.T.Errorf("failed to delete curl power monitor job:%s :%v", name, err)
	}

	f.WaitUntil(fmt.Sprintf("curl power monitor %s in %s is deleted", name, ns), func(ctx context.Context) (bool, error) {
		curlJob := batchv1.Job{}
		err := f.client.Get(ctx, client.ObjectKey{Name: name, Namespace: ns}, &curlJob)
		return errors.IsNotFound(err), nil
	}, fns...)
}

func (f Framework) CreateNamespace(name string) *corev1.Namespace {
	namespace := corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Namespace",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}

	err := f.client.Patch(context.TODO(), &namespace, client.Apply, client.ForceOwnership, client.FieldOwner("e2e-test"))
	assert.NoError(f.T, err, "failed to create namespace")

	f.T.Cleanup(func() {
		f.DeleteNamespace(name, Timeout(5*time.Minute))
	})

	return &namespace
}

func (f Framework) DeleteNamespace(name string, fns ...AssertOptionFn) {
	f.T.Helper()
	namespace := corev1.Namespace{}
	err := f.client.Get(context.TODO(), client.ObjectKey{Name: name}, &namespace)
	if errors.IsNotFound(err) {
		return
	}
	assert.NoError(f.T, err, "failed to get namespace:%s", name)

	f.T.Logf("%s: deleting namespace %s", time.Now().UTC().Format(time.RFC3339), name)

	err = f.client.Delete(context.Background(), &namespace, &client.DeleteOptions{
		PropagationPolicy: ptr.To(metav1.DeletePropagationForeground),
	})
	if err != nil && !errors.IsNotFound(err) {
		f.T.Errorf("failed to delete namespace:%s :%v", name, err)
	}

	f.WaitUntil(fmt.Sprintf("namespace %s is deleted", name), func(ctx context.Context) (bool, error) {
		namespace := corev1.Namespace{}
		err := f.client.Get(ctx, client.ObjectKey{Name: name}, &namespace)
		return errors.IsNotFound(err), nil
	}, fns...)
}

func (f Framework) CreateSA(name, ns string) *corev1.ServiceAccount {
	serviceAccount := corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
			Labels: map[string]string{
				"app.kubernetes.io/component": "exporter",
				"app.kubernetes.io/part-of":   "test-monitoring-sa",
			},
		},
	}
	err := f.client.Patch(context.TODO(), &serviceAccount, client.Apply, client.ForceOwnership, client.FieldOwner("e2e-test"))
	assert.NoError(f.T, err, "failed to create service account")

	f.T.Cleanup(func() {
		f.DeleteSA(name, ns, Timeout(5*time.Minute))
	})

	return &serviceAccount
}

func (f Framework) DeleteSA(name, ns string, fns ...AssertOptionFn) {
	f.T.Helper()
	serviceAccount := corev1.ServiceAccount{}
	err := f.client.Get(context.TODO(), client.ObjectKey{Name: name, Namespace: ns}, &serviceAccount)
	if errors.IsNotFound(err) {
		return
	}
	assert.NoError(f.T, err, "failed to get service account:%s", name)

	f.T.Logf("%s: deleting service account %s", time.Now().UTC().Format(time.RFC3339), name)

	err = f.client.Delete(context.Background(), &serviceAccount)
	if err != nil && !errors.IsNotFound(err) {
		f.T.Errorf("failed to delete service account:%s :%v", name, err)
	}

	f.WaitUntil(fmt.Sprintf("service account %s in %s is deleted", name, ns), func(ctx context.Context) (bool, error) {
		serviceAccount := corev1.ServiceAccount{}
		err := f.client.Get(ctx, client.ObjectKey{Name: name, Namespace: ns}, &serviceAccount)
		return errors.IsNotFound(err), nil
	}, fns...)
}

// func (f Framework) getJobPod(jobName, ns string) (*corev1.Pod, error) {
// 	f.T.Helper()
// 	pods := corev1.PodList{}
// 	err := f.client.List(context.TODO(), &pods, client.InNamespace(ns), client.MatchingLabels{"job-name": jobName})
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to retrieve job pods:%s :%v", jobName, err)
// 	}
// 	if len(pods.Items) == 0 {
// 		return nil, fmt.Errorf("no pods associated with job %s", jobName)
// 	}
// 	successfulPods := []corev1.Pod{}
// 	for _, pod := range pods.Items {
// 		if pod.Status.Phase == corev1.PodSucceeded {
// 			successfulPods = append(successfulPods, pod)
// 		}
// 	}
// 	if len(successfulPods) != 1 {
// 		return nil, fmt.Errorf("there should be exactly one successful pod associated with job %s", jobName)
// 	}
// 	return &successfulPods[0], nil
// }

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
