// SPDX-FileCopyrightText: 2025 The Kepler Authors
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"golang.org/x/exp/slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sustainable.computing.io/kepler-operator/api/v1alpha1"
	"github.com/sustainable.computing.io/kepler-operator/internal/controller"
	"github.com/sustainable.computing.io/kepler-operator/pkg/utils/k8s"
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
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/ptr"
)

// default ForeverTestTimeout is 30 seconds, some tests fail because they take
// more than 30s, some times, close to 2 minutes, so we set it to 2 minutes.
//
// NOTE: if a specific test case requires more than 2 minutes, use Timeout()
// function to set the timeout for just that assertion.
const TestTimeout = 2 * time.Minute

// Framework provides test utilities for e2e tests.
// It uses Ginkgo/Gomega internally for test organization and assertions.
type Framework struct {
	client client.Client
}

type frameworkFn func(*Framework)

// NewFramework creates a new test framework
func NewFramework(fns ...frameworkFn) *Framework {
	f := &Framework{}
	for _, fn := range fns {
		fn(f)
	}
	if f.client == nil {
		f.client = newClient(newScheme())
	}
	return f
}

// NewGinkgoFramework is an alias for NewFramework for backward compatibility
// Deprecated: Use NewFramework instead
func NewGinkgoFramework(fns ...frameworkFn) *Framework {
	return NewFramework(fns...)
}

func WithClient(c client.Client) frameworkFn {
	return func(f *Framework) {
		f.client = c
	}
}

func (f *Framework) Client() client.Client {
	return f.client
}

func newScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	Expect(corev1.AddToScheme(scheme)).To(Succeed())
	Expect(appsv1.AddToScheme(scheme)).To(Succeed())
	Expect(secv1.AddToScheme(scheme)).To(Succeed())
	Expect(rbacv1.AddToScheme(scheme)).To(Succeed())
	Expect(v1alpha1.AddToScheme(scheme)).To(Succeed())
	Expect(certv1.AddToScheme(scheme)).To(Succeed())
	Expect(batchv1.AddToScheme(scheme)).To(Succeed())
	return scheme
}

var once sync.Once

func newClient(scheme *runtime.Scheme) client.Client {
	once.Do(func() {
		ctrl.SetLogger(zap.New())
	})
	cfg := config.GetConfigOrDie()
	c, err := client.New(cfg, client.Options{Scheme: scheme})
	Expect(err).NotTo(HaveOccurred(), "failed to create client")
	return c
}

type (
	powermonitorinternalFn func(*v1alpha1.PowerMonitorInternal)
	powermonitorFn         func(*v1alpha1.PowerMonitor)
)

func (f *Framework) Patch(obj client.Object) error {
	GinkgoWriter.Printf("%s: creating/updating object %s\n", time.Now().UTC().Format(time.RFC3339), obj.GetName())

	// Clear managedFields to avoid patch conflicts when updating objects retrieved from cluster
	obj.SetManagedFields(nil)

	return f.client.Patch(context.TODO(), obj, client.Apply,
		client.ForceOwnership, client.FieldOwner("e2e-test"),
	)
}

func (f *Framework) GetPowerMonitor(name string) *v1alpha1.PowerMonitor {
	pm := v1alpha1.PowerMonitor{}
	f.ExpectResourceExists(name, "", &pm)

	// Get does not set the type meta information, setting this manually here
	// to help with functions like Patch() that require the type meta
	pm.TypeMeta = metav1.TypeMeta{
		APIVersion: v1alpha1.GroupVersion.String(),
		Kind:       "PowerMonitor",
	}
	return &pm
}

func (f *Framework) NewPowerMonitor(name string, fns ...powermonitorFn) v1alpha1.PowerMonitor {
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

func (f *Framework) CreatePowerMonitor(name string, fns ...powermonitorFn) *v1alpha1.PowerMonitor {
	pm := f.NewPowerMonitor(name, fns...)
	GinkgoWriter.Printf("%s: creating/updating powermonitor %s\n", time.Now().UTC().Format(time.RFC3339), name)
	err := f.client.Patch(context.TODO(), &pm, client.Apply,
		client.ForceOwnership, client.FieldOwner("e2e-test"),
	)
	Expect(err).NotTo(HaveOccurred(), "failed to create powermonitor")

	DeferCleanup(func() {
		GinkgoWriter.Printf("cleanup: deleting powermonitor %s\n", name)
		f.DeletePowerMonitor(name)
	})

	return &pm
}

func (f *Framework) DeletePowerMonitor(name string) {
	pm := v1alpha1.PowerMonitor{}
	pmi := v1alpha1.PowerMonitorInternal{}

	err := f.client.Get(context.Background(), client.ObjectKey{Name: name}, &pm)
	if errors.IsNotFound(err) {
		return
	}
	Expect(err).NotTo(HaveOccurred(), "failed to get powermonitor: %s", name)

	// Get PMI to obtain the deployment namespace in order to wait for deletion for namespace to happen
	err = f.client.Get(context.Background(), client.ObjectKey{Name: name}, &pmi)
	if errors.IsNotFound(err) {
		return
	}
	Expect(err).NotTo(HaveOccurred(), "failed to get power-monitor-internal: %s", name)

	GinkgoWriter.Printf("%s: deleting powermonitor %s\n", time.Now().UTC().Format(time.RFC3339), name)

	err = f.client.Delete(context.Background(), &pm)
	if err != nil && !errors.IsNotFound(err) {
		Fail(fmt.Sprintf("failed to delete powermonitor %s: %v", name, err))
	}

	f.WaitUntil(fmt.Sprintf("powermonitor %s is deleted", name), func(ctx context.Context) (bool, error) {
		GinkgoWriter.Printf("Waiting for powermonitor %s to be deleted\n", name)
		pm := v1alpha1.PowerMonitor{}
		err := f.client.Get(ctx, client.ObjectKey{Name: name}, &pm)
		return errors.IsNotFound(err), nil
	})

	ns := pmi.Spec.Kepler.Deployment.Namespace
	f.WaitUntil(fmt.Sprintf("namespace %s should not exist", ns), func(ctx context.Context) (bool, error) {
		GinkgoWriter.Printf("Waiting for namespace %s to be deleted\n", ns)
		namespace := corev1.Namespace{}
		err := f.client.Get(ctx, client.ObjectKey{Name: ns}, &namespace)
		return errors.IsNotFound(err), nil
	})
}

func (f *Framework) GetPowerMonitorInternal(name string) *v1alpha1.PowerMonitorInternal {
	pmi := v1alpha1.PowerMonitorInternal{}
	f.ExpectResourceExists(name, "", &pmi)

	// Get does not set the type meta information, setting this manually here
	// to help with functions like Patch() that require the type meta
	pmi.TypeMeta = metav1.TypeMeta{
		APIVersion: v1alpha1.GroupVersion.String(),
		Kind:       "PowerMonitorInternal",
	}
	return &pmi
}

func (f *Framework) CreatePowerMonitorInternal(name string, fns ...powermonitorinternalFn) *v1alpha1.PowerMonitorInternal {
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

	GinkgoWriter.Printf("%s: creating/updating power-monitor-internal %s\n", time.Now().UTC().Format(time.RFC3339), name)
	err := f.client.Patch(context.TODO(), &pmi, client.Apply,
		client.ForceOwnership, client.FieldOwner("e2e-test"),
	)
	Expect(err).NotTo(HaveOccurred(), "failed to create power-monitor-internal")

	DeferCleanup(func() {
		GinkgoWriter.Printf("cleanup: deleting powermonitorinternal %s\n", name)
		f.DeletePowerMonitorInternal(name, Timeout(3*time.Minute))
	})

	return &pmi
}

func (f *Framework) DeletePowerMonitorInternal(name string, fns ...AssertOptionFn) {
	pmi := v1alpha1.PowerMonitorInternal{}
	err := f.client.Get(context.TODO(), client.ObjectKey{Name: name}, &pmi)
	if errors.IsNotFound(err) {
		return
	}
	Expect(err).NotTo(HaveOccurred(), "failed to get power-monitor-internal: %s", name)

	ns := pmi.Spec.Kepler.Deployment.Namespace

	GinkgoWriter.Printf("%s: deleting power-monitor-internal %s\n", time.Now().UTC().Format(time.RFC3339), name)

	err = f.client.Delete(context.Background(), &pmi)
	if err != nil && !errors.IsNotFound(err) {
		Fail(fmt.Sprintf("failed to delete power-monitor-internal %s: %v", name, err))
	}

	f.WaitUntil(fmt.Sprintf("power-monitor-internal %s is deleted", name), func(ctx context.Context) (bool, error) {
		GinkgoWriter.Printf("Waiting for power-monitor-internal %s to be deleted\n", name)
		pmi := v1alpha1.PowerMonitorInternal{}
		err := f.client.Get(ctx, client.ObjectKey{Name: name}, &pmi)
		return errors.IsNotFound(err), nil
	}, fns...)

	f.WaitUntil(fmt.Sprintf("namespace %s should not exist", ns), func(ctx context.Context) (bool, error) {
		GinkgoWriter.Printf("Waiting for namespace %s to be deleted\n", ns)
		namespace := corev1.Namespace{}
		err := f.client.Get(ctx, client.ObjectKey{Name: ns}, &namespace)
		return errors.IsNotFound(err), nil
	})
}

// WaitUntilPowerMonitorInternalCondition waits for a PowerMonitorInternal to have a specific condition
func (f *Framework) WaitUntilPowerMonitorInternalCondition(name string, t v1alpha1.ConditionType, s v1alpha1.ConditionStatus, fns ...AssertOptionFn) *v1alpha1.PowerMonitorInternal {
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

// WaitUntilPowerMonitorCondition waits for a PowerMonitor to have a specific condition
func (f *Framework) WaitUntilPowerMonitorCondition(name string, t v1alpha1.ConditionType, s v1alpha1.ConditionStatus, fns ...AssertOptionFn) *v1alpha1.PowerMonitor {
	pm := v1alpha1.PowerMonitor{}
	f.WaitUntil(fmt.Sprintf("powermonitor %s is %s", name, t),
		func(ctx context.Context) (bool, error) {
			err := f.client.Get(ctx, client.ObjectKey{Name: name}, &pm)
			if errors.IsNotFound(err) {
				return true, fmt.Errorf("powermonitor %s is not found", name)
			}

			condition, _ := k8s.FindCondition(pm.Status.Conditions, t)
			return condition.Status == s, nil
		}, fns...)
	return &pm
}

// WaitForResource waits for a resource to exist and returns true if it exists, false if timeout
func (f *Framework) WaitForResource(name, namespace string, obj client.Object, fns ...AssertOptionFn) bool {
	var exists bool
	f.WaitUntil(fmt.Sprintf("resource %s in namespace %s to exist", name, namespace),
		func(ctx context.Context) (bool, error) {
			key := client.ObjectKey{Name: name, Namespace: namespace}
			err := f.client.Get(ctx, key, obj)
			if errors.IsNotFound(err) {
				exists = false
				return true, nil // Stop waiting, resource doesn't exist
			}
			if err != nil {
				return false, err // Continue waiting on other errors
			}
			exists = true
			return true, nil // Stop waiting, resource exists
		}, fns...)
	return exists
}

func (f *Framework) AddResourceLabels(kind, name string, l map[string]string) error {
	if kind != "node" {
		return fmt.Errorf("AddResourceLabels only supports 'node' kind, got: %s", kind)
	}

	node := &corev1.Node{}
	err := f.client.Get(context.Background(), client.ObjectKey{Name: name}, node)
	if err != nil {
		return fmt.Errorf("failed to get node %s: %w", name, err)
	}

	if node.Labels == nil {
		node.Labels = make(map[string]string)
	}

	// Store original labels for cleanup
	originalLabels := make(map[string]string)
	for k := range l {
		if v, exists := node.Labels[k]; exists {
			originalLabels[k] = v
		}
	}

	// Add new labels
	for k, v := range l {
		node.Labels[k] = v
	}

	err = f.client.Update(context.Background(), node)
	if err != nil {
		return fmt.Errorf("failed to update node %s labels: %w", name, err)
	}

	DeferCleanup(func() {
		node := &corev1.Node{}
		err := f.client.Get(context.Background(), client.ObjectKey{Name: name}, node)
		if err != nil {
			GinkgoWriter.Printf("Warning: failed to get node %s for cleanup: %v\n", name, err)
			return
		}

		// Remove added labels or restore original values
		for k := range l {
			if origVal, hadOriginal := originalLabels[k]; hadOriginal {
				node.Labels[k] = origVal
			} else {
				delete(node.Labels, k)
			}
		}

		err = f.client.Update(context.Background(), node)
		if err != nil {
			GinkgoWriter.Printf("Warning: failed to cleanup node %s labels: %v\n", name, err)
		}
	})

	return nil
}

func (f *Framework) RemoveResourceLabels(kind, name string, l []string) error {
	if kind != "node" {
		return fmt.Errorf("RemoveResourceLabels only supports 'node' kind, got: %s", kind)
	}

	node := &corev1.Node{}
	err := f.client.Get(context.Background(), client.ObjectKey{Name: name}, node)
	if err != nil {
		return fmt.Errorf("failed to get node %s: %w", name, err)
	}

	if node.Labels == nil {
		return nil // Nothing to remove
	}

	// Remove labels
	for _, label := range l {
		delete(node.Labels, label)
	}

	err = f.client.Update(context.Background(), node)
	if err != nil {
		return fmt.Errorf("failed to update node %s labels: %w", name, err)
	}

	return nil
}

func (f *Framework) WithPowerMonitorNodeSelector(label map[string]string) func(k *v1alpha1.PowerMonitor) {
	return func(pm *v1alpha1.PowerMonitor) {
		pm.Spec.Kepler.Deployment.NodeSelector = label
	}
}

func (f *Framework) TaintNode(node, taintStr string) error {
	// Parse taint string (format: "key=value:effect")
	parts := strings.Split(taintStr, ":")
	if len(parts) != 2 {
		return fmt.Errorf("invalid taint format: %s (expected key=value:effect)", taintStr)
	}

	keyValue := strings.Split(parts[0], "=")
	if len(keyValue) != 2 {
		return fmt.Errorf("invalid taint key=value format: %s", parts[0])
	}

	taint := corev1.Taint{
		Key:    keyValue[0],
		Value:  keyValue[1],
		Effect: corev1.TaintEffect(parts[1]),
	}

	nodeObj := &corev1.Node{}
	err := f.client.Get(context.Background(), client.ObjectKey{Name: node}, nodeObj)
	if err != nil {
		return fmt.Errorf("failed to get node %s: %w", node, err)
	}

	// Add taint
	nodeObj.Spec.Taints = append(nodeObj.Spec.Taints, taint)

	err = f.client.Update(context.Background(), nodeObj)
	if err != nil {
		return fmt.Errorf("failed to update node %s taints: %w", node, err)
	}

	DeferCleanup(func() {
		// Remove taint
		nodeObj := &corev1.Node{}
		err := f.client.Get(context.Background(), client.ObjectKey{Name: node}, nodeObj)
		if err != nil {
			GinkgoWriter.Printf("Warning: failed to get node %s for taint cleanup: %v\n", node, err)
			return
		}

		// Filter out the taint we added
		var newTaints []corev1.Taint
		for _, t := range nodeObj.Spec.Taints {
			if t.Key != taint.Key || t.Value != taint.Value || t.Effect != taint.Effect {
				newTaints = append(newTaints, t)
			}
		}
		nodeObj.Spec.Taints = newTaints

		err = f.client.Update(context.Background(), nodeObj)
		if err != nil {
			GinkgoWriter.Printf("Warning: failed to remove taint from node %s: %v\n", node, err)
		}
	})

	return err
}

func (f *Framework) WithPowerMonitorTolerations(taints []corev1.Taint) func(k *v1alpha1.PowerMonitor) {
	return func(pm *v1alpha1.PowerMonitor) {
		pm.Spec.Kepler.Deployment.Tolerations = tolerateTaints(taints)
	}
}

func (f *Framework) WithPowerMonitorSecuritySet(mode v1alpha1.SecurityMode, allowedSANames []string) func(k *v1alpha1.PowerMonitor) {
	return func(pm *v1alpha1.PowerMonitor) {
		pm.Spec.Kepler.Deployment.Security.Mode = mode
		pm.Spec.Kepler.Deployment.Security.AllowedSANames = allowedSANames
	}
}

func (f *Framework) WithAdditionalConfigMaps(configMapNames []string) func(k *v1alpha1.PowerMonitor) {
	return func(pm *v1alpha1.PowerMonitor) {
		var configMapRefs []v1alpha1.ConfigMapRef
		for _, name := range configMapNames {
			configMapRefs = append(configMapRefs, v1alpha1.ConfigMapRef{Name: name})
		}
		pm.Spec.Kepler.Config.AdditionalConfigMaps = configMapRefs
	}
}

func (f *Framework) WithPowerMonitorSecrets(secrets []v1alpha1.SecretRef) func(k *v1alpha1.PowerMonitor) {
	return func(pm *v1alpha1.PowerMonitor) {
		pm.Spec.Kepler.Deployment.Secrets = secrets
	}
}

func (f *Framework) WithMaxTerminated(maxTerminated int32) func(k *v1alpha1.PowerMonitor) {
	return func(pm *v1alpha1.PowerMonitor) {
		pm.Spec.Kepler.Config.MaxTerminated = &maxTerminated
	}
}

func (f *Framework) WithStaleness(staleness string) func(k *v1alpha1.PowerMonitor) {
	return func(pm *v1alpha1.PowerMonitor) {
		duration, _ := time.ParseDuration(staleness)
		pm.Spec.Kepler.Config.Staleness = &metav1.Duration{Duration: duration}
	}
}

func (f *Framework) WithSampleRate(sampleRate string) func(k *v1alpha1.PowerMonitor) {
	return func(pm *v1alpha1.PowerMonitor) {
		duration, _ := time.ParseDuration(sampleRate)
		pm.Spec.Kepler.Config.SampleRate = &metav1.Duration{Duration: duration}
	}
}

// CreateTestPowerMonitor creates a PowerMonitor with standard test configuration
// for VM or non-VM environments. It handles the common pattern of creating
// additional ConfigMaps for VM environments and applies standard security settings.
func (f *Framework) CreateTestPowerMonitor(name string, runningOnVM bool, additionalFns ...powermonitorFn) *v1alpha1.PowerMonitor {
	configMapName := "my-custom-config"

	// Combine the base configuration with any additional functions
	var fns []powermonitorFn

	if runningOnVM {
		// For VM environments, add the additional ConfigMap
		fns = append(fns, f.WithAdditionalConfigMaps([]string{configMapName}))
	}

	// Always add the security settings
	fns = append(fns, f.WithPowerMonitorSecuritySet(
		v1alpha1.SecurityModeNone,
		[]string{},
	))

	// Add any additional functions passed by the caller
	fns = append(fns, additionalFns...)

	// Create the PowerMonitor
	pm := f.CreatePowerMonitor(name, fns...)

	// For VM environments, create and patch the additional ConfigMap
	if runningOnVM {
		// Wait for the PowerMonitor to be reconciled
		_ = f.WaitUntilPowerMonitorCondition(name, v1alpha1.Reconciled, v1alpha1.ConditionFalse)

		cfm := f.NewAdditionalConfigMap(configMapName, controller.PowerMonitorDeploymentNS, `dev:
  fake-cpu-meter:
    enabled: true`)
		err := f.Patch(cfm)
		Expect(err).NotTo(HaveOccurred(), "failed to create additional config map for VM environment")
	}

	return pm
}

// CreateTestPowerMonitorInternal creates a PowerMonitorInternal with standard test configuration
// for VM or non-VM environments. It handles the common pattern of creating
// additional ConfigMaps for VM environments and applies standard configuration.
func (f *Framework) CreateTestPowerMonitorInternal(name string, testNs string, runningOnVM bool, keplerImage string, kubeRbacProxyImage string, cluster k8s.Cluster, securityMode v1alpha1.SecurityMode, allowedSANames []string, additionalFns ...powermonitorinternalFn) *v1alpha1.PowerMonitorInternal {
	configMapName := "my-custom-config"

	// Set up the builder with standard configuration
	b := PowerMonitorInternalBuilder{}
	var fns []powermonitorinternalFn

	// Add standard configuration
	fns = append(fns,
		b.WithNamespace(testNs),
		b.WithKeplerImage(keplerImage),
		b.WithKubeRbacProxyImage(kubeRbacProxyImage),
		b.WithCluster(cluster),
		b.WithSecuritySet(securityMode, allowedSANames),
	)

	if runningOnVM {
		// For VM environments, add the additional ConfigMap
		fns = append(fns, b.WithAdditionalConfigMaps([]string{configMapName}))
	}

	// Add any additional functions passed by the caller
	fns = append(fns, additionalFns...)

	// Create the PowerMonitorInternal
	pmi := f.CreatePowerMonitorInternal(name, fns...)

	// For VM environments, create and patch the additional ConfigMap
	if runningOnVM {
		// Wait for the PowerMonitorInternal to be reconciled
		f.WaitForNamespace(pmi.Spec.Kepler.Deployment.Namespace)
		_ = f.WaitUntilPowerMonitorInternalCondition(name, v1alpha1.Reconciled, v1alpha1.ConditionFalse)

		cfm := f.NewAdditionalConfigMap(configMapName, testNs, `dev:
  fake-cpu-meter:
    enabled: true`)
		err := f.Patch(cfm)
		Expect(err).NotTo(HaveOccurred(), "failed to create additional config map for VM environment")
	}

	return pmi
}

func (f *Framework) NewAdditionalConfigMap(configMapName, namespace, config string) *corev1.ConfigMap {
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

func (f *Framework) GetSchedulableNodes() []corev1.Node {
	var nodes corev1.NodeList
	err := f.client.List(context.TODO(), &nodes)
	Expect(err).NotTo(HaveOccurred(), "failed to get nodes")

	var ret []corev1.Node
	for _, n := range nodes.Items {
		if isSchedulableNode(n) {
			ret = append(ret, n)
		}
	}
	return ret
}

// DeployOpenshiftCerts creates cert-manager resources for TLS certificate management.
// Note: This function assumes cert-manager is already installed (which happens during
// 'make cluster-up' via hack/cluster.sh).
func (f *Framework) DeployOpenshiftCerts(serviceName, serviceNamespace, clusterIssuerName, caCertName, caCertSecretName, pmIssuerName, tlsCertName, tlsCertSecretName string) {
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
		fmt.Sprintf("%s.%s.svc", serviceName, serviceNamespace),
	})
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

func (f *Framework) CreateSelfSignedClusterIssuer(name string) *certv1.ClusterIssuer {
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
	Expect(err).NotTo(HaveOccurred(), "failed to create ClusterIssuer")

	DeferCleanup(func() {
		f.DeleteSelfSignedClusterIssuer(name, Timeout(5*time.Minute))
	})
	return &issuer
}

func (f *Framework) DeleteSelfSignedClusterIssuer(name string, fns ...AssertOptionFn) {
	issuer := certv1.ClusterIssuer{}
	err := f.client.Get(context.TODO(), client.ObjectKey{Name: name}, &issuer)
	if errors.IsNotFound(err) {
		return
	}
	Expect(err).NotTo(HaveOccurred(), "failed to get self signed cluster issuer: %s", name)

	GinkgoWriter.Printf("%s: deleting self signed cluster issuer %s\n", time.Now().UTC().Format(time.RFC3339), name)

	err = f.client.Delete(context.Background(), &issuer)
	if err != nil && !errors.IsNotFound(err) {
		Fail(fmt.Sprintf("failed to delete self signed cluster issuer %s: %v", name, err))
	}

	f.WaitUntil(fmt.Sprintf("self signed cluster %s is deleted", name), func(ctx context.Context) (bool, error) {
		issuer := certv1.ClusterIssuer{}
		err := f.client.Get(ctx, client.ObjectKey{Name: name}, &issuer)
		return errors.IsNotFound(err), nil
	}, fns...)
}

func (f *Framework) CreateCACertificate(name, secretName, ns, issuerName string) *certv1.Certificate {
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
	Expect(err).NotTo(HaveOccurred(), "failed to create CA Certificate")

	DeferCleanup(func() {
		f.DeleteCACertificate(name, name, ns, Timeout(5*time.Minute))
	})

	return &cert
}

func (f *Framework) DeleteCACertificate(name, secretName, ns string, fns ...AssertOptionFn) {
	caCert := certv1.Certificate{}
	err := f.client.Get(context.TODO(), client.ObjectKey{Name: name, Namespace: ns}, &caCert)
	if errors.IsNotFound(err) {
		return
	}
	Expect(err).NotTo(HaveOccurred(), "failed to get ca certificate: %s", name)

	caSecret := corev1.Secret{}
	err = f.client.Get(context.TODO(), client.ObjectKey{Name: secretName, Namespace: ns}, &caSecret)
	if errors.IsNotFound(err) {
		return
	}
	Expect(err).NotTo(HaveOccurred(), "failed to get ca secret: %s", secretName)

	GinkgoWriter.Printf("%s: deleting ca certificate %s\n", time.Now().UTC().Format(time.RFC3339), name)

	err = f.client.Delete(context.Background(), &caCert)
	if err != nil && !errors.IsNotFound(err) {
		Fail(fmt.Sprintf("failed to delete ca certificate %s: %v", name, err))
	}

	f.WaitUntil(fmt.Sprintf("ca certificate %s in %s is deleted", name, ns), func(ctx context.Context) (bool, error) {
		caCert := certv1.Certificate{}
		err := f.client.Get(ctx, client.ObjectKey{Name: name, Namespace: ns}, &caCert)
		return errors.IsNotFound(err), nil
	}, fns...)

	GinkgoWriter.Printf("%s: deleting ca secret %s\n", time.Now().UTC().Format(time.RFC3339), secretName)

	err = f.client.Delete(context.Background(), &caSecret)
	if err != nil && !errors.IsNotFound(err) {
		Fail(fmt.Sprintf("failed to delete ca secret %s: %v", secretName, err))
	}

	f.WaitUntil(fmt.Sprintf("ca secret %s in %s is deleted", secretName, ns), func(ctx context.Context) (bool, error) {
		caSecret := corev1.Secret{}
		err := f.client.Get(ctx, client.ObjectKey{Name: secretName, Namespace: ns}, &caSecret)
		return errors.IsNotFound(err), nil
	}, fns...)
}

func (f *Framework) CreateCAIssuer(name, secretName, ns string) *certv1.Issuer {
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
	Expect(err).NotTo(HaveOccurred(), "failed to create CA Issuer")

	DeferCleanup(func() {
		f.DeleteCAIssuer(name, ns, Timeout(5*time.Minute))
	})

	return &issuer
}

func (f *Framework) DeleteCAIssuer(name, ns string, fns ...AssertOptionFn) {
	issuer := certv1.Issuer{}
	err := f.client.Get(context.TODO(), client.ObjectKey{Name: name, Namespace: ns}, &issuer)
	if errors.IsNotFound(err) {
		return
	}
	Expect(err).NotTo(HaveOccurred(), "failed to get ca issuer: %s", name)

	GinkgoWriter.Printf("%s: deleting ca issuer %s\n", time.Now().UTC().Format(time.RFC3339), name)

	err = f.client.Delete(context.Background(), &issuer)
	if err != nil && !errors.IsNotFound(err) {
		Fail(fmt.Sprintf("failed to delete ca issuer %s: %v", name, err))
	}

	f.WaitUntil(fmt.Sprintf("ca issuer %s in %s is deleted", name, ns), func(ctx context.Context) (bool, error) {
		issuer := certv1.Issuer{}
		err := f.client.Get(ctx, client.ObjectKey{Name: name, Namespace: ns}, &issuer)
		return errors.IsNotFound(err), nil
	}, fns...)
}

func (f *Framework) CreateTLSCertificate(name, secretName, ns, issuerName string, dnsNames []string) *certv1.Certificate {
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
	Expect(err).NotTo(HaveOccurred(), "failed to create TLS Certificate")

	DeferCleanup(func() {
		f.DeleteTLSCertificate(name, name, ns, Timeout(5*time.Minute))
	})

	return &cert
}

func (f *Framework) DeleteTLSCertificate(name, secretName, ns string, fns ...AssertOptionFn) {
	caCert := certv1.Certificate{}
	err := f.client.Get(context.TODO(), client.ObjectKey{Name: name, Namespace: ns}, &caCert)
	if errors.IsNotFound(err) {
		return
	}
	Expect(err).NotTo(HaveOccurred(), "failed to get ca certificate: %s", name)

	caSecret := corev1.Secret{}
	err = f.client.Get(context.TODO(), client.ObjectKey{Name: secretName, Namespace: ns}, &caSecret)
	if errors.IsNotFound(err) {
		return
	}
	Expect(err).NotTo(HaveOccurred(), "failed to get ca secret: %s", secretName)

	GinkgoWriter.Printf("%s: deleting ca certificate %s\n", time.Now().UTC().Format(time.RFC3339), name)

	err = f.client.Delete(context.Background(), &caCert)
	if err != nil && !errors.IsNotFound(err) {
		Fail(fmt.Sprintf("failed to delete ca certificate %s: %v", name, err))
	}

	f.WaitUntil(fmt.Sprintf("ca certificate %s in %s is deleted", name, ns), func(ctx context.Context) (bool, error) {
		caCert := certv1.Certificate{}
		err := f.client.Get(ctx, client.ObjectKey{Name: name, Namespace: ns}, &caCert)
		return errors.IsNotFound(err), nil
	}, fns...)

	GinkgoWriter.Printf("%s: deleting ca secret %s\n", time.Now().UTC().Format(time.RFC3339), secretName)

	err = f.client.Delete(context.Background(), &caSecret)
	if err != nil && !errors.IsNotFound(err) {
		Fail(fmt.Sprintf("failed to delete ca secret %s: %v", secretName, err))
	}

	f.WaitUntil(fmt.Sprintf("ca secret %s in %s is deleted", secretName, ns), func(ctx context.Context) (bool, error) {
		caSecret := corev1.Secret{}
		err := f.client.Get(ctx, client.ObjectKey{Name: secretName, Namespace: ns}, &caSecret)
		return errors.IsNotFound(err), nil
	}, fns...)
}

func (f *Framework) CreateCurlPowerMonitorTestSuite(testJobName, testSAName, testNs, audience, serviceURL, caCertSecretName, caCertSecretNs string) string {
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

	logs, err := f.GetJobLogs(testJobName, testNs)
	if err != nil {
		Fail(fmt.Sprintf("failed to get job pod's logs %s: %v", testJobName, err))
	}
	return logs
}

func (f *Framework) CreateCAConfigMap(name, ns, caCertSecretName, caCertSecretNs string) *corev1.ConfigMap {
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
	f.ExpectResourceExists(caCertSecretName, caCertSecretNs, &caSecret)
	caCertData := caSecret.Data["tls.crt"]
	newCAConfigMap.Data["service-ca.crt"] = string(caCertData)

	err := f.client.Patch(context.TODO(), &newCAConfigMap, client.Apply, client.ForceOwnership, client.FieldOwner("e2e-test"))
	Expect(err).NotTo(HaveOccurred(), "failed to create ca bundle config map")

	DeferCleanup(func() {
		f.DeleteCAConfigMap(name, ns, Timeout(5*time.Minute))
	})

	return &newCAConfigMap
}

func (f *Framework) DeleteCAConfigMap(name, ns string, fns ...AssertOptionFn) {
	caConfigMap := corev1.ConfigMap{}
	err := f.client.Get(context.TODO(), client.ObjectKey{Name: name, Namespace: ns}, &caConfigMap)
	if errors.IsNotFound(err) {
		return
	}
	Expect(err).NotTo(HaveOccurred(), "failed to get ca bundle config map: %s", name)

	GinkgoWriter.Printf("%s: deleting ca bundle config map %s\n", time.Now().UTC().Format(time.RFC3339), name)

	err = f.client.Delete(context.Background(), &caConfigMap)
	if err != nil && !errors.IsNotFound(err) {
		Fail(fmt.Sprintf("failed to delete ca config map %s: %v", name, err))
	}

	f.WaitUntil(fmt.Sprintf("ca bundle configmap %s in %s is deleted", name, ns), func(ctx context.Context) (bool, error) {
		caConfigMap := corev1.ConfigMap{}
		err := f.client.Get(ctx, client.ObjectKey{Name: name, Namespace: ns}, &caConfigMap)
		return errors.IsNotFound(err), nil
	}, fns...)
}

func (f *Framework) CreateCurlPowerMonitorJob(name, ns, saName, caConfigMapName, audience, serviceURL string) *batchv1.Job {
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
	Expect(err).NotTo(HaveOccurred(), "failed to create curl power monitor job")

	DeferCleanup(func() {
		f.DeleteCurlPowerMonitorJob(name, ns, Timeout(5*time.Minute))
	})

	return &curlJob
}

func (f *Framework) DeleteCurlPowerMonitorJob(name, ns string, fns ...AssertOptionFn) {
	curlJob := batchv1.Job{}
	err := f.client.Get(context.TODO(), client.ObjectKey{Name: name, Namespace: ns}, &curlJob)
	if errors.IsNotFound(err) {
		return
	}
	Expect(err).NotTo(HaveOccurred(), "failed to get curl power monitor job: %s", name)

	GinkgoWriter.Printf("%s: deleting curl power monitor job %s\n", time.Now().UTC().Format(time.RFC3339), name)

	foregroundPolicy := metav1.DeletePropagationForeground
	err = f.client.Delete(context.Background(), &curlJob, &client.DeleteOptions{
		GracePeriodSeconds: ptr.To(int64(0)),
		PropagationPolicy:  &foregroundPolicy,
	})
	if err != nil && !errors.IsNotFound(err) {
		Fail(fmt.Sprintf("failed to delete curl power monitor job %s: %v", name, err))
	}

	f.WaitUntil(fmt.Sprintf("curl power monitor %s in %s is deleted", name, ns), func(ctx context.Context) (bool, error) {
		curlJob := batchv1.Job{}
		err := f.client.Get(ctx, client.ObjectKey{Name: name, Namespace: ns}, &curlJob)
		return errors.IsNotFound(err), nil
	}, fns...)
}

func (f *Framework) CreateNamespace(name string) *corev1.Namespace {
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
	Expect(err).NotTo(HaveOccurred(), "failed to create namespace")

	DeferCleanup(func() {
		f.DeleteNamespace(name, Timeout(5*time.Minute))
	})

	return &namespace
}

func (f *Framework) DeleteNamespace(name string, fns ...AssertOptionFn) {
	namespace := corev1.Namespace{}
	err := f.client.Get(context.TODO(), client.ObjectKey{Name: name}, &namespace)
	if errors.IsNotFound(err) {
		return
	}
	Expect(err).NotTo(HaveOccurred(), "failed to get namespace: %s", name)

	GinkgoWriter.Printf("%s: deleting namespace %s\n", time.Now().UTC().Format(time.RFC3339), name)

	err = f.client.Delete(context.Background(), &namespace, &client.DeleteOptions{
		PropagationPolicy: ptr.To(metav1.DeletePropagationForeground),
	})
	if err != nil && !errors.IsNotFound(err) {
		Fail(fmt.Sprintf("failed to delete namespace %s: %v", name, err))
	}

	f.WaitUntil(fmt.Sprintf("namespace %s is deleted", name), func(ctx context.Context) (bool, error) {
		namespace := corev1.Namespace{}
		err := f.client.Get(ctx, client.ObjectKey{Name: name}, &namespace)
		return errors.IsNotFound(err), nil
	}, fns...)
}

func (f *Framework) CreateSA(name, ns string) *corev1.ServiceAccount {
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
	Expect(err).NotTo(HaveOccurred(), "failed to create service account")

	DeferCleanup(func() {
		f.DeleteSA(name, ns, Timeout(5*time.Minute))
	})

	return &serviceAccount
}

func (f *Framework) DeleteSA(name, ns string, fns ...AssertOptionFn) {
	serviceAccount := corev1.ServiceAccount{}
	err := f.client.Get(context.TODO(), client.ObjectKey{Name: name, Namespace: ns}, &serviceAccount)
	if errors.IsNotFound(err) {
		return
	}
	Expect(err).NotTo(HaveOccurred(), "failed to get service account: %s", name)

	GinkgoWriter.Printf("%s: deleting service account %s\n", time.Now().UTC().Format(time.RFC3339), name)

	err = f.client.Delete(context.Background(), &serviceAccount)
	if err != nil && !errors.IsNotFound(err) {
		Fail(fmt.Sprintf("failed to delete service account %s: %v", name, err))
	}

	f.WaitUntil(fmt.Sprintf("service account %s in %s is deleted", name, ns), func(ctx context.Context) (bool, error) {
		serviceAccount := corev1.ServiceAccount{}
		err := f.client.Get(ctx, client.ObjectKey{Name: name, Namespace: ns}, &serviceAccount)
		return errors.IsNotFound(err), nil
	}, fns...)
}

// WaitForNamespace waits for a namespace to be created and available
func (f *Framework) WaitForNamespace(name string, fns ...AssertOptionFn) {
	f.WaitUntil(fmt.Sprintf("namespace %s is created", name), func(ctx context.Context) (bool, error) {
		ns := corev1.Namespace{}
		err := f.client.Get(ctx, client.ObjectKey{Name: name}, &ns)
		return err == nil, nil
	}, fns...)
}

// ContainerWithName finds a container by name in the given list of containers
// Returns the container if found, or an error if not found
func (f *Framework) ContainerWithName(containers []corev1.Container, name string) (*corev1.Container, error) {
	for i, container := range containers {
		if container.Name == name {
			return &containers[i], nil
		}
	}
	return nil, fmt.Errorf("container with name %q not found", name)
}

// GetJobLogs retrieves logs from the first pod of a job
func (f *Framework) GetJobLogs(jobName, namespace string) (string, error) {
	// Get the job to find its pods
	job := &batchv1.Job{}
	err := f.client.Get(context.Background(), client.ObjectKey{Name: jobName, Namespace: namespace}, job)
	if err != nil {
		return "", fmt.Errorf("failed to get job %s/%s: %w", namespace, jobName, err)
	}

	// List pods owned by this job
	podList := &corev1.PodList{}
	err = f.client.List(context.Background(), podList,
		client.InNamespace(namespace),
		client.MatchingLabels(job.Spec.Selector.MatchLabels),
	)
	if err != nil {
		return "", fmt.Errorf("failed to list pods for job %s/%s: %w", namespace, jobName, err)
	}

	if len(podList.Items) == 0 {
		return "", fmt.Errorf("no pods found for job %s/%s", namespace, jobName)
	}

	// Get logs from the first pod
	pod := podList.Items[0]

	// Use the raw Kubernetes client for logs
	cfg := config.GetConfigOrDie()
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return "", fmt.Errorf("failed to create kubernetes clientset: %w", err)
	}

	req := clientset.CoreV1().Pods(namespace).GetLogs(pod.Name, &corev1.PodLogOptions{})
	podLogs, err := req.Stream(context.Background())
	if err != nil {
		return "", fmt.Errorf("failed to get logs for pod %s/%s: %w", namespace, pod.Name, err)
	}
	defer podLogs.Close()

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(podLogs)
	if err != nil {
		return "", fmt.Errorf("failed to read logs from pod %s/%s: %w", namespace, pod.Name, err)
	}

	return buf.String(), nil
}

func (f *Framework) CreateTestSecret(name, ns string, data map[string]string) *corev1.Secret {
	secret := corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
			Labels: map[string]string{
				"app.kubernetes.io/component": "test",
				"app.kubernetes.io/part-of":   "e2e-test-secrets",
			},
		},
		Type:       corev1.SecretTypeOpaque,
		StringData: data,
	}
	err := f.client.Patch(context.TODO(), &secret, client.Apply, client.ForceOwnership, client.FieldOwner("e2e-test"))
	Expect(err).NotTo(HaveOccurred(), "failed to create test secret")
	DeferCleanup(func() {
		f.DeleteTestSecret(name, ns, Timeout(5*time.Minute))
	})
	return &secret
}

func (f *Framework) DeleteTestSecret(name, ns string, fns ...AssertOptionFn) {
	secret := corev1.Secret{}
	err := f.client.Get(context.TODO(), client.ObjectKey{Name: name, Namespace: ns}, &secret)
	if errors.IsNotFound(err) {
		return
	}
	Expect(err).NotTo(HaveOccurred(), "failed to get test secret: %s", name)

	GinkgoWriter.Printf("%s: deleting test secret %s\n", time.Now().UTC().Format(time.RFC3339), name)

	err = f.client.Delete(context.Background(), &secret)
	if err != nil && !errors.IsNotFound(err) {
		Fail(fmt.Sprintf("failed to delete test secret %s: %v", name, err))
	}

	f.WaitUntil(fmt.Sprintf("test secret %s in %s is deleted", name, ns), func(ctx context.Context) (bool, error) {
		secret := corev1.Secret{}
		err := f.client.Get(ctx, client.ObjectKey{Name: name, Namespace: ns}, &secret)
		return errors.IsNotFound(err), nil
	}, fns...)
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

func (f *Framework) WaitForOpenshiftCerts(serviceName, serviceNamespace, tlsCertSecretName string) {
	f.WaitUntil("OpenShift TLS secret is created", func(ctx context.Context) (bool, error) {
		tlsSecret := corev1.Secret{}
		err := f.client.Get(ctx, client.ObjectKey{
			Name:      tlsCertSecretName,
			Namespace: serviceNamespace,
		}, &tlsSecret)
		if err != nil {
			return false, nil
		}

		// Check if the secret has the required TLS data
		if len(tlsSecret.Data["tls.crt"]) == 0 || len(tlsSecret.Data["tls.key"]) == 0 {
			return false, nil
		}

		return true, nil
	})
}

func (f *Framework) CreateCAConfigMapFromOpenshiftServiceCert(name, ns, tlsCertSecretName, tlsCertSecretNs string) *corev1.ConfigMap {
	newCAConfigMap := corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
			Annotations: map[string]string{
				"service.beta.openshift.io/inject-cabundle": "true",
			},
		},
		Data: map[string]string{},
	}

	err := f.Patch(&newCAConfigMap)
	Expect(err).NotTo(HaveOccurred(), "failed to create CA ConfigMap with injection annotation")

	f.WaitUntil("OpenShift injects service CA into ConfigMap", func(ctx context.Context) (bool, error) {
		caConfigMap := corev1.ConfigMap{}
		err := f.client.Get(ctx, client.ObjectKey{
			Name:      name,
			Namespace: ns,
		}, &caConfigMap)
		if err != nil {
			return false, nil
		}

		// Check if OpenShift has injected the CA certificate
		serviceCaCert := caConfigMap.Data["service-ca.crt"]
		if len(serviceCaCert) == 0 {
			return false, nil
		}

		return true, nil
	})

	return &newCAConfigMap
}

func (f *Framework) CreateCurlPowerMonitorTestSuiteForOpenShift(testJobName, testSAName, testNs, audience, serviceURL, caCertConfigMapName, caCertConfigMapNs string) string {
	f.CreateNamespace(testNs)

	f.CreateSA(testSAName, testNs)

	caConfigMapName := "test-monitoring-serving-certs-ca-bundle"
	f.CreateCAConfigMapFromOpenshiftServiceCert(caConfigMapName, testNs, caCertConfigMapName, caCertConfigMapNs)

	curlJob := f.CreateCurlPowerMonitorJob(testJobName, testNs, testSAName, caConfigMapName, audience, serviceURL)
	f.WaitUntil(fmt.Sprintf("job %s in %s is complete", testJobName, testNs), func(ctx context.Context) (bool, error) {
		err := f.client.Get(ctx, client.ObjectKeyFromObject(curlJob), curlJob)
		if err != nil {
			return false, err
		}
		return curlJob.Status.Succeeded > 0, nil
	}, Timeout(5*time.Minute))

	logs, err := f.GetJobLogs(testJobName, testNs)
	if err != nil {
		Fail(fmt.Sprintf("failed to get job pod's logs %s: %v", testJobName, err))
	}
	return logs
}
