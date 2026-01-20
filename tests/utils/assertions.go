// SPDX-FileCopyrightText: 2025 The Kepler Authors
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/gomega"
	"github.com/sustainable.computing.io/kepler-operator/api/v1alpha1"
	"github.com/sustainable.computing.io/kepler-operator/pkg/utils/k8s"
	"sigs.k8s.io/controller-runtime/pkg/client"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
)

type AssertOption struct {
	PollInterval time.Duration
	WaitTimeout  time.Duration
}
type AssertOptionFn func(*AssertOption)

func Wait(interval, timeout time.Duration) AssertOptionFn {
	return func(o *AssertOption) {
		o.PollInterval = interval
		o.WaitTimeout = timeout
	}
}

func Timeout(timeout time.Duration) AssertOptionFn {
	return func(o *AssertOption) {
		o.WaitTimeout = timeout
	}
}

func PollInterval(interval time.Duration) AssertOptionFn {
	return func(o *AssertOption) {
		o.PollInterval = interval
	}
}

func ShortWait() AssertOptionFn {
	return Wait(500*time.Millisecond, 5*time.Second)
}

func NoWait() AssertOptionFn {
	return Wait(1*time.Millisecond, 1*time.Millisecond)
}

func assertOption(fns ...AssertOptionFn) AssertOption {
	option := AssertOption{
		PollInterval: 5 * time.Second,
		WaitTimeout:  TestTimeout,
	}
	for _, fn := range fns {
		fn(&option)
	}
	return option
}

// WaitUntil waits for a condition to be true using Gomega's Eventually
func (f *Framework) WaitUntil(msg string, fn wait.ConditionWithContextFunc, fns ...AssertOptionFn) {
	opt := assertOption(fns...)
	ctx, cancel := context.WithTimeout(context.Background(), opt.WaitTimeout)
	defer cancel()

	err := wait.PollUntilContextTimeout(ctx, opt.PollInterval, opt.WaitTimeout, true, fn)
	Expect(err).NotTo(HaveOccurred(), "failed waiting for %s (timeout %v)", msg, opt.WaitTimeout)
}

// ExpectResourceExists waits for a resource to exist using Gomega's Eventually
func (f *Framework) ExpectResourceExists(name, ns string, obj client.Object, fns ...AssertOptionFn) {
	opt := assertOption(fns...)
	key := client.ObjectKey{Name: name, Namespace: ns}

	Eventually(func(g Gomega) {
		err := f.client.Get(context.Background(), key, obj)
		g.Expect(err).NotTo(HaveOccurred(), "resource %s/%s should exist", ns, name)
	}).WithTimeout(opt.WaitTimeout).WithPolling(opt.PollInterval).Should(Succeed())
}

// ExpectNoResourceExists waits for a resource to not exist using Gomega's Eventually
func (f *Framework) ExpectNoResourceExists(name, ns string, obj client.Object, fns ...AssertOptionFn) {
	opt := assertOption(fns...)
	key := client.ObjectKey{Name: name, Namespace: ns}

	Eventually(func(g Gomega) {
		err := f.client.Get(context.Background(), key, obj)
		g.Expect(errors.IsNotFound(err)).To(BeTrue(), "resource %s/%s should not exist", ns, name)
	}).WithTimeout(opt.WaitTimeout).WithPolling(opt.PollInterval).Should(Succeed())
}

// ExpectPowerMonitorCondition waits for a PowerMonitor to have a specific condition using Gomega
func (f *Framework) ExpectPowerMonitorCondition(name string, t v1alpha1.ConditionType, s v1alpha1.ConditionStatus, fns ...AssertOptionFn) *v1alpha1.PowerMonitor {
	opt := assertOption(fns...)
	pm := &v1alpha1.PowerMonitor{}

	Eventually(func(g Gomega) {
		err := f.client.Get(context.Background(), client.ObjectKey{Name: name}, pm)
		g.Expect(err).NotTo(HaveOccurred(), "powermonitor %s should exist", name)

		condition, _ := k8s.FindCondition(pm.Status.Conditions, t)
		g.Expect(condition.Status).To(Equal(s), "powermonitor %s should have condition %s=%s", name, t, s)
	}).WithTimeout(opt.WaitTimeout).WithPolling(opt.PollInterval).Should(Succeed())

	return pm
}

// ExpectPowerMonitorInternalCondition waits for a PowerMonitorInternal to have a specific condition using Gomega
func (f *Framework) ExpectPowerMonitorInternalCondition(name string, t v1alpha1.ConditionType, s v1alpha1.ConditionStatus, fns ...AssertOptionFn) *v1alpha1.PowerMonitorInternal {
	opt := assertOption(fns...)
	pmi := &v1alpha1.PowerMonitorInternal{}

	Eventually(func(g Gomega) {
		err := f.client.Get(context.Background(), client.ObjectKey{Name: name}, pmi)
		g.Expect(err).NotTo(HaveOccurred(), "powermonitorinternal %s should exist", name)

		condition, _ := k8s.FindCondition(pmi.Status.Conditions, t)
		g.Expect(condition.Status).To(Equal(s), "powermonitorinternal %s should have condition %s=%s", name, t, s)
	}).WithTimeout(opt.WaitTimeout).WithPolling(opt.PollInterval).Should(Succeed())

	return pmi
}

// ExpectPowerMonitorInternalStatus verifies the PowerMonitorInternal status using Gomega
func (f *Framework) ExpectPowerMonitorInternalStatus(name string, fns ...AssertOptionFn) {
	pmi := f.ExpectPowerMonitorInternalCondition(name, v1alpha1.Reconciled, v1alpha1.ConditionTrue, fns...)
	Expect(pmi.Spec.Kepler.Deployment.Tolerations).To(Equal([]corev1.Toleration{{Operator: "Exists"}}))

	reconciled, err := k8s.FindCondition(pmi.Status.Conditions, v1alpha1.Reconciled)
	Expect(err).NotTo(HaveOccurred(), "unable to get reconciled condition")
	Expect(reconciled.ObservedGeneration).To(Equal(pmi.Generation))
	Expect(reconciled.Status).To(Equal(v1alpha1.ConditionTrue))

	pmi = f.ExpectPowerMonitorInternalCondition(name, v1alpha1.Available, v1alpha1.ConditionTrue, fns...)
	available, err := k8s.FindCondition(pmi.Status.Conditions, v1alpha1.Available)
	Expect(err).NotTo(HaveOccurred(), "unable to get available condition")
	Expect(available.ObservedGeneration).To(Equal(pmi.Generation))
	Expect(available.Status).To(Equal(v1alpha1.ConditionTrue))
}

// ExpectDaemonSetExists waits for a DaemonSet to exist and returns it
func (f *Framework) ExpectDaemonSetExists(name, ns string, fns ...AssertOptionFn) *appsv1.DaemonSet {
	ds := &appsv1.DaemonSet{}
	f.ExpectResourceExists(name, ns, ds, fns...)
	return ds
}

// ExpectNamespace waits for a namespace to exist
func (f *Framework) ExpectNamespace(name string, fns ...AssertOptionFn) {
	ns := &corev1.Namespace{}
	f.ExpectResourceExists(name, "", ns, fns...)
}

// ExpectDaemonSetGeneration waits for a DaemonSet to have a specific generation
func (f *Framework) ExpectDaemonSetGeneration(name, ns string, expectedGeneration int64, fns ...AssertOptionFn) *appsv1.DaemonSet {
	opt := assertOption(fns...)
	ds := &appsv1.DaemonSet{}

	Eventually(func(g Gomega) {
		err := f.client.Get(context.Background(), client.ObjectKey{Name: name, Namespace: ns}, ds)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(ds.Status.ObservedGeneration).To(Equal(expectedGeneration),
			"DaemonSet %s/%s should have generation %d", ns, name, expectedGeneration)
	}).WithTimeout(opt.WaitTimeout).WithPolling(opt.PollInterval).Should(Succeed())

	return ds
}

// ExpectSecretHashChanged waits for a DaemonSet's secret hash annotation to change
func (f *Framework) ExpectSecretHashChanged(name, ns, secretName, oldHash string, fns ...AssertOptionFn) string {
	opt := assertOption(fns...)
	annotationKey := fmt.Sprintf("powermonitor.sustainable.computing.io/secret-tls-hash-%s", secretName)
	var newHash string

	Eventually(func(g Gomega) {
		ds := &appsv1.DaemonSet{}
		err := f.client.Get(context.Background(), client.ObjectKey{Name: name, Namespace: ns}, ds)
		g.Expect(err).NotTo(HaveOccurred())

		newHash = ds.Spec.Template.Annotations[annotationKey]
		g.Expect(newHash).NotTo(Equal(oldHash), "Secret hash annotation should change")
	}).WithTimeout(opt.WaitTimeout).WithPolling(opt.PollInterval).Should(Succeed())

	return newHash
}

// Deprecated: Use ExpectResourceExists instead
func (f *Framework) AssertResourceExists(name, ns string, obj client.Object, fns ...AssertOptionFn) {
	f.ExpectResourceExists(name, ns, obj, fns...)
}

// Deprecated: Use ExpectNoResourceExists instead
func (f *Framework) AssertNoResourceExists(name, ns string, obj client.Object, fns ...AssertOptionFn) {
	f.ExpectNoResourceExists(name, ns, obj, fns...)
}

// Deprecated: Use ExpectPowerMonitorInternalStatus instead
func (f *Framework) AssertPowerMonitorInternalStatus(name string, fns ...AssertOptionFn) {
	f.ExpectPowerMonitorInternalStatus(name, fns...)
}
