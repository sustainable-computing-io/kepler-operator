package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sustainable.computing.io/kepler-operator/pkg/api/v1alpha1"
	"github.com/sustainable.computing.io/kepler-operator/pkg/components"
	"github.com/sustainable.computing.io/kepler-operator/pkg/components/exporter"
	"github.com/sustainable.computing.io/kepler-operator/pkg/utils/k8s"
	"github.com/sustainable.computing.io/kepler-operator/pkg/utils/test"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func TestKepler_Reconciliation(t *testing.T) {
	f := test.NewFramework(t)

	// pre-condition
	f.AssertNoResourceExits("kepler", "", &v1alpha1.Kepler{}, test.NoWait())

	// when
	f.CreateKepler("kepler")
	ds := appsv1.DaemonSet{}

	// then
	f.AssertResourceExits(exporter.DaemonSetName, components.Namespace, &ds)
	f.AssertResourceExits(components.Namespace, "", &corev1.Namespace{})

	kepler := f.GetKepler("kepler")
	status := kepler.Status
	reconciled, err := k8s.FindCondition(status.Conditions, v1alpha1.Reconciled)
	assert.NoError(t, err, "unable to get reconciled condition")

	assert.Equal(t, reconciled.ObservedGeneration, kepler.Generation)
	assert.Equal(t, reconciled.Status, v1alpha1.ConditionTrue)
}

func TestKepler_Deletion(t *testing.T) {
	f := test.NewFramework(t)

	// pre-condition: ensure kepler exists
	f.CreateKepler("kepler")
	ds := appsv1.DaemonSet{}
	f.AssertResourceExits(exporter.DaemonSetName, components.Namespace, &ds)

	kepler := f.GetKepler("kepler")
	status := kepler.Status
	reconciled, err := k8s.FindCondition(status.Conditions, v1alpha1.Reconciled)
	assert.NoError(t, err, "unable to get reconciled condition")
	assert.Equal(t, reconciled.Status, v1alpha1.ConditionTrue)

	// When kepler is deleted
	f.DeleteKepler("kepler")

	// It cleans up the resources
	f.AssertNoResourceExits(exporter.DaemonSetName, components.Namespace, &appsv1.DaemonSet{})
	f.AssertNoResourceExits(components.Namespace, "", &corev1.Namespace{})
}

func TestBadKepler_Reconciliation(t *testing.T) {
	f := test.NewFramework(t)
	f.AssertNoResourceExits("invalid-name", "", &v1alpha1.Kepler{}, test.NoWait())
	f.CreateKepler("invalid-name")
	ds := appsv1.DaemonSet{}
	f.AssertNoResourceExits(exporter.DaemonSetName, components.Namespace, &ds)
}
