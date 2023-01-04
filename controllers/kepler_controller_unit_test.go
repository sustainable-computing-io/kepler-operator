package controllers

import (
	"context"

	"github.com/go-logr/logr"
	monitoring "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/stretchr/testify/assert"
	keplersystemv1alpha1 "github.com/sustainable.computing.io/kepler-operator/api/v1alpha1"

	"testing"

	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/log"

	//"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	DaemonSetName               = KeplerOperatorName + "-exporter"
	ServiceName                 = KeplerOperatorName + "-exporter"
	KeplerOperatorName          = "kepler-operator"
	KeplerOperatorNameSpace     = "kepler"
	ServiceAccountName          = KeplerOperatorName
	ServiceAccountNameSpace     = KeplerOperatorNameSpace
	ClusterRoleName             = "kepler-clusterrole"
	ClusterRoleNameSpace        = ""
	ClusterRoleBindingName      = "kepler-clusterrole-binding"
	ClusterRoleBindingNameSpace = ""
	DaemonSetNameSpace          = KeplerOperatorNameSpace
	ServiceNameSpace            = KeplerOperatorNameSpace
	CollectorConfigMapName      = KeplerOperatorName + "-exporter-cfm"
	CollectorConfigMapNameSpace = KeplerOperatorNameSpace
)

func generateDefaultOperatorSettings() (context.Context, *KeplerReconciler, *keplersystemv1alpha1.Kepler, logr.Logger, client.Client) {
	ctx := context.Background()
	_ = log.FromContext(ctx)
	logger := log.Log.WithValues("kepler", types.NamespacedName{Name: "kepler-operator", Namespace: "kepler"})

	keplerInstance := &keplersystemv1alpha1.Kepler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      KeplerOperatorName,
			Namespace: KeplerOperatorNameSpace,
		},
		Spec: keplersystemv1alpha1.KeplerSpec{
			Collector: &keplersystemv1alpha1.CollectorSpec{
				Image: "quay.io/sustainable_computing_io/kepler:latest",
			},
		},
	}

	keplerobjs := []runtime.Object{keplerInstance}

	s := scheme.Scheme
	s.AddKnownTypes(keplersystemv1alpha1.SchemeBuilder.GroupVersion, keplerInstance)

	clientBuilder := fake.NewClientBuilder()
	clientBuilder = clientBuilder.WithRuntimeObjects(keplerobjs...)
	clientBuilder = clientBuilder.WithScheme(s)
	cl := clientBuilder.Build()
	monitoring.AddToScheme(s)
	keplerReconciler := &KeplerReconciler{Client: cl, Scheme: s, Log: logger}

	return ctx, keplerReconciler, keplerInstance, logger, cl
}

func CheckSetControllerReference(OwnerName string, OwnerKind string, obj client.Object) bool {
	for _, ownerReference := range obj.GetOwnerReferences() {
		if ownerReference.Name == OwnerName && ownerReference.Kind == OwnerKind {
			//owner has been set properly
			return true
		}
	}
	return false
}

func testVerifyCollectorReconciler(t *testing.T, ctx context.Context, client client.Client) {
	//Verify mock client objects exist
	foundServiceAccount := &corev1.ServiceAccount{}
	foundClusterRole := &rbacv1.ClusterRole{}
	foundClusterRoleBinding := &rbacv1.ClusterRoleBinding{}
	serviceAccountError := client.Get(ctx, types.NamespacedName{Name: KeplerOperatorName, Namespace: KeplerOperatorNameSpace}, foundServiceAccount)
	if serviceAccountError != nil {
		t.Fatalf("service account was not stored")
	}
	clusterRoleError := client.Get(ctx, types.NamespacedName{Name: ClusterRoleName, Namespace: ClusterRoleNameSpace}, foundClusterRole)
	if clusterRoleError != nil {
		t.Fatalf("cluster role was not stored")
	}
	clusterRoleBindingError := client.Get(ctx, types.NamespacedName{Name: ClusterRoleBindingName, Namespace: ClusterRoleBindingNameSpace}, foundClusterRoleBinding)
	if clusterRoleBindingError != nil {
		t.Fatalf("cluster role binding was not stored")
	}

	foundService := &corev1.Service{}
	serviceError := client.Get(ctx, types.NamespacedName{Name: ServiceName, Namespace: KeplerOperatorNameSpace}, foundService)

	if serviceError != nil {
		t.Fatal("service was not stored")
	}
	foundDaemonSet := &appsv1.DaemonSet{}
	daemonSetError := client.Get(ctx, types.NamespacedName{Name: DaemonSetName, Namespace: KeplerOperatorNameSpace}, foundDaemonSet)
	if daemonSetError != nil {
		t.Fatal("daemon Object was not stored")
	}

	foundConfigMap := &corev1.ConfigMap{}
	configMapError := client.Get(ctx, types.NamespacedName{Name: CollectorConfigMapName, Namespace: CollectorConfigMapNameSpace}, foundConfigMap)
	if configMapError != nil {
		t.Fatal("config map was not stored")
	}

	//skip Service Monitor

	//Verify Collector related produced objects are valid

	//testVerifyServiceAccountSpec(t, *foundServiceAccount, *foundClusterRole, *foundClusterRoleBinding)
	testVerifyServiceSpec(t, *foundService)
	//Note testVerifyDaemonSpec already ensures SA is assigned to Daemonset
	testVerifyDaemonSpec(t, *foundServiceAccount, *foundDaemonSet)
	testVerifyConfigMap(t, *foundConfigMap)

	//Verify Collector related cross object relationships are valid

	//Verify Service selector matches daemonset spec template labels
	//Service Selector must exist correctly to connect to daemonset
	for key, value := range foundService.Spec.Selector {
		assert.Contains(t, foundDaemonSet.Spec.Template.ObjectMeta.Labels, key)
		assert.Equal(t, value, foundDaemonSet.Spec.Template.ObjectMeta.Labels[key])
	}
	//Verify ConfigMap exists in Daemonset Volumes
	encounteredConfigMapVolume := false
	for _, volume := range foundDaemonSet.Spec.Template.Spec.Volumes {
		if volume.VolumeSource.ConfigMap != nil {
			//found configmap
			if foundConfigMap.ObjectMeta.Name == volume.VolumeSource.ConfigMap.Name {
				encounteredConfigMapVolume = true
			}
		}
	}
	assert.True(t, encounteredConfigMapVolume)

}

func testVerifyConfigMap(t *testing.T, returnedConfigMap corev1.ConfigMap) {
	// check SetControllerReference has been set (all objects require owners) properly
	result := CheckSetControllerReference(KeplerOperatorName, "Kepler", &returnedConfigMap)
	if !result {
		t.Fatal("Failed to Set Controller Reference")
	}
	//check if ConfigMap contains proper datamap
	assert.NotEmpty(t, returnedConfigMap.Data)
	assert.Equal(t, KeplerOperatorNameSpace, returnedConfigMap.Data["KEPLER_NAMESPACE"])
}

func testVerifyServiceSpec(t *testing.T, returnedService corev1.Service) {
	// check SetControllerReference has been set (all objects require owners) properly
	result := CheckSetControllerReference(KeplerOperatorName, "Kepler", &returnedService)
	if !result {
		t.Fatal("Failed to Set Controller Reference")
	}
	//check if CreateOrUpdate Object has properly set up required fields, nested fields, and variable fields for SA
	assert.NotEmpty(t, returnedService.ObjectMeta)
	assert.Equal(t, ServiceName, returnedService.ObjectMeta.Name)
	assert.Equal(t, ServiceNameSpace, returnedService.ObjectMeta.Namespace)
	assert.NotEmpty(t, returnedService.Spec)
	assert.NotEmpty(t, returnedService.Spec.Ports)
	assert.NotEmpty(t, returnedService.Spec.Selector)
	assert.Equal(t, "None", returnedService.Spec.ClusterIP)

}

/*
	func testVerifyServiceAccountSpec(t *testing.T, returnedServiceAccount corev1.ServiceAccount, returnedRole rbacv1.ClusterRole, returnedRoleBinding rbacv1.ClusterRoleBinding) {
		// check SetControllerReference has been set (all objects require owners) properly for SA, Role, RoleBinding
		assert.Equal(t, 1, len(returnedServiceAccount.GetOwnerReferences()))
		//assert.Equal(t, 1, len(returnedRole.GetOwnerReferences()))
		//assert.Equal(t, 1, len(returnedRoleBinding.GetOwnerReferences()))
		assert.Equal(t, KeplerOperatorName, returnedServiceAccount.GetOwnerReferences()[0].Name)
		//assert.Equal(t, KeplerOperatorName, returnedRole.GetOwnerReferences()[0].Name)
		//assert.Equal(t, KeplerOperatorName, returnedRoleBinding.GetOwnerReferences()[0].Name)

		assert.Equal(t, "Kepler", returnedServiceAccount.GetOwnerReferences()[0].Kind)
		//assert.Equal(t, "Kepler", returnedRole.GetOwnerReferences()[0].Kind)
		//assert.Equal(t, "Kepler", returnedRoleBinding.GetOwnerReferences()[0].Kind)

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
		assert.Equal(t, "ClusterRole", returnedRoleBinding.RoleRef.Kind)
		assert.Equal(t, returnedRole.Name, returnedRoleBinding.RoleRef.Name)

}
*/
func testVerifyDaemonSpec(t *testing.T, returnedServiceAccount corev1.ServiceAccount, returnedDaemonSet appsv1.DaemonSet) {
	// check SetControllerReference has been set (all objects require owners) properly
	result := CheckSetControllerReference(KeplerOperatorName, "Kepler", &returnedDaemonSet)
	if !result {
		t.Fatal("Failed to Set Controller Reference")
	}
	// check if CreateOrUpdate Object has properly set up required fields, nested fields, and variable fields
	assert.NotEmpty(t, returnedDaemonSet.Spec)
	assert.NotEmpty(t, returnedDaemonSet.Spec.Template)
	assert.NotEmpty(t, returnedDaemonSet.Spec.Template.Spec)
	assert.NotEmpty(t, returnedDaemonSet.ObjectMeta)
	assert.NotEmpty(t, returnedDaemonSet.Spec.Template.ObjectMeta)

	assert.NotEqual(t, 0, len(returnedDaemonSet.Spec.Template.Spec.Containers))

	for _, container := range returnedDaemonSet.Spec.Template.Spec.Containers {
		//check security
		if container.Name == "kepler-exporter" {
			assert.True(t, *container.SecurityContext.Privileged)
		}
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

func TestEnsureKeplerOperator(t *testing.T) {
	ctx := context.Background()
	_ = log.FromContext(ctx)

	logger := log.Log.WithValues("kepler", types.NamespacedName{Name: "kepler-operator", Namespace: "kepler"})

	keplerInstance := &keplersystemv1alpha1.Kepler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      KeplerOperatorName,
			Namespace: KeplerOperatorNameSpace,
		},
		Spec: keplersystemv1alpha1.KeplerSpec{
			Collector: &keplersystemv1alpha1.CollectorSpec{
				Image: "quay.io/sustainable_computing_io/kepler:latest",
			},
		},
	}

	keplerobjs := []runtime.Object{keplerInstance}

	s := scheme.Scheme
	s.AddKnownTypes(keplersystemv1alpha1.SchemeBuilder.GroupVersion, keplerInstance)

	clientBuilder := fake.NewClientBuilder()
	clientBuilder = clientBuilder.WithRuntimeObjects(keplerobjs...)
	clientBuilder = clientBuilder.WithScheme(s)
	cl := clientBuilder.Build()
	monitoring.AddToScheme(s)
	r := &KeplerReconciler{Client: cl, Scheme: s, Log: logger}
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      KeplerOperatorName,
			Namespace: KeplerOperatorNameSpace,
		},
	}

	res, err := r.Reconcile(ctx, req)
	//continue reconcoiling until requeue has been terminated accordingly
	for timeout := time.After(30 * time.Second); res.Requeue; {
		select {
		case <-timeout:
			t.Fatalf("main reconciler never terminates")
		default:
		}
		res, err = r.Reconcile(ctx, req)
	}
	//once reconciling has terminated accordingly, perform expected tests
	if err != nil {
		t.Fatalf("reconcile: (%v)", err)
	}
	//testVerifyMainReconciler(t, ctx, cl)

}

func TestEnsureDaemon(t *testing.T) {
	ctx, keplerReconciler, keplerInstance, logger, client := generateDefaultOperatorSettings()
	r := collectorReconciler{
		KeplerReconciler: *keplerReconciler,
		Instance:         keplerInstance,
		Ctx:              ctx,
	}
	r.serviceAccount = &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      KeplerOperatorName,
			Namespace: KeplerOperatorNameSpace,
		},
	}

	res, err := r.ensureDaemonSet(logger)
	//basic check
	assert.True(t, res)
	if err != nil {
		t.Fatal("daemonset has failed which should not happen")
	}
	foundDaemonSet := &appsv1.DaemonSet{}
	daemonSetError := client.Get(ctx, types.NamespacedName{Name: DaemonSetName, Namespace: DaemonSetNameSpace}, foundDaemonSet)

	if daemonSetError != nil {
		t.Fatal("daemonset has not been stored")
	}

	testVerifyDaemonSpec(t, *r.serviceAccount, *foundDaemonSet)

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
	assert.True(t, res)
	if err != nil {
		t.Fatal("daemonset has failed which should not happen")
	}
	foundDaemonSet = &appsv1.DaemonSet{}
	daemonSetError = client.Get(ctx, types.NamespacedName{Name: DaemonSetName, Namespace: KeplerOperatorNameSpace}, foundDaemonSet)
	if daemonSetError != nil {
		t.Fatal("daemonset has not been stored")
	}

	testVerifyDaemonSpec(t, *r.serviceAccount, *foundDaemonSet)

}

/*
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
*/
func TestEnsureService(t *testing.T) {
	ctx, keplerReconciler, keplerInstance, logger, client := generateDefaultOperatorSettings()

	numOfReconciliations := 3

	r := collectorReconciler{
		KeplerReconciler: *keplerReconciler,
		Instance:         keplerInstance,
		Ctx:              ctx,
	}

	for i := 0; i < numOfReconciliations; i++ {
		res, err := r.ensureService(logger)
		//basic check
		assert.True(t, res)
		if err != nil {
			t.Fatal("service has failed which should not happen")
		}
		foundService := &corev1.Service{}
		serviceError := client.Get(ctx, types.NamespacedName{Name: ServiceName, Namespace: ServiceNameSpace}, foundService)

		if serviceError != nil {
			t.Fatal("service has not been stored")
		}

		testVerifyServiceSpec(t, *foundService)

	}
}

//TODO: Test ServiceMonitor if necessary

// Test CollectorReconciler As a Whole

func TestCollectorReconciler(t *testing.T) {
	ctx, keplerReconciler, keplerInstance, logger, client := generateDefaultOperatorSettings()
	numOfReconciliations := 3
	for i := 0; i < numOfReconciliations; i++ {
		_, err := CollectorReconciler(ctx, keplerInstance, keplerReconciler, logger)
		if err != nil {
			t.Fatal("CollectorReconciler has failed")
		}
		//Run testVerifyCollectorReconciler
		testVerifyCollectorReconciler(t, ctx, client)

	}
}

func TestConfigMap(t *testing.T) {
	ctx, keplerReconciler, keplerInstance, logger, client := generateDefaultOperatorSettings()

	numOfReconciliations := 3

	r := collectorReconciler{
		KeplerReconciler: *keplerReconciler,
		Instance:         keplerInstance,
		Ctx:              ctx,
	}

	for i := 0; i < numOfReconciliations; i++ {
		res, err := r.ensureConfigMap(logger)
		//basic check
		assert.True(t, res)
		if err != nil {
			t.Fatal("configmap has failed which should not happen")
		}
		foundConfigMap := &corev1.ConfigMap{}
		configMapError := client.Get(ctx, types.NamespacedName{Name: CollectorConfigMapName, Namespace: CollectorConfigMapNameSpace}, foundConfigMap)

		if configMapError != nil {
			t.Fatal("configmap has not been stored")
		}

		testVerifyConfigMap(t, *foundConfigMap)

	}

}
