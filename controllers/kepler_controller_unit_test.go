package controllers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	keplersystemv1alpha1 "github.com/sustainable.computing.io/kepler-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	DaemonSetName           = "kepler-operator-exporter"
	KeplerOperatorName      = "kepler-operator"
	KeplerOperatorNameSpace = "kepler"
	DaemonSetComponentLabel = "exporter"
	DaemonSetNameLabel      = "kepler-exporter"
	ServiceAccountName      = KeplerOperatorName
	ServiceAccountNameSpace = KeplerOperatorNameSpace
	RoleName                = KeplerOperatorName
	RoleNameSpace           = KeplerOperatorNameSpace
	RoleBindingName         = KeplerOperatorName
	RoleBindingNameSpace    = KeplerOperatorNameSpace
)

func testVerifyServiceAccountSpec(t *testing.T, r collectorReconciler, returnedServiceAccount corev1.ServiceAccount, returnedRole rbacv1.Role, returnedRoleBinding rbacv1.RoleBinding) {
	// check SetControllerReference has been set (all objects require owners) properly for SA, Role, RoleBinding
	assert.Equal(t, 1, len(returnedServiceAccount.GetOwnerReferences()))
	assert.Equal(t, 1, len(returnedRole.GetOwnerReferences()))
	assert.Equal(t, 1, len(returnedRoleBinding.GetOwnerReferences()))
	assert.Equal(t, KeplerOperatorName, returnedServiceAccount.GetOwnerReferences()[0].Name)
	assert.Equal(t, KeplerOperatorName, returnedRole.GetOwnerReferences()[0].Name)
	assert.Equal(t, KeplerOperatorName, returnedRoleBinding.GetOwnerReferences()[0].Name)

	assert.Equal(t, "Kepler", returnedServiceAccount.GetOwnerReferences()[0].Kind)
	assert.Equal(t, "Kepler", returnedRole.GetOwnerReferences()[0].Kind)
	assert.Equal(t, "Kepler", returnedRoleBinding.GetOwnerReferences()[0].Kind)

	//check if CreateOrUpdate Object has properly set up required fields, nested fields, and variable fields for SA
	assert.NotEmpty(t, returnedServiceAccount.ObjectMeta)
	assert.Equal(t, ServiceAccountName, returnedServiceAccount.ObjectMeta.Name)
	assert.Equal(t, ServiceAccountNameSpace, returnedServiceAccount.ObjectMeta.Namespace)

	//check if CreateOrUpdate Object has properly set up required fields, nested fields, and variable fields for Role
	assert.NotEmpty(t, returnedRole.ObjectMeta)
	assert.Equal(t, RoleName, returnedRole.ObjectMeta.Name)
	assert.Equal(t, RoleNameSpace, returnedRole.ObjectMeta.Namespace)
	assert.NotEmpty(t, returnedRole.Rules)

	//check if CreateOrUpdate Object has properly set up required fields, nested fields, and variable fields for RoleBinding
	assert.NotEmpty(t, returnedRoleBinding.ObjectMeta)
	assert.Equal(t, RoleBindingName, returnedRoleBinding.ObjectMeta.Name)
	assert.Equal(t, RoleBindingNameSpace, returnedRoleBinding.ObjectMeta.Namespace)
	assert.NotEmpty(t, returnedRoleBinding.RoleRef)
	assert.NotEmpty(t, returnedRoleBinding.Subjects)
	assert.Equal(t, "ServiceAccount", returnedRoleBinding.Subjects[0].Kind)
	assert.Equal(t, returnedServiceAccount.Name, returnedRoleBinding.Subjects[0].Name)
	assert.Equal(t, "rbac.authorization.k8s.io", returnedRoleBinding.RoleRef.APIGroup)
	assert.Equal(t, "Role", returnedRoleBinding.RoleRef.Kind)
	assert.Equal(t, returnedRole.Name, returnedRoleBinding.RoleRef.Name)

}

func testVerifyDaemonSpec(t *testing.T, r collectorReconciler, returnedDaemonSet appsv1.DaemonSet) {
	// check SetControllerReference has been set (all objects require owners) properly
	assert.Equal(t, 1, len(returnedDaemonSet.GetOwnerReferences()))
	assert.Equal(t, KeplerOperatorName, returnedDaemonSet.GetOwnerReferences()[0].Name)
	assert.Equal(t, "Kepler", returnedDaemonSet.GetOwnerReferences()[0].Kind)
	// check if CreateOrUpdate Object has properly set up required fields, nested fields, and variable fields
	assert.NotEmpty(t, returnedDaemonSet.Spec)
	assert.NotEmpty(t, returnedDaemonSet.Spec.Template)
	assert.NotEmpty(t, returnedDaemonSet.Spec.Template.Spec)
	assert.NotEmpty(t, returnedDaemonSet.ObjectMeta)
	assert.NotEmpty(t, returnedDaemonSet.Spec.Template.ObjectMeta)

	assert.NotEqual(t, 0, len(returnedDaemonSet.Spec.Template.Spec.Containers))

	for _, container := range returnedDaemonSet.Spec.Template.Spec.Containers {
		assert.NotEmpty(t, container.Image)
		assert.NotEmpty(t, container.Name)

	}

	assert.Equal(t, DaemonSetName, returnedDaemonSet.Spec.Template.ObjectMeta.Name)
	assert.True(t, returnedDaemonSet.Spec.Template.Spec.HostNetwork)
	assert.Equal(t, r.serviceAccount.Name, returnedDaemonSet.Spec.Template.Spec.ServiceAccountName)

	// check if daemonset obeys general rules
	assert.Equal(t, returnedDaemonSet.Spec.Selector.MatchLabels, returnedDaemonSet.Spec.Template.ObjectMeta.Labels)
	if returnedDaemonSet.Spec.Template.Spec.RestartPolicy != "" {
		assert.Equal(t, corev1.RestartPolicyAlways, returnedDaemonSet.Spec.Template.Spec.RestartPolicy)
	}
	for _, container := range returnedDaemonSet.Spec.Template.Spec.Containers {

		for _, port := range container.Ports {
			assert.NotEmpty(t, port.ContainerPort)
			assert.Less(t, port.ContainerPort, int32(65536))
			assert.Greater(t, port.ContainerPort, int32(0))
		}

	}
	//check that probe ports correspond to an existing containe port
	// currently we assume the probe ports are integers and we only use integer ports (no referencing ports by name)
	for _, container := range returnedDaemonSet.Spec.Template.Spec.Containers {
		if container.LivenessProbe != nil {
			assert.NotEmpty(t, container.LivenessProbe.ProbeHandler)
			encountered := false
			for _, port := range container.Ports {
				if container.LivenessProbe.HTTPGet != nil {
					assert.NotEmpty(t, container.LivenessProbe.HTTPGet.Port)
					if port.ContainerPort == int32(container.LivenessProbe.HTTPGet.Port.IntValue()) {
						encountered = true
					}
				} else if container.LivenessProbe.TCPSocket != nil {
					assert.NotEmpty(t, container.LivenessProbe.TCPSocket.Port)
					if port.ContainerPort == int32(container.LivenessProbe.TCPSocket.Port.IntValue()) {
						encountered = true
					}
				} else if container.LivenessProbe.Exec != nil {
					//TODO: Include Checks
				}
			}
			assert.True(t, encountered)
		}
		//not in use
		if container.ReadinessProbe != nil {
			assert.NotEmpty(t, container.ReadinessProbe.ProbeHandler)
			encountered := false
			for _, port := range container.Ports {
				if container.ReadinessProbe.HTTPGet != nil {
					assert.NotEmpty(t, container.ReadinessProbe.HTTPGet.Port)
					if port.ContainerPort == int32(container.ReadinessProbe.HTTPGet.Port.IntValue()) {
						encountered = true
					}
				} else if container.ReadinessProbe.TCPSocket != nil {
					assert.NotEmpty(t, container.ReadinessProbe.TCPSocket.Port)
					if port.ContainerPort == int32(container.ReadinessProbe.TCPSocket.Port.IntValue()) {
						encountered = true
					}
				} else if container.ReadinessProbe.Exec != nil {
					//TODO: Include Checks
				}
			}
			assert.True(t, encountered)

		}
		//not in use
		if container.StartupProbe != nil {
			assert.NotEmpty(t, container.StartupProbe.ProbeHandler)
			encountered := false
			for _, port := range container.Ports {
				if container.StartupProbe.HTTPGet != nil {
					assert.NotEmpty(t, container.StartupProbe.HTTPGet.Port)
					if port.ContainerPort == int32(container.StartupProbe.HTTPGet.Port.IntValue()) {
						encountered = true
					}
				} else if container.StartupProbe.TCPSocket != nil {
					assert.NotEmpty(t, container.StartupProbe.TCPSocket.Port)
					if port.ContainerPort == int32(container.StartupProbe.TCPSocket.Port.IntValue()) {
						encountered = true
					}
				} else if container.StartupProbe.Exec != nil {
					//TODO: Include Checks
				}
			}
			assert.True(t, encountered)
		}
	}

	// ensure volumemounts reference existing volumes
	volumes := returnedDaemonSet.Spec.Template.Spec.Volumes
	//TODO: note that volumes that are not mounted is not allowed. Is this worth addressing?
	for _, container := range returnedDaemonSet.Spec.Template.Spec.Containers {
		encountered := false
		for _, volumeMount := range container.VolumeMounts {
			for _, volume := range volumes {
				if volumeMount.Name == volume.Name { //&& volumeMount.MountPath == volume.VolumeSource.HostPath.Path {
					encountered = true
				}
			}
		}
		assert.True(t, encountered)
	}
}

func TestEnsureDaemon(t *testing.T) {
	ctx := context.Background()
	_ = log.FromContext(ctx)

	logger := log.Log.WithValues("kepler", types.NamespacedName{Name: "kepler-operator", Namespace: "kepler"})

	scheme := runtime.NewScheme()
	_ = keplersystemv1alpha1.AddToScheme(scheme)
	client := NewClient()

	keplerReconciler := &KeplerReconciler{
		Client: client,
		Scheme: scheme,
		Log:    logger,
	}

	keplerInstance := &keplersystemv1alpha1.Kepler{
		ObjectMeta: v1.ObjectMeta{
			Name:      KeplerOperatorName,
			Namespace: KeplerOperatorNameSpace,
		},
		Spec: keplersystemv1alpha1.KeplerSpec{
			Collector: &keplersystemv1alpha1.CollectorSpec{
				Image: "quay.io/sustainable_computing_io/kepler:latest",
			},
		},
	}

	r := collectorReconciler{
		Ctx:              ctx,
		Instance:         keplerInstance,
		KeplerReconciler: *keplerReconciler,
		serviceAccount: &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      KeplerOperatorName,
				Namespace: KeplerOperatorNameSpace,
			},
		},
	}

	res, err := r.ensureDaemonSet(logger)
	//basic check
	assert.Equal(t, true, res)
	if err != nil {
		t.Fatal("DaemonSet has failed which should not happen")
	}

	results, ok := client.NameSpacedNameToObject[KeplerKey{Name: "kepler-operator-exporter", Namespace: "kepler", ObjectType: "DaemonSet"}]
	if !ok {
		t.Fatal("Daemonset has not been properly created")
	}

	returnedDaemonSet, ok := results.obj.(*appsv1.DaemonSet)
	if ok {
		testVerifyDaemonSpec(t, r, *returnedDaemonSet)

	} else {
		t.Fatal("Object is not DaemonSet")
	}

	r = collectorReconciler{
		Ctx:              ctx,
		Instance:         keplerInstance,
		KeplerReconciler: *keplerReconciler,
		serviceAccount: &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "random",
				Namespace: "random_two",
			},
		},
	}

	res, err = r.ensureDaemonSet(logger)
	//basic check
	assert.Equal(t, true, res)
	if err != nil {
		t.Fatal("DaemonSet has failed which should not happen")
	}

	results, ok = client.NameSpacedNameToObject[KeplerKey{Name: "kepler-operator-exporter", Namespace: "kepler", ObjectType: "DaemonSet"}]
	if !ok {
		t.Fatal("Daemonset has not been properly created")
	}

	returnedDaemonSet, ok = results.obj.(*appsv1.DaemonSet)
	if ok {
		testVerifyDaemonSpec(t, r, *returnedDaemonSet)

	} else {
		t.Fatal("Object is not DaemonSet")
	}

}

func TestEnsureServiceAccount(t *testing.T) {
	ctx := context.Background()
	_ = log.FromContext(ctx)

	logger := log.Log.WithValues("kepler", types.NamespacedName{Name: "kepler-operator", Namespace: "kepler"})

	scheme := runtime.NewScheme()
	_ = keplersystemv1alpha1.AddToScheme(scheme)
	client := NewClient()

	keplerReconciler := &KeplerReconciler{
		Client: client,
		Scheme: scheme,
		Log:    logger,
	}

	keplerInstance := &keplersystemv1alpha1.Kepler{
		ObjectMeta: v1.ObjectMeta{
			Name:      KeplerOperatorName,
			Namespace: KeplerOperatorNameSpace,
		},
		Spec: keplersystemv1alpha1.KeplerSpec{
			Collector: &keplersystemv1alpha1.CollectorSpec{
				Image: "quay.io/sustainable_computing_io/kepler:latest",
			},
		},
	}

	r := collectorReconciler{
		Ctx:              ctx,
		Instance:         keplerInstance,
		KeplerReconciler: *keplerReconciler,
	}

	numOfReconciliations := 3
	for i := 0; i < numOfReconciliations; i++ {
		//should also affect role and role binding
		res, err := r.ensureServiceAccount(logger)
		//basic check
		assert.Equal(t, true, res)
		if err != nil {
			t.Fatal("ServiceAccountReconciler has failed which should not happen")
		}

		resultsServiceAccount, okSA := client.NameSpacedNameToObject[KeplerKey{Name: KeplerOperatorName, Namespace: KeplerOperatorNameSpace, ObjectType: "ServiceAccount"}]
		resultsRole, okR := client.NameSpacedNameToObject[KeplerKey{Name: KeplerOperatorName, Namespace: KeplerOperatorNameSpace, ObjectType: "Role"}]
		resultsRoleBinding, okRB := client.NameSpacedNameToObject[KeplerKey{Name: KeplerOperatorName, Namespace: KeplerOperatorNameSpace, ObjectType: "RoleBinding"}]
		if !okSA {
			t.Fatal("ServiceAccount has not been properly created")
		}
		if !okR {
			t.Fatal("Role has not been properly created")
		}
		if !okRB {
			t.Fatal("RoleBinding has not been properly created")
		}

		returnedServiceAccount, okSA := resultsServiceAccount.obj.(*corev1.ServiceAccount)
		returnedRole, okR := resultsRole.obj.(*rbacv1.Role)
		returnedRoleBinding, okRB := resultsRoleBinding.obj.(*rbacv1.RoleBinding)

		if okSA && okR && okRB {
			testVerifyServiceAccountSpec(t, r, *returnedServiceAccount, *returnedRole, *returnedRoleBinding)

		} else {
			t.Fatal("Object is not ServiceAccount, Role, or RoleBinding")
		}
	}

}
