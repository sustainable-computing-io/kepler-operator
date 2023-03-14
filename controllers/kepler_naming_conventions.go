package controllers

// List of Suffix Names for generated Kepler Kubernetes Objects.
// Suffix Names may be a concatenation of Kepler Instance and Suffix Name or Suffix Name only.

const (
	DaemonSetNameSuffix                               = "-exporter"
	ServiceNameSuffix                                 = "-exporter"
	DaemonSetNameSpaceSuffix                          = ""
	ServiceNameSpaceSuffix                            = ""
	ServiceAccountNameSuffix                          = "-sa"
	ServiceAccountNameSpaceSuffix                     = ""
	ClusterRoleNameSuffix                             = "kepler-clusterrole"
	ClusterRoleNameSpaceSuffix                        = ""
	ClusterRoleBindingNameSuffix                      = "kepler-clusterrole-binding"
	ClusterRoleBindingNameSpaceSuffix                 = ""
	CollectorConfigMapNameSuffix                      = "-exporter-cfm"
	CollectorConfigMapNameSpaceSuffix                 = ""
	SCCObjectNameSuffix                               = "kepler-scc"
	SCCObjectNameSpaceSuffix                          = ""
	MachineConfigCGroupKernelArgMasterNameSuffix      = "50-master-cgroupv2"
	MachineConfigCGroupKernelArgWorkerNameSuffix      = "50-worker-cgroupv2"
	MachineConfigDevelMasterNameSuffix                = "51-master-kernel-devel"
	MachineConfigDevelWorkerNameSuffix                = "51-worker-kernel-devel"
	MachineConfigCGroupKernelArgMasterNameSpaceSuffix = ""
	MachineConfigCGroupKernelArgWorkerNameSpaceSuffix = ""
	MachineConfigDevelMasterNameSpaceSuffix           = ""
	MachineConfigDevelWorkerNameSpaceSuffix           = ""
	ModelServerPersistentVolumeNameSuffix             = "kepler-model-server-pv"
	ModelServerPersistentVolumeClaimNameSuffix        = "kepler-model-server-pvc"
	ModelServerConfigMapNameSuffix                    = "kepler-model-server-cfm"
	ModelServerServiceNameSuffix                      = "kepler-model-server"
	ModelServerDeploymentNameSuffix                   = "kepler-model-server"
)
