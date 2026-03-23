// SPDX-FileCopyrightText: 2025 The Kepler Authors
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"context"
	"time"

	. "github.com/onsi/gomega"
	"github.com/sustainable.computing.io/kepler-operator/api/v1alpha1"
	"github.com/sustainable.computing.io/kepler-operator/pkg/utils/k8s"
	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
)

// ConditionFunc checks a condition and returns (done, error).
// When done is true, the condition is met. When error is non-nil, it indicates a transient problem.
type ConditionFunc func(ctx context.Context) (bool, error)

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

// WaitUntil polls a condition function until it returns true or the timeout expires.
func (f *Framework) WaitUntil(msg string, fn ConditionFunc, fns ...AssertOptionFn) {
	opt := assertOption(fns...)

	Eventually(func(g Gomega) {
		done, err := fn(context.Background())
		if err != nil {
			g.Expect(err).NotTo(HaveOccurred(), "error while waiting for %s", msg)
		}
		g.Expect(done).To(BeTrue(), "condition not met for: %s", msg)
	}).WithTimeout(opt.WaitTimeout).WithPolling(opt.PollInterval).Should(Succeed(),
		"failed waiting for %s (timeout %v)", msg, opt.WaitTimeout)
}

// ExpectResourceExists waits for a resource to exist and populates obj with its state.
func (f *Framework) ExpectResourceExists(name, ns string, obj client.Object, fns ...AssertOptionFn) {
	opt := assertOption(fns...)
	key := client.ObjectKey{Name: name, Namespace: ns}

	Eventually(func(g Gomega) {
		err := f.client.Get(context.Background(), key, obj)
		g.Expect(err).NotTo(HaveOccurred(), "resource %s/%s should exist", ns, name)
	}).WithTimeout(opt.WaitTimeout).WithPolling(opt.PollInterval).Should(Succeed())
}

// ExpectNoResourceExists waits until the specified resource no longer exists.
func (f *Framework) ExpectNoResourceExists(name, ns string, obj client.Object, fns ...AssertOptionFn) {
	opt := assertOption(fns...)
	key := client.ObjectKey{Name: name, Namespace: ns}

	Eventually(func(g Gomega) {
		err := f.client.Get(context.Background(), key, obj)
		g.Expect(errors.IsNotFound(err)).To(BeTrue(), "resource %s/%s should not exist", ns, name)
	}).WithTimeout(opt.WaitTimeout).WithPolling(opt.PollInterval).Should(Succeed())
}

// ExpectPowerMonitorCondition waits for a PowerMonitor to reach the specified condition status.
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

// ExpectPowerMonitorInternalCondition waits for a PowerMonitorInternal to reach the specified condition status.
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

// ExpectPowerMonitorInternalStatus verifies that a PowerMonitorInternal is reconciled and available.
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
