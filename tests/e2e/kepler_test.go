package e2e

import (
	"encoding/json"
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
	f.AssertResourceExists(
		exporter.DaemonSetName,
		components.Namespace,
		&ds,
		test.Timeout(10*time.Second),
	)

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
	// ensure the default toleration is set
	assert.Equal(t, []corev1.Toleration{{Operator: "Exists"}}, kepler.Spec.Exporter.Deployment.Tolerations)

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

func TestNodeSelector(t *testing.T) {
	f := test.NewFramework(t)
	// Ensure Kepler is not deployed (by any chance)
	f.AssertNoResourceExists("kepler", "", &v1alpha1.Kepler{}, test.Timeout(10*time.Second))

	nodes, err := f.GetResourceNames("node")
	assert.NoError(t, err, "failed to get node names")
	assert.NotZero(t, len(nodes), "got zero nodes")

	node := nodes[0]
	var labels k8s.StringMap = map[string]string{"e2e-test": "true"}
	err = f.AddResourceLabels("node", node, labels)
	assert.NoError(t, err, "could not label node")

	f.CreateKepler("kepler", func(k *v1alpha1.Kepler) {
		k.Spec.Exporter.Deployment.NodeSelector = labels
	})

	f.AssertResourceExists(components.Namespace, "", &corev1.Namespace{})
	ds := appsv1.DaemonSet{}
	f.AssertResourceExists(exporter.DaemonSetName, components.Namespace, &ds)

	kepler := f.WaitUntilKeplerCondition("kepler", v1alpha1.Available)
	assert.EqualValues(t, 1, kepler.Status.NumberAvailable)

	f.DeleteKepler("kepler")

	ns := components.NewKeplerNamespace()
	f.AssertNoResourceExists(ns.Name, "", ns)
	f.AssertNoResourceExists(exporter.DaemonSetName, components.Namespace, &ds)
}

func TestTaint_WithToleration(t *testing.T) {

	f := test.NewFramework(t)
	// Ensure Kepler is not deployed (by any chance)
	f.AssertNoResourceExists("kepler", "", &v1alpha1.Kepler{}, test.Timeout(10*time.Second))

	var err error
	// choose one node
	nodes := getNodes(f)
	node := nodes[0]
	taints := getTaintsForNode(f, node)

	e2eTestTaint := corev1.Taint{
		Key:    "key1",
		Value:  "value1",
		Effect: corev1.TaintEffectNoSchedule,
	}

	err = f.TaintNode(node, e2eTestTaint.ToString())
	assert.NoError(t, err, "failed to taint node %s", node)

	f.CreateKepler("kepler", func(k *v1alpha1.Kepler) {
		k.Spec.Exporter.Deployment.Tolerations = tolerateTaints(append(taints, e2eTestTaint))
	})

	f.AssertResourceExists(components.Namespace, "", &corev1.Namespace{})
	ds := appsv1.DaemonSet{}
	f.AssertResourceExists(exporter.DaemonSetName, components.Namespace, &ds)

	kepler := f.WaitUntilKeplerCondition("kepler", v1alpha1.Available)
	assert.EqualValues(t, len(nodes), kepler.Status.NumberAvailable)

	f.DeleteKepler("kepler")

	ns := components.NewKeplerNamespace()
	f.AssertNoResourceExists(ns.Name, "", ns)
	f.AssertNoResourceExists(exporter.DaemonSetName, components.Namespace, &ds)

}

func getNodes(f *test.Framework) []string {
	f.T.Logf("%s: getting nodes", time.Now().UTC().Format(time.RFC3339))
	nodes, err := f.GetResourceNames("node")
	assert.NoError(f.T, err, "failed to get node names")
	assert.NotZero(f.T, len(nodes), "got zero nodes")
	return nodes
}

func getTaintsForNode(f *test.Framework, node string) []corev1.Taint {
	f.T.Logf("%s: getting taints for node: %s", time.Now().UTC().Format(time.RFC3339), node)
	taintsStr, err := f.GetTaints(node)
	assert.NoError(f.T, err, "failed to get taint for node %s", node)
	var taints []corev1.Taint
	if taintsStr != "" {
		err = json.Unmarshal([]byte(taintsStr), &taints)
		assert.NoError(f.T, err, "failed to unmarshal taints %s", taintsStr)
	}
	return taints
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
