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
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/sustainable.computing.io/kepler-operator/pkg/api/v1alpha1"
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

func (f Framework) WaitUntil(msg string, fn wait.ConditionFunc, fns ...AssertOptionFn) {
	f.T.Helper()
	opt := assertOption(fns...)

	err := wait.PollImmediate(opt.PollInterval, opt.WaitTimeout, fn)
	assert.NoError(f.T, err, "failed waiting for %s (timeout %v)", msg, opt.WaitTimeout)
}

func (f Framework) AssertResourceExists(name, ns string, obj client.Object, fns ...AssertOptionFn) {
	f.T.Helper()
	opt := assertOption(fns...)
	key := types.NamespacedName{Name: name, Namespace: ns}

	var getErr error
	wait.PollImmediate(opt.PollInterval, opt.WaitTimeout, func() (bool, error) {
		getErr = f.client.Get(context.Background(), key, obj)
		// NOTE: return true (stop loop) if resource exists
		return getErr == nil, nil
	})

	assert.NoError(f.T, getErr, "failed to find %v (timeout %v)", key, opt.WaitTimeout)
}

func (f Framework) AssertNoResourceExists(name, ns string, obj client.Object, fns ...AssertOptionFn) {
	f.T.Helper()
	opt := assertOption(fns...)
	key := types.NamespacedName{Name: name, Namespace: ns}

	err := wait.PollImmediate(opt.PollInterval, opt.WaitTimeout, func() (bool, error) {

		getErr := f.client.Get(context.Background(), key, obj)
		// NOTE: return true (stop loop) if resource does not exist
		return errors.IsNotFound(getErr), nil
	})

	if wait.Interrupted(err) {
		f.T.Errorf("%s (%v) exists after %v", k8s.GVKName(obj), key, opt.WaitTimeout)
	}
}

func (f Framework) AssertInternalStatus(name string) {
	// the status will be updated
	ki := f.WaitUntilInternalCondition(name, v1alpha1.Reconciled, v1alpha1.ConditionTrue)
	assert.Equal(f.T, []corev1.Toleration{{Operator: "Exists"}}, ki.Spec.Exporter.Deployment.Tolerations)

	reconciled, err := k8s.FindCondition(ki.Status.Exporter.Conditions, v1alpha1.Reconciled)
	assert.NoError(f.T, err, "unable to get reconciled condition")
	assert.Equal(f.T, reconciled.ObservedGeneration, ki.Generation)
	assert.Equal(f.T, reconciled.Status, v1alpha1.ConditionTrue)
	//
	ki = f.WaitUntilInternalCondition(name, v1alpha1.Available, v1alpha1.ConditionTrue)
	available, err := k8s.FindCondition(ki.Status.Exporter.Conditions, v1alpha1.Available)
	assert.NoError(f.T, err, "unable to get available condition")
	assert.Equal(f.T, available.ObservedGeneration, ki.Generation)
	assert.Equal(f.T, available.Status, v1alpha1.ConditionTrue)

	f.WaitUntilInternalHasExpectedRunning(name)
}
