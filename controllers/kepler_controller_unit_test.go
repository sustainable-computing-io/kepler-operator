package controllers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"testing"

	"github.com/go-logr/logr"
	securityv1 "github.com/openshift/api/security/v1"
	mcfgv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	monitoring "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/stretchr/testify/assert"
	keplersystemv1alpha1 "github.com/sustainable.computing.io/kepler-operator/api/v1alpha1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	DaemonSetName                          = KeplerOperatorName + DaemonSetNameSuffix
	ServiceName                            = KeplerOperatorName + ServiceNameSuffix
	KeplerOperatorName                     = "kepler" //Sample name
	KeplerOperatorNameSpace                = ""       // Set to default
	ServiceAccountName                     = KeplerOperatorName + ServiceAccountNameSuffix
	ServiceAccountNameSpace                = KeplerOperatorNameSpace + ServiceAccountNameSpaceSuffix
	ClusterRoleName                        = ClusterRoleNameSuffix
	ClusterRoleNameSpace                   = ClusterRoleNameSpaceSuffix
	ClusterRoleBindingName                 = ClusterRoleBindingNameSuffix
	ClusterRoleBindingNameSpace            = ClusterRoleBindingNameSpaceSuffix
	DaemonSetNameSpace                     = KeplerOperatorNameSpace + DaemonSetNameSpaceSuffix
	ServiceNameSpace                       = KeplerOperatorNameSpace + ServiceNameSpaceSuffix
	CollectorConfigMapName                 = KeplerOperatorName + CollectorConfigMapNameSuffix
	CollectorConfigMapNameSpace            = KeplerOperatorNameSpace + CollectorConfigMapNameSpaceSuffix
	SCCObjectName                          = SCCObjectNameSuffix
	SCCObjectNameSpace                     = SCCObjectNameSpaceSuffix
	MachineConfigCGroupKernelArgMasterName = MachineConfigCGroupKernelArgMasterNameSuffix
	MachineConfigCGroupKernelArgWorkerName = MachineConfigCGroupKernelArgWorkerNameSuffix
	MachineConfigDevelMasterName           = MachineConfigDevelMasterNameSuffix
	MachineConfigDevelWorkerName           = MachineConfigDevelWorkerNameSuffix
	ModelServerDeploymentName              = ModelServerDeploymentNameSuffix
	ModelServerDeploymentNameSpace         = KeplerOperatorNameSpace
	ModelServerServiceName                 = ModelServerServiceNameSuffix
	ModelServerServiceNameSpace            = KeplerOperatorNameSpace
	ModelServerConfigMapName               = ModelServerConfigMapNameSuffix
	ModelServerConfigMapNameSpace          = KeplerOperatorNameSpace
	ModelServerPVName                      = ModelServerPersistentVolumeNameSuffix
	ModelServerPVNameSpace                 = KeplerOperatorNameSpace
	ModelServerPVClaimName                 = ModelServerPersistentVolumeClaimNameSuffix
	ModelServerPVClaimNameSpace            = KeplerOperatorNameSpace
)

func generateDefaultOperatorSettings() (context.Context, *KeplerReconciler, *keplersystemv1alpha1.Kepler, logr.Logger, client.Client, error) {
	ctx := context.Background()
	_ = log.FromContext(ctx)
	logger := log.Log.WithValues("kepler", types.NamespacedName{Name: KeplerOperatorName, Namespace: KeplerOperatorNameSpace})

	keplerInstance := &keplersystemv1alpha1.Kepler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      KeplerOperatorName,
			Namespace: KeplerOperatorNameSpace,
		},
		Spec: keplersystemv1alpha1.KeplerSpec{
			Collector: &keplersystemv1alpha1.CollectorSpec{
				Image:         "quay.io/sustainable_computing_io/kepler:latest",
				CollectorPort: 9103,
			},
			ModelServerExporter: &keplersystemv1alpha1.ModelServerExporterSpec{},
			ModelServerTrainer:  &keplersystemv1alpha1.ModelServerTrainerSpec{},
		},
	}

	keplerobjs := []runtime.Object{keplerInstance}

	s := scheme.Scheme
	s.AddKnownTypes(keplersystemv1alpha1.SchemeBuilder.GroupVersion, keplerInstance)

	clientBuilder := fake.NewClientBuilder()
	clientBuilder = clientBuilder.WithRuntimeObjects(keplerobjs...)
	clientBuilder = clientBuilder.WithScheme(s)
	cl := clientBuilder.Build()
	err := monitoring.AddToScheme(s)
	if err != nil {
		logger.V(1).Error(err, "failed to add prometheus operator to scheme")
		return ctx, nil, keplerInstance, logger, cl, err
	}
	err = mcfgv1.AddToScheme(s)
	if err != nil {
		logger.V(1).Error(err, "failed to add machineconfig operator to scheme")
		return ctx, nil, keplerInstance, logger, cl, err
	}
	err = securityv1.AddToScheme(s)
	if err != nil {
		logger.V(1).Error(err, "failed to add openshift security to scheme")
		return ctx, nil, keplerInstance, logger, cl, err
	}
	keplerReconciler := &KeplerReconciler{Client: cl, Scheme: s, Log: logger}

	return ctx, keplerReconciler, keplerInstance, logger, cl, nil
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

func testVerifyMainReconciler(t *testing.T, ctx context.Context, client client.Client) {
	//Check Kepler Instance has been updated as desired
	foundKepler := &keplersystemv1alpha1.Kepler{}
	foundKeplerError := client.Get(ctx, types.NamespacedName{Name: KeplerOperatorName, Namespace: KeplerOperatorNameSpace}, foundKepler)
	if foundKeplerError != nil {
		t.Fatalf("Kepler Instance was not created: (%v)", foundKeplerError)
	}
	assert.Equal(t, keplersystemv1alpha1.ConditionReconciled, foundKepler.Status.Conditions.Type)
	assert.Equal(t, "Reconcile complete", foundKepler.Status.Conditions.Message)
	assert.Equal(t, keplersystemv1alpha1.ReconciledReasonComplete, foundKepler.Status.Conditions.Reason)

	//Verify Sub-Reconcilers
	testVerifyCollectorReconciler(t, ctx, client)
	testVerifyModelServerReconciler(t, ctx, client, foundKepler)
}

func testVerifyCollectorReconciler(t *testing.T, ctx context.Context, client client.Client) {
	//Verify mock client objects exist
	foundServiceAccount := &corev1.ServiceAccount{}
	foundClusterRole := &rbacv1.ClusterRole{}
	foundClusterRoleBinding := &rbacv1.ClusterRoleBinding{}
	serviceAccountError := client.Get(ctx, types.NamespacedName{Name: ServiceAccountName, Namespace: ServiceAccountNameSpace}, foundServiceAccount)
	if serviceAccountError != nil {
		t.Fatalf("service account was not stored: (%v)", serviceAccountError)
	}
	clusterRoleError := client.Get(ctx, types.NamespacedName{Name: ClusterRoleName, Namespace: ClusterRoleNameSpace}, foundClusterRole)
	if clusterRoleError != nil {
		t.Fatalf("cluster role was not stored: (%v)", clusterRoleError)
	}
	clusterRoleBindingError := client.Get(ctx, types.NamespacedName{Name: ClusterRoleBindingName, Namespace: ClusterRoleBindingNameSpace}, foundClusterRoleBinding)
	if clusterRoleBindingError != nil {
		t.Fatalf("cluster role binding was not stored: (%v)", clusterRoleBindingError)
	}

	foundService := &corev1.Service{}
	serviceError := client.Get(ctx, types.NamespacedName{Name: ServiceName, Namespace: KeplerOperatorNameSpace}, foundService)

	if serviceError != nil {
		t.Fatalf("service was not stored: (%v)", serviceError)
	}
	foundDaemonSet := &appsv1.DaemonSet{}
	daemonSetError := client.Get(ctx, types.NamespacedName{Name: DaemonSetName, Namespace: KeplerOperatorNameSpace}, foundDaemonSet)
	if daemonSetError != nil {
		t.Fatalf("daemon Object was not stored: (%v)", daemonSetError)
	}

	foundConfigMap := &corev1.ConfigMap{}
	configMapError := client.Get(ctx, types.NamespacedName{Name: CollectorConfigMapName, Namespace: CollectorConfigMapNameSpace}, foundConfigMap)
	if configMapError != nil {
		t.Fatalf("config map was not stored: (%v)", configMapError)
	}
	foundSCC := &securityv1.SecurityContextConstraints{}
	sccError := client.Get(ctx, types.NamespacedName{Name: SCCObjectName, Namespace: SCCObjectNameSpace}, foundSCC)
	if sccError != nil {
		if strings.Contains(sccError.Error(), "no matches for kind") {
			fmt.Printf("resulting error not a timeout: %s", sccError)
		} else {
			t.Fatalf("scc was not stored: (%v)", sccError)
		}
	} else {
		testVerifySCC(t, *foundSCC)
	}

	foundMasterCgroupKernelArgs := &mcfgv1.MachineConfig{}
	foundWorkerCgroupKernelArgs := &mcfgv1.MachineConfig{}
	foundMasterDevel := &mcfgv1.MachineConfig{}
	foundWorkerDevel := &mcfgv1.MachineConfig{}
	masterCgroupKernelArgsError := client.Get(ctx, types.NamespacedName{Name: MachineConfigCGroupKernelArgMasterName, Namespace: ""}, foundMasterCgroupKernelArgs)
	workerCgroupKernelArgsError := client.Get(ctx, types.NamespacedName{Name: MachineConfigCGroupKernelArgWorkerName, Namespace: ""}, foundWorkerCgroupKernelArgs)
	masterDevelError := client.Get(ctx, types.NamespacedName{Name: MachineConfigDevelMasterName, Namespace: ""}, foundMasterDevel)
	workerDevelError := client.Get(ctx, types.NamespacedName{Name: MachineConfigDevelWorkerName, Namespace: ""}, foundWorkerDevel)

	if masterCgroupKernelArgsError != nil && strings.Contains(masterCgroupKernelArgsError.Error(), "no matches for kind") {
		fmt.Printf("resulting error not a timeout: %s", masterCgroupKernelArgsError)
	} else {
		if masterCgroupKernelArgsError != nil {
			t.Fatalf("cgroup kernel arguments master machine config has not been stored: (%v)", masterCgroupKernelArgsError)
		}
		if workerCgroupKernelArgsError != nil {
			t.Fatalf("cgroup kernel arguments worker machine config has not been stored: (%v)", workerCgroupKernelArgsError)
		}

		if masterDevelError != nil {
			t.Fatalf("devel master machine config has not been stored: (%v)", masterDevelError)
		}

		if workerDevelError != nil {
			t.Fatalf("devel worker machine config has not been stored: (%v)", workerDevelError)
		}

		testVerifyBasicMachineConfig(t, *foundMasterCgroupKernelArgs, *foundWorkerCgroupKernelArgs, *foundMasterDevel, *foundWorkerDevel)
	}

	testVerifyServiceAccountSpec(t, *foundServiceAccount, *foundClusterRole, *foundClusterRoleBinding)
	testVerifyServiceSpec(t, *foundService)
	//Note testVerifyDaemonSpec already ensures SA is assigned to Daemonset
	testVerifyDaemonSpec(t, *foundServiceAccount, *foundDaemonSet)
	testVerifyConfigMap(t, *foundConfigMap)

	//Verify Collector related cross object relationships are valid

	//Verify ServiceAccount Specified in DaemonSet
	assert.Equal(t, foundServiceAccount.Name, foundDaemonSet.Spec.Template.Spec.ServiceAccountName)

	//Verify Service selector matches daemonset spec template labels
	//Service Selector must exist correctly to connect to daemonset
	// Service Selector or SCC Labels is subset of MatchLabels and Labels in Daemonset (DaemonSet MatchLbels and Labels should be superset)
	for key, value := range foundService.Spec.Selector {
		assert.Contains(t, foundDaemonSet.Spec.Template.ObjectMeta.Labels, key)
		assert.Equal(t, value, foundDaemonSet.Spec.Template.ObjectMeta.Labels[key])
	}
	//Verify SCC Labels and Daemonset correspond
	for key, value := range foundSCC.ObjectMeta.Labels {
		assert.Contains(t, foundDaemonSet.Spec.Template.ObjectMeta.Labels, key)
		assert.Equal(t, value, foundDaemonSet.Spec.Template.ObjectMeta.Labels[key])
	}
	//Verify SCC User includes Kepler
	assert.Contains(t, foundSCC.Users, KeplerOperatorName)
	//Verify SCC User includes Kepler's Service Account
	for _, user := range foundSCC.Users {
		if strings.Contains(user, "system:serviceaccount:") {
			assert.Equal(t, "system:serviceaccount:"+ServiceAccountNameSpace+":"+ServiceAccountName, user)
		}
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

func testVerifySCC(t *testing.T, returnedSCC securityv1.SecurityContextConstraints) {
	// ensure some basic, desired settings are in place
	assert.NotEmpty(t, returnedSCC.ObjectMeta.Labels)
	assert.True(t, returnedSCC.AllowPrivilegedContainer)
	assert.Equal(t, securityv1.FSGroupStrategyOptions{
		Type: securityv1.FSGroupStrategyRunAsAny,
	}, returnedSCC.FSGroup)
	assert.Equal(t, securityv1.SELinuxContextStrategyOptions{
		Type: securityv1.SELinuxStrategyRunAsAny,
	},
		returnedSCC.SELinuxContext)
	assert.Equal(t, securityv1.RunAsUserStrategyOptions{
		Type: securityv1.RunAsUserStrategyRunAsAny,
	},
		returnedSCC.RunAsUser)

}

func testVerifyBasicMachineConfig(t *testing.T, cgroupMasterMC mcfgv1.MachineConfig, cgroupWorkerMC mcfgv1.MachineConfig, develMasterMC mcfgv1.MachineConfig, develWorkerMC mcfgv1.MachineConfig) {
	// check if all relevant Machine Config Features have been deployed correctly

	assert.NotEmpty(t, cgroupMasterMC.Labels)
	assert.Contains(t, cgroupMasterMC.Labels, "machineconfiguration.openshift.io/role")
	assert.Equal(t, "master", cgroupMasterMC.Labels["machineconfiguration.openshift.io/role"])

	assert.NotEmpty(t, cgroupWorkerMC.Labels)
	assert.Contains(t, cgroupWorkerMC.Labels, "machineconfiguration.openshift.io/role")
	assert.Equal(t, "worker", cgroupWorkerMC.Labels["machineconfiguration.openshift.io/role"])

	assert.NotEmpty(t, develMasterMC.Labels)
	assert.Contains(t, develMasterMC.Labels, "machineconfiguration.openshift.io/role")
	assert.Equal(t, "master", develMasterMC.Labels["machineconfiguration.openshift.io/role"])

	assert.NotEmpty(t, develWorkerMC.Labels)
	assert.Contains(t, develWorkerMC.Labels, "machineconfiguration.openshift.io/role")
	assert.Equal(t, "worker", develWorkerMC.Labels["machineconfiguration.openshift.io/role"])

	// check if all relevant Machine Config Objects have correct spec
	assert.NotEmpty(t, develMasterMC.Spec)
	assert.NotEmpty(t, develWorkerMC.Spec)
	assert.NotEmpty(t, cgroupMasterMC.Spec)
	assert.NotEmpty(t, cgroupWorkerMC.Spec)

	assert.NotEmpty(t, develMasterMC.Spec.Extensions)
	assert.Contains(t, develMasterMC.Spec.Extensions, "kernel-devel")

	assert.NotEmpty(t, develWorkerMC.Spec.Extensions)
	assert.Contains(t, develWorkerMC.Spec.Extensions, "kernel-devel")

	assert.NotEmpty(t, cgroupMasterMC.Spec.KernelArguments)
	assert.Contains(t, cgroupMasterMC.Spec.KernelArguments, "systemd.unified_cgroup_hierarchy=1")
	assert.Contains(t, cgroupMasterMC.Spec.KernelArguments, "cgroup_no_v1='all'")

	assert.NotEmpty(t, cgroupWorkerMC.Spec.KernelArguments)
	assert.Contains(t, cgroupWorkerMC.Spec.KernelArguments, "systemd.unified_cgroup_hierarchy=1")
	assert.Contains(t, cgroupWorkerMC.Spec.KernelArguments, "cgroup_no_v1='all'")
}

func testVerifyConfigMap(t *testing.T, returnedConfigMap corev1.ConfigMap) {
	// check SetControllerReference has been set (all objects require owners) properly
	result := CheckSetControllerReference(KeplerOperatorName, "Kepler", &returnedConfigMap)
	if !result {
		t.Fatalf("failed to set controller reference: config map")
	}
	//check if ConfigMap contains proper datamap
	assert.NotEmpty(t, returnedConfigMap.Data)
	assert.Contains(t, returnedConfigMap.Data, "KEPLER_NAMESPACE")
	assert.Contains(t, returnedConfigMap.Data, "METRIC_PATH")
	assert.Equal(t, KeplerOperatorNameSpace, returnedConfigMap.Data["KEPLER_NAMESPACE"])
	assert.Equal(t, "/metrics", returnedConfigMap.Data["METRIC_PATH"])
}

func testVerifyServiceSpec(t *testing.T, returnedService corev1.Service) {
	// check SetControllerReference has been set (all objects require owners) properly
	result := CheckSetControllerReference(KeplerOperatorName, "Kepler", &returnedService)
	if !result {
		t.Fatalf("failed to set controller reference: service")
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

func testVerifyServiceAccountSpec(t *testing.T, returnedServiceAccount corev1.ServiceAccount, returnedClusterRole rbacv1.ClusterRole, returnedClusterRoleBinding rbacv1.ClusterRoleBinding) {
	// check SetControllerReference has been set (all objects require owners) properly for SA, Role, RoleBinding
	result := CheckSetControllerReference(KeplerOperatorName, "Kepler", &returnedServiceAccount)
	if !result {
		t.Fatalf("failed to set controller reference: service account")
	}

	//assert.Equal(t, 1, len(returnedServiceAccount.GetOwnerReferences()))
	//assert.Equal(t, 1, len(returnedRole.GetOwnerReferences()))
	//assert.Equal(t, 1, len(returnedRoleBinding.GetOwnerReferences()))
	//assert.Equal(t, KeplerOperatorName, returnedServiceAccount.GetOwnerReferences()[0].Name)
	//assert.Equal(t, KeplerOperatorName, returnedRole.GetOwnerReferences()[0].Name)
	//assert.Equal(t, KeplerOperatorName, returnedRoleBinding.GetOwnerReferences()[0].Name)

	//assert.Equal(t, "Kepler", returnedServiceAccount.GetOwnerReferences()[0].Kind)
	//assert.Equal(t, "Kepler", returnedRole.GetOwnerReferences()[0].Kind)
	//assert.Equal(t, "Kepler", returnedRoleBinding.GetOwnerReferences()[0].Kind)

	//check if CreateOrUpdate Object has properly set up required fields, nested fields, and variable fields for SA
	assert.NotEmpty(t, returnedServiceAccount.ObjectMeta)
	assert.Equal(t, ServiceAccountName, returnedServiceAccount.ObjectMeta.Name)
	assert.Equal(t, ServiceAccountNameSpace, returnedServiceAccount.ObjectMeta.Namespace)

	//check if CreateOrUpdate Object has properly set up required fields, nested fields, and variable fields for ClusterRole
	assert.NotEmpty(t, returnedClusterRole.ObjectMeta)
	assert.Equal(t, ClusterRoleName, returnedClusterRole.ObjectMeta.Name)
	assert.Equal(t, ClusterRoleNameSpace, returnedClusterRole.ObjectMeta.Namespace)
	assert.NotEmpty(t, returnedClusterRole.Rules)

	//check if CreateOrUpdate Object has properly set up required fields, nested fields, and variable fields for ClusterRoleBinding
	assert.NotEmpty(t, returnedClusterRoleBinding.ObjectMeta)
	assert.Equal(t, ClusterRoleBindingName, returnedClusterRoleBinding.ObjectMeta.Name)
	assert.Equal(t, ClusterRoleBindingNameSpace, returnedClusterRoleBinding.ObjectMeta.Namespace)
	assert.NotEmpty(t, returnedClusterRoleBinding.RoleRef)
	assert.NotEmpty(t, returnedClusterRoleBinding.Subjects)
	assert.Equal(t, returnedServiceAccount.Kind, returnedClusterRoleBinding.Subjects[0].Kind)
	assert.Equal(t, returnedServiceAccount.Name, returnedClusterRoleBinding.Subjects[0].Name)
	assert.Equal(t, returnedServiceAccount.Namespace, returnedClusterRoleBinding.Subjects[0].Namespace)
	assert.Equal(t, "rbac.authorization.k8s.io", returnedClusterRoleBinding.RoleRef.APIGroup)
	assert.Equal(t, returnedClusterRole.Kind, returnedClusterRoleBinding.RoleRef.Kind)
	assert.Equal(t, returnedClusterRole.Name, returnedClusterRoleBinding.RoleRef.Name)

}

func testVerifyDaemonSpec(t *testing.T, returnedServiceAccount corev1.ServiceAccount, returnedDaemonSet appsv1.DaemonSet) {
	// check SetControllerReference has been set (all objects require owners) properly
	result := CheckSetControllerReference(KeplerOperatorName, "Kepler", &returnedDaemonSet)
	if !result {
		t.Fatalf("failed to set controller reference: daemonset")
	}
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
		//check security
		if container.Name == "kepler-exporter" {
			assert.True(t, *container.SecurityContext.Privileged)
			assert.Equal(t, "quay.io/sustainable_computing_io/kepler:latest", container.Image)
		}

	}

	assert.Equal(t, DaemonSetName, returnedDaemonSet.ObjectMeta.Name)
	assert.Equal(t, DaemonSetNameSpace, returnedDaemonSet.ObjectMeta.Namespace)

	assert.Equal(t, DaemonSetName, returnedDaemonSet.Spec.Template.ObjectMeta.Name)
	assert.Equal(t, returnedServiceAccount.Name, returnedDaemonSet.Spec.Template.Spec.ServiceAccountName)
	// check if daemonset obeys general rules
	//TODO: MATCH LABELS IS subset to labels. SAME WITH SELECTOR IN SERVICE
	// NEED TO MAKE SURE RELATED SERVICE CONNECTS TO EXISTING LABELS IN DAEMONSET PODS TOO
	//assert.Equal(t, returnedDaemonSet.Spec.Selector.MatchLabels, returnedDaemonSet.Spec.Template.ObjectMeta.Labels)
	assert.NotEmpty(t, returnedDaemonSet.Spec.Selector.MatchLabels)
	assert.NotEmpty(t, returnedDaemonSet.Spec.Template.ObjectMeta.Labels)
	for key, value := range returnedDaemonSet.Spec.Selector.MatchLabels {
		assert.Contains(t, returnedDaemonSet.Spec.Template.ObjectMeta.Labels, key)
		assert.Equal(t, value, returnedDaemonSet.Spec.Template.ObjectMeta.Labels[key])
	}

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
	//TODO: note that volumes that are not mounted is not allowed (is this true). Is this worth addressing?
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

func testVerifyModelServerDeployment(t *testing.T, returnedMSDeployment appsv1.Deployment) {
	// check SetControllerReference has been set (all objects require owners) properly
	result := CheckSetControllerReference(KeplerOperatorName, "Kepler", &returnedMSDeployment)
	if !result {
		t.Fatalf("failed to set controller reference: model server deployment")
	}
	// check if CreateOrUpdate Object has properly set up required fields, nested fields, and variable fields
	assert.NotEmpty(t, returnedMSDeployment.Spec)
	assert.NotEmpty(t, returnedMSDeployment.Spec.Template)
	assert.NotEmpty(t, returnedMSDeployment.Spec.Template.Spec)
	assert.NotEmpty(t, returnedMSDeployment.ObjectMeta)
	assert.NotEmpty(t, returnedMSDeployment.Spec.Template.ObjectMeta)

	assert.NotEqual(t, 0, len(returnedMSDeployment.Spec.Template.Spec.Containers))

	for _, container := range returnedMSDeployment.Spec.Template.Spec.Containers {
		assert.NotEmpty(t, container.Image)
		assert.NotEmpty(t, container.Name)
		//check security
		if container.Name == "model-server-api" {
			assert.Equal(t, ModelServerContainerImage, container.Image)
			assert.Equal(t, container.Command, []string{"python3.8", "model_server.py"})
		} else if container.Name == "online-trainer" {
			assert.Equal(t, ModelServerContainerImage, container.Image)
			assert.Equal(t, container.Command, []string{"python3.8", "online_trainer.py"})
		}

	}

	assert.Equal(t, ModelServerDeploymentName, returnedMSDeployment.ObjectMeta.Name)
	assert.Equal(t, ModelServerDeploymentNameSpace, returnedMSDeployment.ObjectMeta.Namespace)

	// ensure matchlabels and labels match
	assert.NotEmpty(t, returnedMSDeployment.Spec.Selector.MatchLabels)
	assert.NotEmpty(t, returnedMSDeployment.Spec.Template.ObjectMeta.Labels)
	for key, value := range returnedMSDeployment.Spec.Selector.MatchLabels {
		assert.Contains(t, returnedMSDeployment.Spec.Template.ObjectMeta.Labels, key)
		assert.Equal(t, value, returnedMSDeployment.Spec.Template.ObjectMeta.Labels[key])
	}

	// ensure volumemounts reference existing connected volumes
	volumes := returnedMSDeployment.Spec.Template.Spec.Volumes
	for _, container := range returnedMSDeployment.Spec.Template.Spec.Containers {
		encountered := false
		for _, volumeMount := range container.VolumeMounts {
			for _, volume := range volumes {
				if volumeMount.Name == volume.Name {
					encountered = true
				}
			}
		}
		assert.True(t, encountered)
	}

}

func testVerifyModelServerService(t *testing.T, returnedMSService corev1.Service) {
	// check SetControllerReference has been set (all objects require owners) properly
	result := CheckSetControllerReference(KeplerOperatorName, "Kepler", &returnedMSService)
	if !result {
		t.Fatalf("failed to set controller reference: model server service")
	}
	//check if CreateOrUpdate Object has properly set up required fields, nested fields, and variable fields for SA
	assert.NotEmpty(t, returnedMSService.ObjectMeta)
	assert.Equal(t, ModelServerServiceName, returnedMSService.ObjectMeta.Name)
	assert.Equal(t, ModelServerServiceNameSpace, returnedMSService.ObjectMeta.Namespace)
	assert.NotEmpty(t, returnedMSService.Spec)
	assert.NotEmpty(t, returnedMSService.Spec.Ports)
	assert.NotEmpty(t, returnedMSService.Spec.Selector)
	assert.Equal(t, "None", returnedMSService.Spec.ClusterIP)

}

func testVerifyModelServerPersistentVolume(t *testing.T, returnedMSPV corev1.PersistentVolume) {
	assert.NotEmpty(t, returnedMSPV.ObjectMeta)
	assert.NotEmpty(t, returnedMSPV.Spec)
}

func testVerifyModelServerPersistentVolumeClaim(t *testing.T, returnedMSPVC corev1.PersistentVolumeClaim) {
	assert.NotEmpty(t, returnedMSPVC.ObjectMeta)
	assert.NotEmpty(t, returnedMSPVC.Spec)
}

func testVerifyModelServerConfigMap(t *testing.T, returnedMSConfigMap corev1.ConfigMap, keplerInstance *keplersystemv1alpha1.Kepler) {
	// check SetControllerReference has been set (all objects require owners) properly
	result := CheckSetControllerReference(KeplerOperatorName, "Kepler", &returnedMSConfigMap)
	if !result {
		t.Fatalf("failed to set controller reference: model server config map")
	}
	//check if ConfigMap contains proper datamap
	assert.NotEmpty(t, returnedMSConfigMap.Data)
	assert.Contains(t, returnedMSConfigMap.Data, "MNT_PATH")
	assert.Equal(t, "/mnt", returnedMSConfigMap.Data["MNT_PATH"])
	if keplerInstance.Spec.ModelServerExporter != nil {
		assert.Contains(t, returnedMSConfigMap.Data, "MODEL_SERVER_ENABLE")
		assert.Equal(t, "true", returnedMSConfigMap.Data["MODEL_SERVER_ENABLE"])
		assert.Contains(t, returnedMSConfigMap.Data, "PROM_SERVER")
		assert.NotEmpty(t, returnedMSConfigMap.Data["PROM_SERVER"])
		assert.Contains(t, returnedMSConfigMap.Data, "MODEL_PATH")
		assert.NotEmpty(t, returnedMSConfigMap.Data["MODEL_PATH"])
		assert.Contains(t, returnedMSConfigMap.Data, "MODEL_SERVER_PORT")
		assert.NotEmpty(t, returnedMSConfigMap.Data["MODEL_SERVER_PORT"])
		assert.Contains(t, returnedMSConfigMap.Data, "MODEL_SERVER_URL")
		assert.NotEmpty(t, returnedMSConfigMap.Data["MODEL_SERVER_URL"])
		assert.Contains(t, returnedMSConfigMap.Data, "MODEL_SERVER_REQ_PATH")
		assert.NotEmpty(t, returnedMSConfigMap.Data["MODEL_SERVER_REQ_PATH"])
	}

	if keplerInstance.Spec.ModelServerTrainer != nil {
		assert.Contains(t, returnedMSConfigMap.Data, "PROM_QUERY_INTERVAL")
		assert.NotEmpty(t, returnedMSConfigMap.Data["PROM_QUERY_INTERVAL"])
		assert.Contains(t, returnedMSConfigMap.Data, "PROM_QUERY_STEP")
		assert.NotEmpty(t, returnedMSConfigMap.Data["PROM_QUERY_STEP"])
		assert.Contains(t, returnedMSConfigMap.Data, "PROM_SSL_DISABLE")
		assert.NotEmpty(t, returnedMSConfigMap.Data["PROM_SSL_DISABLE"])
		assert.Contains(t, returnedMSConfigMap.Data, "PROM_HEADERS")
		assert.Contains(t, returnedMSConfigMap.Data, "INITIAL_MODELS_LOC")
		assert.NotEmpty(t, returnedMSConfigMap.Data["INITIAL_MODELS_LOC"])

	}

}

func testVerifyModelServerReconciler(t *testing.T, ctx context.Context, client client.Client, keplerInstance *keplersystemv1alpha1.Kepler) {

	foundModelServerDeployment := &appsv1.Deployment{}
	modelServerDeploymentError := client.Get(ctx, types.NamespacedName{Name: ModelServerDeploymentName, Namespace: ModelServerDeploymentNameSpace}, foundModelServerDeployment)
	if modelServerDeploymentError != nil {
		t.Fatalf("model server deployment was not stored: (%v)", modelServerDeploymentError)
	}
	foundModelServerService := &corev1.Service{}
	modelServerServiceError := client.Get(ctx, types.NamespacedName{Name: ModelServerServiceName, Namespace: ModelServerServiceNameSpace}, foundModelServerService)
	if modelServerServiceError != nil {
		t.Fatalf("model server service was not stored: (%v)", modelServerServiceError)
	}
	foundModelServerConfigMap := &corev1.ConfigMap{}
	modelServerConfigMapError := client.Get(ctx, types.NamespacedName{Name: ModelServerConfigMapName, Namespace: ModelServerConfigMapNameSpace}, foundModelServerConfigMap)
	if modelServerConfigMapError != nil {
		t.Fatalf("model server configmap was not stored: (%v)", modelServerConfigMapError)
	}
	foundModelServerPV := &corev1.PersistentVolume{}
	modelServerPVError := client.Get(ctx, types.NamespacedName{Name: ModelServerPVName, Namespace: ModelServerPVNameSpace}, foundModelServerPV)
	if modelServerPVError != nil {
		t.Fatalf("model server pv was not stored: (%v)", modelServerPVError)
	}
	foundModelServerPVC := &corev1.PersistentVolumeClaim{}
	modelServerPVCError := client.Get(ctx, types.NamespacedName{Name: ModelServerPVClaimName, Namespace: ModelServerPVClaimNameSpace}, foundModelServerPVC)
	if modelServerPVCError != nil {
		t.Fatalf("model server pvc was not stored: (%v)", modelServerPVCError)
	}

	// test individual model server kubernetes objects
	testVerifyModelServerConfigMap(t, *foundModelServerConfigMap, keplerInstance)
	testVerifyModelServerDeployment(t, *foundModelServerDeployment)
	testVerifyModelServerPersistentVolume(t, *foundModelServerPV)
	testVerifyModelServerPersistentVolumeClaim(t, *foundModelServerPVC)
	testVerifyModelServerService(t, *foundModelServerService)

	// verify volume claim's volume name and volume
	assert.Equal(t, foundModelServerPV.Name, foundModelServerPVC.Spec.VolumeName)
	// verify service label and deployment label
	for key, value := range foundModelServerService.Spec.Selector {
		assert.Contains(t, foundModelServerDeployment.Spec.Template.ObjectMeta.Labels, key)
		assert.Equal(t, value, foundModelServerDeployment.Spec.Template.ObjectMeta.Labels[key])
	}

	// verify deployment contains service and configmap mounts
	//Verify ConfigMap exists in Daemonset Volumes
	encounteredConfigMapVolume := false
	encounteredPVCVolume := false
	for _, volume := range foundModelServerDeployment.Spec.Template.Spec.Volumes {
		if volume.VolumeSource.ConfigMap != nil {
			//found configmap
			if foundModelServerConfigMap.ObjectMeta.Name == volume.VolumeSource.ConfigMap.Name {
				encounteredConfigMapVolume = true
			}
		}
		if volume.VolumeSource.PersistentVolumeClaim != nil {
			//found persistentvolumeclaim
			if foundModelServerPVC.Name == volume.VolumeSource.PersistentVolumeClaim.ClaimName {
				encounteredPVCVolume = true
			}
		}
	}
	assert.True(t, encounteredConfigMapVolume)
	assert.True(t, encounteredPVCVolume)

}

func TestEnsureDaemon(t *testing.T) {
	ctx, keplerReconciler, keplerInstance, logger, client, err := generateDefaultOperatorSettings()
	if err != nil {
		t.Fatalf("generate test environment failed: (%v)", err)
	}
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
		t.Fatalf("daemonset has failed which should not happen: (%v)", err)
	}
	foundDaemonSet := &appsv1.DaemonSet{}
	daemonSetError := client.Get(ctx, types.NamespacedName{Name: DaemonSetName, Namespace: DaemonSetNameSpace}, foundDaemonSet)

	if daemonSetError != nil {
		t.Fatalf("daemonset has not been stored: (%v)", daemonSetError)
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
		t.Fatalf("daemonset has failed which should not happen: (%v)", err)
	}
	foundDaemonSet = &appsv1.DaemonSet{}
	daemonSetError = client.Get(ctx, types.NamespacedName{Name: DaemonSetName, Namespace: KeplerOperatorNameSpace}, foundDaemonSet)
	if daemonSetError != nil {
		t.Fatalf("daemonset has not been stored: (%v)", daemonSetError)
	}

	testVerifyDaemonSpec(t, *r.serviceAccount, *foundDaemonSet)

}

func TestEnsureServiceAccount(t *testing.T) {
	ctx, keplerReconciler, keplerInstance, logger, client, err := generateDefaultOperatorSettings()
	if err != nil {
		t.Fatalf("generate test environment failed: (%v)", err)
	}
	r := collectorReconciler{
		KeplerReconciler: *keplerReconciler,
		Instance:         keplerInstance,
		Ctx:              ctx,
	}
	numOfReconciliations := 3
	for i := 0; i < numOfReconciliations; i++ {
		//should also affect role and role binding
		res, err := r.ensureServiceAccount(logger)
		//basic check
		assert.True(t, res)
		if err != nil {
			t.Fatalf("service account reconciler has failed which should not happen: (%v)", err)
		}
		foundServiceAccount := &corev1.ServiceAccount{}
		serviceAccountError := client.Get(ctx, types.NamespacedName{Name: ServiceAccountName, Namespace: ServiceAccountNameSpace}, foundServiceAccount)
		foundClusterRole := &rbacv1.ClusterRole{}
		clusterRoleError := client.Get(ctx, types.NamespacedName{Name: ClusterRoleName, Namespace: ClusterRoleNameSpace}, foundClusterRole)
		foundClusterRoleBinding := &rbacv1.ClusterRoleBinding{}
		clusterRoleBindingError := client.Get(ctx, types.NamespacedName{Name: ClusterRoleBindingName, Namespace: ClusterRoleBindingNameSpace}, foundClusterRoleBinding)

		if serviceAccountError != nil {
			t.Fatalf("service account has not been stored: (%v)", serviceAccountError)
		}
		if clusterRoleError != nil {
			t.Fatalf("cluster role has not been stored: (%v)", clusterRoleError)
		}
		if clusterRoleBindingError != nil {
			t.Fatalf("cluster rolebinding has not been stored: (%v)", clusterRoleBindingError)
		}

		testVerifyServiceAccountSpec(t, *foundServiceAccount, *foundClusterRole, *foundClusterRoleBinding)

	}

}

func TestEnsureService(t *testing.T) {
	ctx, keplerReconciler, keplerInstance, logger, client, err := generateDefaultOperatorSettings()
	if err != nil {
		t.Fatalf("generate test environment failed: (%v)", err)
	}

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
			t.Fatalf("service has failed which should not happen: (%v)", err)
		}
		foundService := &corev1.Service{}
		serviceError := client.Get(ctx, types.NamespacedName{Name: ServiceName, Namespace: ServiceNameSpace}, foundService)

		if serviceError != nil {
			t.Fatalf("service has not been stored: (%v)", serviceError)
		}

		testVerifyServiceSpec(t, *foundService)

	}
}

func TestConfigMap(t *testing.T) {
	ctx, keplerReconciler, keplerInstance, logger, client, err := generateDefaultOperatorSettings()
	if err != nil {
		t.Fatalf("generate test environment failed: (%v)", err)
	}

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
			t.Fatalf("configmap has failed which should not happen: (%v)", err)
		}
		foundConfigMap := &corev1.ConfigMap{}
		configMapError := client.Get(ctx, types.NamespacedName{Name: CollectorConfigMapName, Namespace: CollectorConfigMapNameSpace}, foundConfigMap)

		if configMapError != nil {
			t.Fatalf("configmap has not been stored: (%v)", configMapError)
		}

		testVerifyConfigMap(t, *foundConfigMap)

	}

}

func TestSCC(t *testing.T) {
	ctx, keplerReconciler, keplerInstance, logger, client, err := generateDefaultOperatorSettings()
	if err != nil {
		t.Fatalf("generate test environment failed: (%v)", err)
	}

	numOfReconciliations := 3

	r := collectorReconciler{
		KeplerReconciler: *keplerReconciler,
		Instance:         keplerInstance,
		Ctx:              ctx,
	}

	for i := 0; i < numOfReconciliations; i++ {
		res, err := r.ensureSCC(logger)
		assert.True(t, res)
		if err != nil {
			t.Fatalf("scc has failed which should not happen: (%v)", err)
		}
		foundSCC := &securityv1.SecurityContextConstraints{}
		sccError := client.Get(ctx, types.NamespacedName{Name: SCCObjectName, Namespace: SCCObjectNameSpace}, foundSCC)

		if sccError != nil && strings.Contains(err.Error(), "no matches for kind") {
			fmt.Printf("resulting error not a timeout: %s", sccError)

		}
		if sccError != nil {
			t.Fatalf("scc has not been stored: (%v)", sccError)
		}
		testVerifySCC(t, *foundSCC)

	}
}

func TestBasicMachineConfig(t *testing.T) {
	ctx, keplerReconciler, keplerInstance, logger, client, err := generateDefaultOperatorSettings()
	if err != nil {
		t.Fatalf("generate test environment failed: (%v)", err)
	}

	numOfReconciliations := 3

	r := collectorReconciler{
		KeplerReconciler: *keplerReconciler,
		Instance:         keplerInstance,
		Ctx:              ctx,
	}

	for i := 0; i < numOfReconciliations; i++ {
		res, err := r.ensureMachineConfig(logger)
		assert.True(t, res)
		if err != nil {
			t.Fatalf("machineconfig has failed which should not happen: (%v)", err)
		}
		foundMasterCgroupKernelArgs := &mcfgv1.MachineConfig{}
		foundWorkerCgroupKernelArgs := &mcfgv1.MachineConfig{}
		foundMasterDevel := &mcfgv1.MachineConfig{}
		foundWorkerDevel := &mcfgv1.MachineConfig{}
		masterCgroupKernelArgsError := client.Get(ctx, types.NamespacedName{Name: MachineConfigCGroupKernelArgMasterName, Namespace: ""}, foundMasterCgroupKernelArgs)
		workerCgroupKernelArgsError := client.Get(ctx, types.NamespacedName{Name: MachineConfigCGroupKernelArgWorkerName, Namespace: ""}, foundWorkerCgroupKernelArgs)
		masterDevelError := client.Get(ctx, types.NamespacedName{Name: MachineConfigDevelMasterName, Namespace: ""}, foundMasterDevel)
		workerDevelError := client.Get(ctx, types.NamespacedName{Name: MachineConfigDevelWorkerName, Namespace: ""}, foundWorkerDevel)

		if masterCgroupKernelArgsError != nil && strings.Contains(masterCgroupKernelArgsError.Error(), "no matches for kind") {
			fmt.Printf("resulting error not a timeout: %s", masterCgroupKernelArgsError)
		} else {
			if masterCgroupKernelArgsError != nil {
				t.Fatalf("cgroup kernel arguments master machine config has not been stored: (%v)", masterCgroupKernelArgsError)
			}
			if workerCgroupKernelArgsError != nil {
				t.Fatalf("cgroup kernel arguments worker machine config has not been stored: (%v)", workerCgroupKernelArgsError)
			}

			if masterDevelError != nil {
				t.Fatalf("devel master machine config has not been stored: (%v)", masterDevelError)
			}

			if workerDevelError != nil {
				t.Fatalf("devel worker machine config has not been stored: (%v)", workerDevelError)
			}

			testVerifyBasicMachineConfig(t, *foundMasterCgroupKernelArgs, *foundWorkerCgroupKernelArgs, *foundMasterDevel, *foundWorkerDevel)
		}
	}

}

func TestModelServerConfigMap(t *testing.T) {
	ctx, keplerReconciler, keplerInstance, logger, client, err := generateDefaultOperatorSettings()
	if err != nil {
		t.Fatalf("generate test environment failed: (%v)", err)
	}

	numOfReconciliations := 3

	m := ModelServerDeployment{
		Context:  ctx,
		Instance: keplerInstance,
		Image:    ModelServerContainerImage,
		Client:   client,
		Scheme:   keplerReconciler.Scheme,
	}

	for i := 0; i < numOfReconciliations; i++ {
		res, err := m.ensureModelServerConfigMap(logger)
		//basic check
		assert.True(t, res)
		if err != nil {
			t.Fatalf("model server configmap has failed which should not happen: (%v)", err)
		}
		foundMSConfigMap := &corev1.ConfigMap{}
		msConfigMapError := client.Get(ctx, types.NamespacedName{Name: ModelServerConfigMapName, Namespace: ModelServerConfigMapNameSpace}, foundMSConfigMap)

		if msConfigMapError != nil {
			t.Fatalf("model server configmap has not been stored: (%v)", msConfigMapError)
		}

		testVerifyModelServerConfigMap(t, *foundMSConfigMap, keplerInstance)
	}
}

func TestModelServerLocalStorage(t *testing.T) {
	ctx, keplerReconciler, keplerInstance, logger, client, err := generateDefaultOperatorSettings()
	if err != nil {
		t.Fatalf("generate test environment failed: (%v)", err)
	}

	numOfReconciliations := 3

	m := ModelServerDeployment{
		Context:  ctx,
		Instance: keplerInstance,
		Image:    ModelServerContainerImage,
		Client:   client,
		Scheme:   keplerReconciler.Scheme,
	}

	for i := 0; i < numOfReconciliations; i++ {
		resPV, errPV := m.ensureModelServerPersistentVolume(logger)
		resPVC, errPVC := m.ensureModelServerPersistentVolumeClaim(logger)
		//basic check
		assert.True(t, resPV)
		if errPV != nil {
			t.Fatalf("model server pv has failed which should not happen: (%v)", errPV)
		}
		assert.True(t, resPVC)
		if errPVC != nil {
			t.Fatalf("model server pvc has failed which should not happen: (%v)", errPVC)
		}
		foundMSPV := &corev1.PersistentVolume{}
		msPVError := client.Get(ctx, types.NamespacedName{Name: ModelServerPVName, Namespace: ModelServerPVNameSpace}, foundMSPV)
		if msPVError != nil {
			t.Fatalf("model server pv has not been stored: (%v)", msPVError)
		}
		foundMSPVC := &corev1.PersistentVolumeClaim{}
		msPVCError := client.Get(ctx, types.NamespacedName{Name: ModelServerPVClaimName, Namespace: ModelServerPVClaimNameSpace}, foundMSPVC)
		if msPVCError != nil {
			t.Fatalf("model server pvc has not been stored: (%v)", msPVCError)
		}
		testVerifyModelServerPersistentVolume(t, *foundMSPV)
		testVerifyModelServerPersistentVolumeClaim(t, *foundMSPVC)
	}
}

func TestModelServerService(t *testing.T) {
	ctx, keplerReconciler, keplerInstance, logger, client, err := generateDefaultOperatorSettings()
	if err != nil {
		t.Fatalf("generate test environment failed: (%v)", err)
	}

	numOfReconciliations := 3

	m := ModelServerDeployment{
		Context:  ctx,
		Instance: keplerInstance,
		Image:    ModelServerContainerImage,
		Client:   client,
		Scheme:   keplerReconciler.Scheme,
	}

	for i := 0; i < numOfReconciliations; i++ {
		res, err := m.ensureModelServerService(logger)
		//basic check
		assert.True(t, res)
		if err != nil {
			t.Fatalf("model server service has failed which should not happen: (%v)", err)
		}
		foundMSService := &corev1.Service{}
		msServiceError := client.Get(ctx, types.NamespacedName{Name: ModelServerServiceName, Namespace: ModelServerServiceNameSpace}, foundMSService)

		if msServiceError != nil {
			t.Fatalf("model server service has not been stored: (%v)", msServiceError)
		}

		testVerifyModelServerService(t, *foundMSService)
	}
}

func TestModelServerDeployment(t *testing.T) {
	ctx, keplerReconciler, keplerInstance, logger, client, err := generateDefaultOperatorSettings()
	if err != nil {
		t.Fatalf("generate test environment failed: (%v)", err)
	}

	m := ModelServerDeployment{
		Context:  ctx,
		Instance: keplerInstance,
		Image:    ModelServerContainerImage,
		Client:   client,
		Scheme:   keplerReconciler.Scheme,
		PersistentVolumeClaim: &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ModelServerPVClaimName,
				Namespace: ModelServerPVClaimNameSpace,
			},
		},
		ConfigMap: &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ModelServerConfigMapName,
				Namespace: ModelServerConfigMapNameSpace,
			},
		},
	}

	res, err := m.ensureModelServerDeployment(logger)
	//basic check
	assert.True(t, res)
	if err != nil {
		t.Fatalf("model server deployment has failed which should not happen: (%v)", err)
	}
	foundMSDeployment := &appsv1.Deployment{}
	msDeploymentError := client.Get(ctx, types.NamespacedName{Name: ModelServerDeploymentName, Namespace: ModelServerDeploymentNameSpace}, foundMSDeployment)

	if msDeploymentError != nil {
		t.Fatalf("model server deployment has not been stored: (%v)", msDeploymentError)
	}

	testVerifyModelServerDeployment(t, *foundMSDeployment)

}

// Test CollectorReconciler As a Whole

func TestCollectorReconciler(t *testing.T) {
	ctx, keplerReconciler, keplerInstance, logger, client, err := generateDefaultOperatorSettings()
	if err != nil {
		t.Fatalf("generate test environment failed: (%v)", err)
	}
	numOfReconciliations := 3
	for i := 0; i < numOfReconciliations; i++ {
		_, err := CollectorReconciler(ctx, keplerInstance, keplerReconciler, logger)
		if err != nil {
			// This will never occur because such errors are handled already
			/*if strings.Contains(err.Error(), "no matches for kind") {
				if strings.Contains(err.Error(), "SecurityContextConstraints") || strings.Contains(err.Error(), "MachineConfig") {
					logger.V(1).Info("Not OpenShift skip SecurityContextConstraints and MachineConfig")
					continue
				}
			} else {*/
			t.Fatalf("collector reconciler has failed: (%v)", err)

		}
		//Run testVerifyCollectorReconciler
		testVerifyCollectorReconciler(t, ctx, client)

	}
}

// Test ModelServerReconciler as a whole

func TestModelServerReconciler(t *testing.T) {
	ctx, keplerReconciler, keplerInstance, logger, client, err := generateDefaultOperatorSettings()
	if err != nil {
		t.Fatalf("generate test environment failed: (%v)", err)
	}
	numOfReconciliations := 3
	for i := 0; i < numOfReconciliations; i++ {
		_, err = ModelServerReconciler(ctx, keplerInstance, keplerReconciler, logger)
		if err != nil {
			t.Fatalf("model server reconciler has failed: (%v)", err)

		}
		//Run testVerifyModelServerReconciler
		testVerifyModelServerReconciler(t, ctx, client, keplerInstance)

	}

}

// Test KeplerOperator as a whole

func TestEnsureKeplerOperator(t *testing.T) {
	ctx, keplerReconciler, _, _, client, err := generateDefaultOperatorSettings()
	if err != nil {
		t.Fatalf("generate test environment failed: (%v)", err)
	}
	r := keplerReconciler
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      KeplerOperatorName,
			Namespace: KeplerOperatorNameSpace,
		},
	}
	//should only call reconcile once (Additional reconciliations will be called if requeing is required)
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
	testVerifyMainReconciler(t, ctx, client)

}
