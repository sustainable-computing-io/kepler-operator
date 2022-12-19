package controllers

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
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
	ServiceName             = "kepler-operator-exporter"
	KeplerOperatorName      = "kepler-operator"
	KeplerOperatorNameSpace = "kepler"
	ServiceAccountName      = KeplerOperatorName
	ServiceAccountNameSpace = KeplerOperatorNameSpace
	RoleName                = KeplerOperatorName
	RoleNameSpace           = KeplerOperatorNameSpace
	RoleBindingName         = KeplerOperatorName
	RoleBindingNameSpace    = KeplerOperatorNameSpace
	DaemonSetNameSpace      = KeplerOperatorNameSpace
	ServiceNameSpace        = KeplerOperatorNameSpace
)

func generateDefaultOperatorSettings() (context.Context, *KeplerReconciler, *keplersystemv1alpha1.Kepler, collectorReconciler, logr.Logger, *KeplerClient) {
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
	return ctx, keplerReconciler, keplerInstance, r, logger, client
}

/*
	func testVerifyMainReconciler(t *testing.T, client *KeplerClient) {
		//Verify that Collector Objects have been created and verified
		testVerifyCollectorReconciler(t, client)
	}
*/
func testVerifyCollectorReconciler(t *testing.T, client *KeplerClient) {
	//Verify mock client objects exist
	resultsServiceAccount, okSA := client.NameSpacedNameToObject[KeplerKey{Name: KeplerOperatorName, Namespace: KeplerOperatorNameSpace, ObjectType: "ServiceAccount"}]
	resultsRole, okR := client.NameSpacedNameToObject[KeplerKey{Name: KeplerOperatorName, Namespace: KeplerOperatorNameSpace, ObjectType: "Role"}]
	resultsRoleBinding, okRB := client.NameSpacedNameToObject[KeplerKey{Name: KeplerOperatorName, Namespace: KeplerOperatorNameSpace, ObjectType: "RoleBinding"}]

	if !okSA || !okR || !okRB {
		t.Fatal("ServiceAccount Objects do not exist")
	}

	returnedServiceAccount, okSA := resultsServiceAccount.obj.(*corev1.ServiceAccount)
	returnedRole, okR := resultsRole.obj.(*rbacv1.Role)
	returnedRoleBinding, okRB := resultsRoleBinding.obj.(*rbacv1.RoleBinding)

	if !okSA || !okR || !okRB {
		t.Fatal("Could not convert ServiceAccount Objects")
	}

	resultsService, ok := client.NameSpacedNameToObject[KeplerKey{Name: ServiceName, Namespace: KeplerOperatorNameSpace, ObjectType: "Service"}]
	if !ok {
		t.Fatal("Service Object does not exist")
	}
	returnedService, ok := resultsService.obj.(*corev1.Service)
	if !ok {
		t.Fatal("Could not convert Service")
	}

	resultsDaemonSet, ok := client.NameSpacedNameToObject[KeplerKey{Name: DaemonSetName, Namespace: KeplerOperatorNameSpace, ObjectType: "DaemonSet"}]
	if !ok {
		t.Fatal("Daemonset Object does not exist")
	}

	returnedDaemonSet, ok := resultsDaemonSet.obj.(*appsv1.DaemonSet)
	if !ok {
		t.Fatal("Could not convert Daemonset Object")
	}

	//skip Service Monitor

	//Verify Collector related produced objects are valid

	testVerifyServiceAccountSpec(t, *returnedServiceAccount, *returnedRole, *returnedRoleBinding)
	testVerifyServiceSpec(t, *returnedService)
	//Note testVerifyDaemonSpec already ensures SA is assigned to Daemonset
	testVerifyDaemonSpec(t, *returnedServiceAccount, *returnedDaemonSet)

	//Verify Collector related cross object relationships are valid

	//Verify Service selector matches daemonset spec template labels
	//Service Selector must exist correctly to connect to daemonset
	for key, value := range returnedService.Spec.Selector {
		assert.Contains(t, returnedDaemonSet.Spec.Template.ObjectMeta.Labels, key)
		assert.Equal(t, value, returnedDaemonSet.Spec.Template.ObjectMeta.Labels[key])
	}

}

func testVerifyServiceSpec(t *testing.T, returnedService corev1.Service) {
	//TODO: CheckSetControllerReference should become a helper test function
	// check SetControllerReference has been set (all objects require owners) properly
	assert.Equal(t, 1, len(returnedService.GetOwnerReferences()))
	assert.Equal(t, KeplerOperatorName, returnedService.GetOwnerReferences()[0].Name)
	assert.Equal(t, "Kepler", returnedService.GetOwnerReferences()[0].Kind)
	//check if CreateOrUpdate Object has properly set up required fields, nested fields, and variable fields for SA
	assert.NotEmpty(t, returnedService.ObjectMeta)
	assert.Equal(t, ServiceName, returnedService.ObjectMeta.Name)
	assert.Equal(t, ServiceNameSpace, returnedService.ObjectMeta.Namespace)
	assert.NotEmpty(t, returnedService.Spec)
	assert.NotEmpty(t, returnedService.Spec.Ports)
	assert.NotEmpty(t, returnedService.Spec.Selector)
	assert.Equal(t, "None", returnedService.Spec.ClusterIP)

}

func testVerifyServiceAccountSpec(t *testing.T, returnedServiceAccount corev1.ServiceAccount, returnedRole rbacv1.Role, returnedRoleBinding rbacv1.RoleBinding) {
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

func testVerifyDaemonSpec(t *testing.T, returnedServiceAccount corev1.ServiceAccount, returnedDaemonSet appsv1.DaemonSet) {
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

	assert.Equal(t, DaemonSetName, returnedDaemonSet.ObjectMeta.Name)
	assert.Equal(t, DaemonSetNameSpace, returnedDaemonSet.ObjectMeta.Namespace)

	assert.Equal(t, DaemonSetName, returnedDaemonSet.Spec.Template.ObjectMeta.Name)
	assert.True(t, returnedDaemonSet.Spec.Template.Spec.HostNetwork)
	assert.Equal(t, returnedServiceAccount.Name, returnedDaemonSet.Spec.Template.Spec.ServiceAccountName)

	// check if daemonset obeys general rules
	//TODO: MATCH LABELS IS subset to labels. SAME WITH SELECTOR IN SERVICE
	// NEED TO MAKE SURE RELATED SERVICE CONNECTS TO EXISTING LABELS IN DAEMONSET PODS TOO
	for key, value := range returnedDaemonSet.Spec.Selector.MatchLabels {
		assert.Contains(t, returnedDaemonSet.Spec.Template.ObjectMeta.Labels, key)
		assert.Equal(t, value, returnedDaemonSet.Spec.Template.ObjectMeta.Labels[key])
	}

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
	ctx, keplerReconciler, keplerInstance, r, logger, client := generateDefaultOperatorSettings()
	r.serviceAccount = &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      KeplerOperatorName,
			Namespace: KeplerOperatorNameSpace,
		},
	}

	res, err := r.ensureDaemonSet(logger)
	//basic check
	assert.Equal(t, true, res)
	if err != nil {
		t.Fatal("DaemonSet has failed which should not happen")
	}

	results, ok := client.NameSpacedNameToObject[KeplerKey{Name: DaemonSetName, Namespace: KeplerOperatorNameSpace, ObjectType: "DaemonSet"}]
	if !ok {
		t.Fatal("Daemonset has not been properly created")
	}

	returnedDaemonSet, ok := results.obj.(*appsv1.DaemonSet)
	if ok {
		testVerifyDaemonSpec(t, *r.serviceAccount, *returnedDaemonSet)

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

	results, ok = client.NameSpacedNameToObject[KeplerKey{Name: DaemonSetName, Namespace: KeplerOperatorNameSpace, ObjectType: "DaemonSet"}]
	if !ok {
		t.Fatal("Daemonset has not been properly created")
	}

	returnedDaemonSet, ok = results.obj.(*appsv1.DaemonSet)
	if ok {
		testVerifyDaemonSpec(t, *r.serviceAccount, *returnedDaemonSet)

	} else {
		t.Fatal("Object is not DaemonSet")
	}

}

func TestEnsureServiceAccount(t *testing.T) {
	_, _, _, r, logger, client := generateDefaultOperatorSettings()

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
			testVerifyServiceAccountSpec(t, *returnedServiceAccount, *returnedRole, *returnedRoleBinding)

		} else {
			t.Fatal("Object is not ServiceAccount, Role, or RoleBinding")
		}
	}

}

func TestEnsureService(t *testing.T) {
	_, _, _, r, logger, client := generateDefaultOperatorSettings()

	numOfReconciliations := 3

	for i := 0; i < numOfReconciliations; i++ {
		res, err := r.ensureService(logger)
		//basic check
		assert.Equal(t, true, res)
		if err != nil {
			t.Fatal("Service has failed which should not happen")
		}

		resultsService, ok := client.NameSpacedNameToObject[KeplerKey{Name: ServiceName, Namespace: KeplerOperatorNameSpace, ObjectType: "Service"}]

		if !ok {
			t.Fatal("Service has not been properly created")
		}

		returnedService, ok := resultsService.obj.(*corev1.Service)
		if ok {
			testVerifyServiceSpec(t, *returnedService)

		} else {
			t.Fatal("Object is not Service")
		}
	}
}

//TODO: Test ServiceMonitor if necessary

// Test CollectorReconciler As a Whole

func TestCollectorReconciler(t *testing.T) {
	ctx, keplerReconciler, keplerInstance, _, logger, client := generateDefaultOperatorSettings()
	numOfReconciliations := 3
	for i := 0; i < numOfReconciliations; i++ {
		_, err := CollectorReconciler(ctx, keplerInstance, keplerReconciler, logger)
		if err != nil {
			t.Fatal("CollectorReconciler has failed")
		}
		//Run testVerifyCollectorReconciler
		testVerifyCollectorReconciler(t, client)

	}
}

/*
func TestMainReconciler(t *testing.T) {
	ctx, keplerReconciler, keplerInstance, _, _, client := generateDefaultOperatorSettings()
	// No Kepler Instance
	_, err := keplerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: KeplerOperatorName, Namespace: KeplerOperatorNameSpace}})

	if err != nil {
		t.Fatal("Error beside NotFoundError has occurred")
	}

	//With Kepler Instance
	err = client.Create(ctx, keplerInstance)
	if err != nil {
		t.Fatal("Failed to create Kepler Instance")
	}

	_, err = keplerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: KeplerOperatorName, Namespace: KeplerOperatorNameSpace}})

	if err != nil {
		t.Fatal("Reconcile Failed")
	}

	//Since no error has been invoked, we should check if the desired objects have been created properly
	testVerifyMainReconciler(t, client)

}
*/
