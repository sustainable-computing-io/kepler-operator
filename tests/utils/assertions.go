// SPDX-FileCopyrightText: 2025 The Kepler Authors
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"context"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/sustainable.computing.io/kepler-operator/api/v1alpha1"
	"github.com/sustainable.computing.io/kepler-operator/pkg/utils/k8s"
	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
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

func (f Framework) WaitUntil(msg string, fn wait.ConditionWithContextFunc, fns ...AssertOptionFn) {
	f.T.Helper()
	opt := assertOption(fns...)
	ctx, cancel := context.WithTimeout(context.Background(), opt.WaitTimeout)
	defer cancel()

	err := wait.PollUntilContextTimeout(ctx, opt.PollInterval, opt.WaitTimeout, true, fn)
	assert.NoError(f.T, err, "failed waiting for %s (timeout %v)", msg, opt.WaitTimeout)
}

func (f Framework) AssertResourceExists(name, ns string, obj client.Object, fns ...AssertOptionFn) {
	f.T.Helper()
	opt := assertOption(fns...)
	key := types.NamespacedName{Name: name, Namespace: ns}

	var getErr error
	ctx, cancel := context.WithTimeout(context.Background(), opt.WaitTimeout)
	defer cancel()

	wait.PollUntilContextTimeout(ctx, opt.PollInterval, opt.WaitTimeout, true, func(ctx context.Context) (bool, error) {
		getErr = f.client.Get(ctx, key, obj)
		// NOTE: return true (stop loop) if resource exists
		return getErr == nil, nil
	})

	assert.NoError(f.T, getErr, "failed to find %v (timeout %v)", key, opt.WaitTimeout)
}

func (f Framework) AssertNoResourceExists(name, ns string, obj client.Object, fns ...AssertOptionFn) {
	f.T.Helper()
	opt := assertOption(fns...)
	key := types.NamespacedName{Name: name, Namespace: ns}

	ctx, cancel := context.WithTimeout(context.Background(), opt.WaitTimeout)
	defer cancel()

	err := wait.PollUntilContextTimeout(ctx, opt.PollInterval, opt.WaitTimeout, true, func(ctx context.Context) (bool, error) {
		getErr := f.client.Get(ctx, key, obj)
		// NOTE: return true (stop loop) if resource does not exist
		return errors.IsNotFound(getErr), nil
	})
	if err != nil {
		f.T.Errorf("%s (%v) exists after %v", k8s.GVKName(obj), key, opt.WaitTimeout)
	}
}

func (f Framework) AssertPowerMonitorInternalStatus(name string, fns ...AssertOptionFn) {
	pmi := f.WaitUntilPowerMonitorInternalCondition(name, v1alpha1.Reconciled, v1alpha1.ConditionTrue, fns...)
	assert.Equal(f.T, []corev1.Toleration{{Operator: "Exists"}}, pmi.Spec.Kepler.Deployment.Tolerations)

	reconciled, err := k8s.FindCondition(pmi.Status.Conditions, v1alpha1.Reconciled)
	assert.NoError(f.T, err, "unable to get reconciled condition")
	assert.Equal(f.T, reconciled.ObservedGeneration, pmi.Generation)
	assert.Equal(f.T, reconciled.Status, v1alpha1.ConditionTrue)

	pmi = f.WaitUntilPowerMonitorInternalCondition(name, v1alpha1.Available, v1alpha1.ConditionTrue, fns...)
	available, err := k8s.FindCondition(pmi.Status.Conditions, v1alpha1.Available)
	assert.NoError(f.T, err, "unable to get available condition")
	assert.Equal(f.T, available.ObservedGeneration, pmi.Generation)
	assert.Equal(f.T, available.Status, v1alpha1.ConditionTrue)
}
