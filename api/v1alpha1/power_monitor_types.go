// SPDX-FileCopyrightText: 2025 The Kepler Authors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/sustainable.computing.io/kepler-operator/internal/config"
)

const (
	// InvalidPowerMonitorResource indicates the CR name was invalid
	InvalidPowerMonitorResource ConditionReason = "InvalidPowerMonitorResource"
)

// SecurityMode defines the security mode for Kepler metrics access
type SecurityMode string

const (
	// SecurityModeNone disables RBAC-based access control for Kepler metrics
	SecurityModeNone SecurityMode = "none"
	// SecurityModeRBAC enables RBAC-based access control for Kepler metrics
	SecurityModeRBAC SecurityMode = "rbac"
)

// PowerMonitorKeplerDeploymentSecuritySpec defines security settings for the Kepler deployment
type PowerMonitorKeplerDeploymentSecuritySpec struct {
	// Mode specifies the security mode (none or rbac)
	// +kubebuilder:validation:Enum=none;rbac
	Mode SecurityMode `json:"mode,omitempty"`
	// AllowedSANames lists service account names allowed to access Kepler metrics
	// +optional
	// +listType=atomic
	AllowedSANames []string `json:"allowedSANames,omitempty"`
}

// MetricsLevelDefault represents the default metric levels for PowerMonitor (node, pod, and vm)
const MetricsLevelDefault = config.MetricsLevelNode | config.MetricsLevelPod | config.MetricsLevelVM

// PowerMonitorKeplerDeploymentSpec defines deployment settings for the Kepler DaemonSet
type PowerMonitorKeplerDeploymentSpec struct {
	// NodeSelector defines which Nodes the Pod is scheduled on
	// +optional
	// +kubebuilder:default={"kubernetes.io/os":"linux"}
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// If specified, define Pod's tolerations
	// +optional
	// +kubebuilder:default={{"key": "", "operator": "Exists", "value": "", "effect": ""}}
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// If set, defines the security mode and allowed SANames
	Security PowerMonitorKeplerDeploymentSecuritySpec `json:"security,omitempty"`

	// Secrets to be mounted in the power monitor containers
	// +optional
	// +listType=atomic
	Secrets []SecretRef `json:"secrets,omitempty"`
}

// PowerMonitorKeplerConfigSpec defines configuration options for Kepler
type PowerMonitorKeplerConfigSpec struct {
	// LogLevel sets the logging verbosity (e.g., debug, info, warn, error)
	// +kubebuilder:default="info"
	// +optional
	LogLevel string `json:"logLevel,omitempty"`

	// AdditionalConfigMaps is a list of ConfigMap names that will be merged with the default ConfigMap
	// These AdditionalConfigMaps must exist in the same namespace as PowerMonitor components
	// +optional
	// +listType=atomic
	AdditionalConfigMaps []ConfigMapRef `json:"additionalConfigMaps,omitempty"`

	// MetricLevels specifies which metrics levels to export
	// Valid values are combinations of: node, process, container, vm, pod
	// +optional
	// +listType=set
	// +kubebuilder:default={"node","pod","vm"}
	// +kubebuilder:validation:items:Enum=node;process;container;vm;pod
	MetricLevels []string `json:"metricLevels,omitempty"`

	// Staleness specifies how long to wait before considering calculated power values as stale
	// Must be a positive duration (e.g., "500ms", "5s", "1h"). Negative values are not allowed.
	// +optional
	// +kubebuilder:default="500ms"
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Pattern="^[0-9]+(\\.[0-9]+)?(ns|us|ms|s|m|h)$"
	Staleness *metav1.Duration `json:"staleness,omitempty"`

	// SampleRate specifies the interval for monitoring resources (processes, containers, vms, etc.)
	// Must be a positive duration (e.g., "5s", "1m", "30s"). Negative values are not allowed.
	// +optional
	// +kubebuilder:default="5s"
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Pattern="^[0-9]+(\\.[0-9]+)?(ns|us|ms|s|m|h)$"
	SampleRate *metav1.Duration `json:"sampleRate,omitempty"`

	// MaxTerminated controls terminated workload tracking behavior
	// Negative values: track unlimited terminated workloads (no capacity limit)
	// Zero: disable terminated workload tracking completely
	// Positive values: track top N terminated workloads by energy consumption
	// +optional
	// +kubebuilder:default=0
	MaxTerminated *int32 `json:"maxTerminated,omitempty"`
}

// ConfigMapRef defines a reference to a ConfigMap
type ConfigMapRef struct {
	// Name of the ConfigMap
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
}

// SecretRef defines a reference to a Secret to be mounted
//
// Mount Path Cautions:
// Exercise caution when setting mount paths for secrets. Avoid mounting secrets to critical system paths
// that may interfere with Kepler's operation or container security:
// - /etc/kepler - Reserved for Kepler configuration files
// - /sys, /proc, /dev - System directories that should remain read-only
// - /usr, /bin, /sbin, /lib - System binaries and libraries
// - / - Root filesystem
//
// Best practices:
// - Use subdirectories like /etc/kepler/secrets/ or /opt/secrets/
// - Ensure mount paths don't conflict with existing volume mounts
// - Test mount paths in development environments before production deployment
// - Monitor Kepler pod logs for mount-related errors
type SecretRef struct {
	// Name of the secret in the same namespace as the Kepler deployment
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// MountPath where the secret should be mounted in the container
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	MountPath string `json:"mountPath"`

	// ReadOnly specifies whether the secret should be mounted read-only
	// +optional
	// +kubebuilder:default=true
	ReadOnly *bool `json:"readOnly,omitempty"`
}

// PowerMonitorKeplerSpec defines the Kepler component specification
type PowerMonitorKeplerSpec struct {
	// Deployment contains the deployment settings for the Kepler DaemonSet
	Deployment PowerMonitorKeplerDeploymentSpec `json:"deployment,omitempty"`
	// Config contains the configuration options for Kepler
	Config PowerMonitorKeplerConfigSpec `json:"config,omitempty"`
}

// PowerMonitorSpec defines the desired state of Power Monitor
type PowerMonitorSpec struct {
	Kepler PowerMonitorKeplerSpec `json:"kepler"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:scope="Cluster"
//+kubebuilder:subresource:status

// +kubebuilder:printcolumn:name="Desired",type=integer,JSONPath=`.status.kepler.desiredNumberScheduled`
// +kubebuilder:printcolumn:name="Current",type=integer,JSONPath=`.status.kepler.currentNumberScheduled`
// +kubebuilder:printcolumn:name="Ready",type=integer,JSONPath=`.status.kepler.numberReady`
// +kubebuilder:printcolumn:name="Up-to-date",type=integer,JSONPath=`.status.kepler.updatedNumberScheduled`
// +kubebuilder:printcolumn:name="Available",type=integer,JSONPath=`.status.kepler.numberAvailable`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="Node-Selector",type=string,JSONPath=`.spec.kepler.deployment.nodeSelector`,priority=10
// +kubebuilder:printcolumn:name="Tolerations",type=string,JSONPath=`.spec.kepler.deployment.tolerations`,priority=10
//
// PowerMonitor is the Schema for the PowerMonitor API
type PowerMonitor struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PowerMonitorSpec   `json:"spec,omitempty"`
	Status PowerMonitorStatus `json:"status,omitempty"`
}

// PowerMonitorKeplerStatus defines the observed state of the Kepler DaemonSet
type PowerMonitorKeplerStatus struct {
	// CurrentNumberScheduled is the number of nodes that are running at least 1 power-monitor pod and are
	// supposed to run the power-monitor pod.
	CurrentNumberScheduled int32 `json:"currentNumberScheduled"`

	// The number of nodes that are running the power-monitor pod, but are not supposed
	// to run the power-monitor pod.
	NumberMisscheduled int32 `json:"numberMisscheduled"`

	// The total number of nodes that should be running the power-monitor
	// pod (including nodes correctly running the power-monitor pod).
	DesiredNumberScheduled int32 `json:"desiredNumberScheduled"`

	// numberReady is the number of nodes that should be running the power-monitor pod
	// and have one or more of the power-monitor pod running with a Ready Condition.
	NumberReady int32 `json:"numberReady"`

	// The total number of nodes that are running updated power-monitor pod
	// +optional
	UpdatedNumberScheduled int32 `json:"updatedNumberScheduled,omitempty"`

	// The number of nodes that should be running the power-monitor pod and have one or
	// more of the power-monitor pod running and available
	// +optional
	NumberAvailable int32 `json:"numberAvailable,omitempty"`

	// The number of nodes that should be running the
	// power-monitor pod and have none of the power-monitor pod running and available
	// +optional
	NumberUnavailable int32 `json:"numberUnavailable,omitempty"`
}

// ConditionType represents the type of condition for a PowerMonitor resource
type ConditionType string

const (
	// Available indicates whether the PowerMonitor is available and serving metrics
	Available ConditionType = "Available"
	// Reconciled indicates whether the PowerMonitor has been successfully reconciled
	Reconciled ConditionType = "Reconciled"
)

// ConditionReason represents the reason for a condition's last transition
type ConditionReason string

const (
	// ReconcileComplete indicates the CR was successfully reconciled
	ReconcileComplete ConditionReason = "ReconcileSuccess"

	// ReconcileError indicates an error was encountered while reconciling the CR
	ReconcileError ConditionReason = "ReconcileError"

	// DaemonSetNotFound indicates the DaemonSet created for a kepler was not found
	DaemonSetNotFound ConditionReason = "DaemonSetNotFound"
	// DaemonSetError indicates an error occurred with the DaemonSet
	DaemonSetError ConditionReason = "DaemonSetError"
	// DaemonSetInProgess indicates the DaemonSet is being updated
	DaemonSetInProgess ConditionReason = "DaemonSetInProgress"
	// DaemonSetUnavailable indicates no DaemonSet pods are available
	DaemonSetUnavailable ConditionReason = "DaemonSetUnavailable"
	// DaemonSetPartiallyAvailable indicates some but not all DaemonSet pods are available
	DaemonSetPartiallyAvailable ConditionReason = "DaemonSetPartiallyAvailable"
	// DaemonSetPodsNotRunning indicates DaemonSet pods exist but are not running
	DaemonSetPodsNotRunning ConditionReason = "DaemonSetPodsNotRunning"
	// DaemonSetRolloutInProgress indicates a DaemonSet rollout is in progress
	DaemonSetRolloutInProgress ConditionReason = "DaemonSetRolloutInProgress"
	// DaemonSetReady indicates the DaemonSet is fully available and ready
	DaemonSetReady ConditionReason = "DaemonSetReady"
	// DaemonSetOutOfSync indicates the DaemonSet spec doesn't match the desired state
	DaemonSetOutOfSync ConditionReason = "DaemonSetOutOfSync"

	// SecretNotFound indicates one or more referenced secrets are missing
	SecretNotFound ConditionReason = "SecretNotFound"
)

// These are valid condition statuses.
// "ConditionTrue" means a resource is in the condition.
// "ConditionFalse" means a resource is not in the condition.
// "ConditionUnknown" means kubernetes can't decide if a resource is in the condition or not.
// In the future, we could add other intermediate conditions, e.g. ConditionDegraded.
type ConditionStatus string

const (
	// ConditionTrue indicates the condition is met
	ConditionTrue ConditionStatus = "True"
	// ConditionFalse indicates the condition is not met
	ConditionFalse ConditionStatus = "False"
	// ConditionUnknown indicates the condition status cannot be determined
	ConditionUnknown ConditionStatus = "Unknown"
	// ConditionDegraded indicates the resource is operational but in a degraded state
	ConditionDegraded ConditionStatus = "Degraded"
)

type Condition struct {
	// Type of Kepler Condition - Reconciled, Available ...
	Type ConditionType `json:"type"`
	// status of the condition, one of True, False, Unknown.
	Status ConditionStatus `json:"status"`
	//
	// observedGeneration represents the .metadata.generation that the condition was set based upon.
	// For instance, if .metadata.generation is currently 12, but the .status.conditions[x].observedGeneration is 9, the condition is out of date
	// with respect to the current state of the instance.
	// +optional
	// +kubebuilder:validation:Minimum=0
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
	// lastTransitionTime is the last time the condition transitioned from one status to another.
	// This should be when the underlying condition changed.  If that is not known, then using the time when the API field changed is acceptable.
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Format=date-time
	LastTransitionTime metav1.Time `json:"lastTransitionTime"`
	// reason contains a programmatic identifier indicating the reason for the condition's last transition.
	// +required
	Reason ConditionReason `json:"reason"`
	// message is a human readable message indicating details about the transition.
	// This may be an empty string.
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MaxLength=32768
	Message string `json:"message"`
}

// PowerMonitorStatus defines the observed state of Power Monitor
type PowerMonitorStatus struct {
	Kepler PowerMonitorKeplerStatus `json:"kepler,omitempty"`
	// conditions represent the latest available observations of power-monitor
	// +operator-sdk:csv:customresourcedefinitions:type=status,xDescriptors="urn:alm:descriptor:com.tectonic.ui:conditions"
	// +listType=atomic
	Conditions []Condition `json:"conditions"`
}

//+kubebuilder:object:root=true

// PowerMonitorList contains a list of PowerMonitor
type PowerMonitorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PowerMonitor `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PowerMonitor{}, &PowerMonitorList{})
}
