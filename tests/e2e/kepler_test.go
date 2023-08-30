package e2e

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/sustainable.computing.io/kepler-operator/pkg/api/v1alpha1"
	"github.com/sustainable.computing.io/kepler-operator/pkg/components"
	"github.com/sustainable.computing.io/kepler-operator/pkg/components/exporter"
	"github.com/sustainable.computing.io/kepler-operator/pkg/utils/k8s"
	"github.com/sustainable.computing.io/kepler-operator/pkg/utils/test"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func TestKepler_Deletion(t *testing.T) {
	f := test.NewFramework(t)

	// pre-condition: ensure kepler exists
	f.CreateKepler("kepler")
	f.WaitUntilKeplerCondition("kepler", v1alpha1.Available)

	//
	ds := appsv1.DaemonSet{}
	f.AssertResourceExists(exporter.DaemonSetName, components.Namespace, &ds, test.Timeout(10*time.Second))

	f.DeleteKepler("kepler")

	ns := components.NewKeplerNamespace()
	f.AssertNoResourceExists(ns.Name, "", ns)
	f.AssertNoResourceExists(exporter.DaemonSetName, components.Namespace, &ds)
}

func TestKepler_Reconciliation(t *testing.T) {
	f := test.NewFramework(t)

	// pre-condition
	f.AssertNoResourceExists("kepler", "", &v1alpha1.Kepler{}, test.NoWait())

	// when
	f.CreateKepler("kepler")

	// then
	f.AssertResourceExists(components.Namespace, "", &corev1.Namespace{})
	ds := appsv1.DaemonSet{}
	f.AssertResourceExists(exporter.DaemonSetName, components.Namespace, &ds)

	kepler := f.WaitUntilKeplerCondition("kepler", v1alpha1.Reconciled)
	reconciled, err := k8s.FindCondition(kepler.Status.Conditions, v1alpha1.Reconciled)
	assert.NoError(t, err, "unable to get reconciled condition")
	assert.Equal(t, reconciled.ObservedGeneration, kepler.Generation)
	assert.Equal(t, reconciled.Status, v1alpha1.ConditionTrue)

	kepler = f.WaitUntilKeplerCondition("kepler", v1alpha1.Available)
	available, err := k8s.FindCondition(kepler.Status.Conditions, v1alpha1.Available)
	assert.NoError(t, err, "unable to get available condition")
	assert.Equal(t, available.ObservedGeneration, kepler.Generation)
	assert.Equal(t, available.Status, v1alpha1.ConditionTrue)

}

func TestBadKepler_Reconciliation(t *testing.T) {
	f := test.NewFramework(t)
	// Ensure Kepler is not deployed (by any chance)
	f.AssertNoResourceExists("kepler", "", &v1alpha1.Kepler{}, test.Timeout(10*time.Second))
	f.AssertNoResourceExists("invalid-name", "", &v1alpha1.Kepler{}, test.NoWait())
	f.CreateKepler("invalid-name")

	ds := appsv1.DaemonSet{}
	f.AssertNoResourceExists(exporter.DaemonSetName, components.Namespace, &ds)
}
