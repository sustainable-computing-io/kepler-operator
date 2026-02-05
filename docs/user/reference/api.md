# API Reference

## Packages
- [kepler.system.sustainable.computing.io/v1alpha1](#keplersystemsustainablecomputingiov1alpha1)


## kepler.system.sustainable.computing.io/v1alpha1

Package v1alpha1 contains API Schema definitions for the kepler.system v1alpha1 API group

### Resource Types
- [PowerMonitor](#powermonitor)
- [PowerMonitorInternal](#powermonitorinternal)
- [PowerMonitorInternalList](#powermonitorinternallist)
- [PowerMonitorList](#powermonitorlist)



#### Condition







_Appears in:_
- [PowerMonitorInternalStatus](#powermonitorinternalstatus)
- [PowerMonitorStatus](#powermonitorstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _[ConditionType](#conditiontype)_ | Type of Kepler Condition - Reconciled, Available ... |  |  |
| `status` _[ConditionStatus](#conditionstatus)_ | status of the condition, one of True, False, Unknown. |  |  |
| `observedGeneration` _integer_ | observedGeneration represents the .metadata.generation that the condition was set based upon.<br />For instance, if .metadata.generation is currently 12, but the .status.conditions[x].observedGeneration is 9, the condition is out of date<br />with respect to the current state of the instance. |  | Minimum: 0 <br /> |
| `lastTransitionTime` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#time-v1-meta)_ | lastTransitionTime is the last time the condition transitioned from one status to another.<br />This should be when the underlying condition changed.  If that is not known, then using the time when the API field changed is acceptable. |  | Format: date-time <br />Required: \{\} <br />Type: string <br /> |
| `reason` _[ConditionReason](#conditionreason)_ | reason contains a programmatic identifier indicating the reason for the condition's last transition. |  |  |
| `message` _string_ | message is a human readable message indicating details about the transition.<br />This may be an empty string. |  | MaxLength: 32768 <br />Required: \{\} <br /> |


#### ConditionReason

_Underlying type:_ _string_

ConditionReason represents the reason for a condition's last transition



_Appears in:_
- [Condition](#condition)

| Field | Description |
| --- | --- |
| `InvalidPowerMonitorResource` | InvalidPowerMonitorResource indicates the CR name was invalid<br /> |
| `ReconcileSuccess` | ReconcileComplete indicates the CR was successfully reconciled<br /> |
| `ReconcileError` | ReconcileError indicates an error was encountered while reconciling the CR<br /> |
| `DaemonSetNotFound` | DaemonSetNotFound indicates the DaemonSet created for a kepler was not found<br /> |
| `DaemonSetError` | DaemonSetError indicates an error occurred with the DaemonSet<br /> |
| `DaemonSetInProgress` | DaemonSetInProgess indicates the DaemonSet is being updated<br /> |
| `DaemonSetUnavailable` | DaemonSetUnavailable indicates no DaemonSet pods are available<br /> |
| `DaemonSetPartiallyAvailable` | DaemonSetPartiallyAvailable indicates some but not all DaemonSet pods are available<br /> |
| `DaemonSetPodsNotRunning` | DaemonSetPodsNotRunning indicates DaemonSet pods exist but are not running<br /> |
| `DaemonSetRolloutInProgress` | DaemonSetRolloutInProgress indicates a DaemonSet rollout is in progress<br /> |
| `DaemonSetReady` | DaemonSetReady indicates the DaemonSet is fully available and ready<br /> |
| `DaemonSetOutOfSync` | DaemonSetOutOfSync indicates the DaemonSet spec doesn't match the desired state<br /> |
| `SecretNotFound` | SecretNotFound indicates one or more referenced secrets are missing<br /> |


#### ConditionStatus

_Underlying type:_ _string_

These are valid condition statuses.
"ConditionTrue" means a resource is in the condition.
"ConditionFalse" means a resource is not in the condition.
"ConditionUnknown" means kubernetes can't decide if a resource is in the condition or not.
In the future, we could add other intermediate conditions, e.g. ConditionDegraded.



_Appears in:_
- [Condition](#condition)

| Field | Description |
| --- | --- |
| `True` | ConditionTrue indicates the condition is met<br /> |
| `False` | ConditionFalse indicates the condition is not met<br /> |
| `Unknown` | ConditionUnknown indicates the condition status cannot be determined<br /> |
| `Degraded` | ConditionDegraded indicates the resource is operational but in a degraded state<br /> |


#### ConditionType

_Underlying type:_ _string_

ConditionType represents the type of condition for a PowerMonitor resource



_Appears in:_
- [Condition](#condition)

| Field | Description |
| --- | --- |
| `Available` | Available indicates whether the PowerMonitor is available and serving metrics<br /> |
| `Reconciled` | Reconciled indicates whether the PowerMonitor has been successfully reconciled<br /> |


#### ConfigMapRef



ConfigMapRef defines a reference to a ConfigMap



_Appears in:_
- [PowerMonitorInternalKeplerConfigSpec](#powermonitorinternalkeplerconfigspec)
- [PowerMonitorKeplerConfigSpec](#powermonitorkeplerconfigspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name of the ConfigMap |  | MinLength: 1 <br /> |


#### PowerMonitor



PowerMonitor is the Schema for the PowerMonitor API



_Appears in:_
- [PowerMonitorList](#powermonitorlist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `kepler.system.sustainable.computing.io/v1alpha1` | | |
| `kind` _string_ | `PowerMonitor` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[PowerMonitorSpec](#powermonitorspec)_ |  |  |  |
| `status` _[PowerMonitorStatus](#powermonitorstatus)_ |  |  |  |






#### PowerMonitorInternal



PowerMonitorInternal is the Schema for the internal kepler 2 API



_Appears in:_
- [PowerMonitorInternalList](#powermonitorinternallist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `kepler.system.sustainable.computing.io/v1alpha1` | | |
| `kind` _string_ | `PowerMonitorInternal` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[PowerMonitorInternalSpec](#powermonitorinternalspec)_ |  |  |  |
| `status` _[PowerMonitorInternalStatus](#powermonitorinternalstatus)_ |  |  |  |


#### PowerMonitorInternalDashboardSpec



PowerMonitorInternalDashboardSpec defines settings for the Kepler Grafana dashboard



_Appears in:_
- [PowerMonitorInternalOpenShiftSpec](#powermonitorinternalopenshiftspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `enabled` _boolean_ | Enabled controls whether to deploy the Grafana dashboard | false |  |


#### PowerMonitorInternalKeplerConfigSpec



PowerMonitorInternalKeplerConfigSpec defines configuration options for internal Kepler deployment



_Appears in:_
- [PowerMonitorInternalKeplerSpec](#powermonitorinternalkeplerspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `logLevel` _string_ | LogLevel sets the logging verbosity (e.g., debug, info, warn, error) | info |  |
| `additionalConfigMaps` _[ConfigMapRef](#configmapref) array_ | AdditionalConfigMaps is a list of ConfigMap names that will be merged with the default ConfigMap<br />These AdditionalConfigMaps must exist in the same namespace as PowerMonitor components |  |  |
| `metricLevels` _string array_ | MetricLevels specifies which metrics levels to export<br />Valid values are combinations of: node, process, container, vm, pod | [node pod vm] | items:Enum: [node process container vm pod] <br /> |
| `staleness` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#duration-v1-meta)_ | Staleness specifies how long to wait before considering calculated power values as stale<br />Must be a positive duration (e.g., "500ms", "5s", "1h"). Negative values are not allowed. | 500ms | Pattern: `^[0-9]+(\.[0-9]+)?(ns\|us\|ms\|s\|m\|h)$` <br />Type: string <br /> |
| `sampleRate` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#duration-v1-meta)_ | SampleRate specifies the interval for monitoring resources (processes, containers, vms, etc.)<br />Must be a positive duration (e.g., "5s", "1m", "30s"). Negative values are not allowed. | 5s | Pattern: `^[0-9]+(\.[0-9]+)?(ns\|us\|ms\|s\|m\|h)$` <br />Type: string <br /> |
| `maxTerminated` _integer_ | MaxTerminated controls terminated workload tracking behavior<br />Negative values: track unlimited terminated workloads (no capacity limit)<br />Zero: disable terminated workload tracking completely<br />Positive values: track top N terminated workloads by energy consumption | 0 |  |


#### PowerMonitorInternalKeplerDeploymentSpec



PowerMonitorInternalKeplerDeploymentSpec extends PowerMonitorKeplerDeploymentSpec with internal deployment settings



_Appears in:_
- [PowerMonitorInternalKeplerSpec](#powermonitorinternalkeplerspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `nodeSelector` _object (keys:string, values:string)_ | NodeSelector defines which Nodes the Pod is scheduled on | \{ kubernetes.io/os:linux \} |  |
| `tolerations` _[Toleration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#toleration-v1-core) array_ | If specified, define Pod's tolerations | [map[effect: key: operator:Exists value:]] |  |
| `security` _[PowerMonitorKeplerDeploymentSecuritySpec](#powermonitorkeplerdeploymentsecurityspec)_ | If set, defines the security mode and allowed SANames |  |  |
| `secrets` _[SecretRef](#secretref) array_ | Secrets to be mounted in the power monitor containers |  |  |
| `image` _string_ | Image specifies the Kepler container image |  | MinLength: 3 <br /> |
| `kubeRbacProxyImage` _string_ | KubeRbacProxyImage specifies the kube-rbac-proxy sidecar image |  | MinLength: 3 <br /> |
| `namespace` _string_ | Namespace specifies the namespace where Kepler will be deployed |  | MinLength: 1 <br /> |


#### PowerMonitorInternalKeplerSpec



PowerMonitorInternalKeplerSpec defines the internal Kepler component specification



_Appears in:_
- [PowerMonitorInternalSpec](#powermonitorinternalspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `deployment` _[PowerMonitorInternalKeplerDeploymentSpec](#powermonitorinternalkeplerdeploymentspec)_ | Deployment contains the deployment settings for the internal Kepler DaemonSet |  | Required: \{\} <br /> |
| `config` _[PowerMonitorInternalKeplerConfigSpec](#powermonitorinternalkeplerconfigspec)_ | Config contains the configuration options for internal Kepler |  |  |


#### PowerMonitorInternalKeplerStatus



PowerMonitorInternalKeplerStatus defines the observed state of the internal Kepler DaemonSet



_Appears in:_
- [PowerMonitorInternalStatus](#powermonitorinternalstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `currentNumberScheduled` _integer_ | CurrentNumberScheduled is the number of nodes that are running at least 1 power-monitor-internal pod and are<br />supposed to run the power-monitor-internal pod. |  |  |
| `numberMisscheduled` _integer_ | The number of nodes that are running the power-monitor-internal pod, but are not supposed<br />to run the power-monitor-internal pod. |  |  |
| `desiredNumberScheduled` _integer_ | The total number of nodes that should be running the power-monitor-internal<br />pod (including nodes correctly running the power-monitor-internal pod). |  |  |
| `numberReady` _integer_ | numberReady is the number of nodes that should be running the power-monitor-internal pod<br />and have one or more of the power-monitor-internal pod running with a Ready Condition. |  |  |
| `updatedNumberScheduled` _integer_ | The total number of nodes that are running updated power-monitor-internal pod |  |  |
| `numberAvailable` _integer_ | The number of nodes that should be running the power-monitor-internal pod and have one or<br />more of the power-monitor-internal pod running and available |  |  |
| `numberUnavailable` _integer_ | The number of nodes that should be running the<br />power-monitor-internal pod and have none of the power-monitor-internal pod running and available |  |  |


#### PowerMonitorInternalList



PowerMonitorInternalList contains a list of PowerMonitorInternal





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `kepler.system.sustainable.computing.io/v1alpha1` | | |
| `kind` _string_ | `PowerMonitorInternalList` | | |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[PowerMonitorInternal](#powermonitorinternal) array_ |  |  |  |


#### PowerMonitorInternalOpenShiftSpec



PowerMonitorInternalOpenShiftSpec defines OpenShift-specific settings



_Appears in:_
- [PowerMonitorInternalSpec](#powermonitorinternalspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `enabled` _boolean_ | Enabled controls whether OpenShift-specific features are enabled | true |  |
| `dashboard` _[PowerMonitorInternalDashboardSpec](#powermonitorinternaldashboardspec)_ | Dashboard configures the Grafana dashboard deployment |  |  |


#### PowerMonitorInternalSpec



PowerMonitorInternalSpec defines the desired state of PowerMonitorInternal



_Appears in:_
- [PowerMonitorInternal](#powermonitorinternal)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `kepler` _[PowerMonitorInternalKeplerSpec](#powermonitorinternalkeplerspec)_ | Kepler contains the Kepler component specification |  | Required: \{\} <br /> |
| `openshift` _[PowerMonitorInternalOpenShiftSpec](#powermonitorinternalopenshiftspec)_ | OpenShift contains OpenShift-specific settings |  |  |


#### PowerMonitorInternalStatus



PowerMonitorInternalStatus defines the observed state of PowerMonitorInternal



_Appears in:_
- [PowerMonitorInternal](#powermonitorinternal)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `kepler` _[PowerMonitorInternalKeplerStatus](#powermonitorinternalkeplerstatus)_ | Kepler contains the status of the internal Kepler DaemonSet |  |  |
| `conditions` _[Condition](#condition) array_ | conditions represent the latest available observations of power-monitor-internal |  |  |


#### PowerMonitorKeplerConfigSpec



PowerMonitorKeplerConfigSpec defines configuration options for Kepler



_Appears in:_
- [PowerMonitorKeplerSpec](#powermonitorkeplerspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `logLevel` _string_ | LogLevel sets the logging verbosity (e.g., debug, info, warn, error) | info |  |
| `additionalConfigMaps` _[ConfigMapRef](#configmapref) array_ | AdditionalConfigMaps is a list of ConfigMap names that will be merged with the default ConfigMap<br />These AdditionalConfigMaps must exist in the same namespace as PowerMonitor components |  |  |
| `metricLevels` _string array_ | MetricLevels specifies which metrics levels to export<br />Valid values are combinations of: node, process, container, vm, pod | [node pod vm] | items:Enum: [node process container vm pod] <br /> |
| `staleness` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#duration-v1-meta)_ | Staleness specifies how long to wait before considering calculated power values as stale<br />Must be a positive duration (e.g., "500ms", "5s", "1h"). Negative values are not allowed. | 500ms | Pattern: `^[0-9]+(\.[0-9]+)?(ns\|us\|ms\|s\|m\|h)$` <br />Type: string <br /> |
| `sampleRate` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#duration-v1-meta)_ | SampleRate specifies the interval for monitoring resources (processes, containers, vms, etc.)<br />Must be a positive duration (e.g., "5s", "1m", "30s"). Negative values are not allowed. | 5s | Pattern: `^[0-9]+(\.[0-9]+)?(ns\|us\|ms\|s\|m\|h)$` <br />Type: string <br /> |
| `maxTerminated` _integer_ | MaxTerminated controls terminated workload tracking behavior<br />Negative values: track unlimited terminated workloads (no capacity limit)<br />Zero: disable terminated workload tracking completely<br />Positive values: track top N terminated workloads by energy consumption | 0 |  |


#### PowerMonitorKeplerDeploymentSecuritySpec



PowerMonitorKeplerDeploymentSecuritySpec defines security settings for the Kepler deployment



_Appears in:_
- [PowerMonitorInternalKeplerDeploymentSpec](#powermonitorinternalkeplerdeploymentspec)
- [PowerMonitorKeplerDeploymentSpec](#powermonitorkeplerdeploymentspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `mode` _[SecurityMode](#securitymode)_ | Mode specifies the security mode (none or rbac) |  | Enum: [none rbac] <br /> |
| `allowedSANames` _string array_ | AllowedSANames lists service account names allowed to access Kepler metrics |  |  |


#### PowerMonitorKeplerDeploymentSpec



PowerMonitorKeplerDeploymentSpec defines deployment settings for the Kepler DaemonSet



_Appears in:_
- [PowerMonitorInternalKeplerDeploymentSpec](#powermonitorinternalkeplerdeploymentspec)
- [PowerMonitorKeplerSpec](#powermonitorkeplerspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `nodeSelector` _object (keys:string, values:string)_ | NodeSelector defines which Nodes the Pod is scheduled on | \{ kubernetes.io/os:linux \} |  |
| `tolerations` _[Toleration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#toleration-v1-core) array_ | If specified, define Pod's tolerations | [map[effect: key: operator:Exists value:]] |  |
| `security` _[PowerMonitorKeplerDeploymentSecuritySpec](#powermonitorkeplerdeploymentsecurityspec)_ | If set, defines the security mode and allowed SANames |  |  |
| `secrets` _[SecretRef](#secretref) array_ | Secrets to be mounted in the power monitor containers |  |  |


#### PowerMonitorKeplerSpec



PowerMonitorKeplerSpec defines the Kepler component specification



_Appears in:_
- [PowerMonitorSpec](#powermonitorspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `deployment` _[PowerMonitorKeplerDeploymentSpec](#powermonitorkeplerdeploymentspec)_ | Deployment contains the deployment settings for the Kepler DaemonSet |  |  |
| `config` _[PowerMonitorKeplerConfigSpec](#powermonitorkeplerconfigspec)_ | Config contains the configuration options for Kepler |  |  |


#### PowerMonitorKeplerStatus



PowerMonitorKeplerStatus defines the observed state of the Kepler DaemonSet



_Appears in:_
- [PowerMonitorStatus](#powermonitorstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `currentNumberScheduled` _integer_ | CurrentNumberScheduled is the number of nodes that are running at least 1 power-monitor pod and are<br />supposed to run the power-monitor pod. |  |  |
| `numberMisscheduled` _integer_ | The number of nodes that are running the power-monitor pod, but are not supposed<br />to run the power-monitor pod. |  |  |
| `desiredNumberScheduled` _integer_ | The total number of nodes that should be running the power-monitor<br />pod (including nodes correctly running the power-monitor pod). |  |  |
| `numberReady` _integer_ | numberReady is the number of nodes that should be running the power-monitor pod<br />and have one or more of the power-monitor pod running with a Ready Condition. |  |  |
| `updatedNumberScheduled` _integer_ | The total number of nodes that are running updated power-monitor pod |  |  |
| `numberAvailable` _integer_ | The number of nodes that should be running the power-monitor pod and have one or<br />more of the power-monitor pod running and available |  |  |
| `numberUnavailable` _integer_ | The number of nodes that should be running the<br />power-monitor pod and have none of the power-monitor pod running and available |  |  |


#### PowerMonitorList



PowerMonitorList contains a list of PowerMonitor





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `kepler.system.sustainable.computing.io/v1alpha1` | | |
| `kind` _string_ | `PowerMonitorList` | | |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[PowerMonitor](#powermonitor) array_ |  |  |  |


#### PowerMonitorSpec



PowerMonitorSpec defines the desired state of Power Monitor



_Appears in:_
- [PowerMonitor](#powermonitor)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `kepler` _[PowerMonitorKeplerSpec](#powermonitorkeplerspec)_ |  |  |  |


#### PowerMonitorStatus



PowerMonitorStatus defines the observed state of Power Monitor



_Appears in:_
- [PowerMonitor](#powermonitor)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `kepler` _[PowerMonitorKeplerStatus](#powermonitorkeplerstatus)_ |  |  |  |
| `conditions` _[Condition](#condition) array_ | conditions represent the latest available observations of power-monitor |  |  |


#### SecretRef



SecretRef defines a reference to a Secret to be mounted

Mount Path Cautions:
Exercise caution when setting mount paths for secrets. Avoid mounting secrets to critical system paths
that may interfere with Kepler's operation or container security:
- /etc/kepler - Reserved for Kepler configuration files
- /sys, /proc, /dev - System directories that should remain read-only
- /usr, /bin, /sbin, /lib - System binaries and libraries
- / - Root filesystem

Best practices:
- Use subdirectories like /etc/kepler/secrets/ or /opt/secrets/
- Ensure mount paths don't conflict with existing volume mounts
- Test mount paths in development environments before production deployment
- Monitor Kepler pod logs for mount-related errors



_Appears in:_
- [PowerMonitorInternalKeplerDeploymentSpec](#powermonitorinternalkeplerdeploymentspec)
- [PowerMonitorKeplerDeploymentSpec](#powermonitorkeplerdeploymentspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name of the secret in the same namespace as the Kepler deployment |  | MinLength: 1 <br />Required: \{\} <br /> |
| `mountPath` _string_ | MountPath where the secret should be mounted in the container |  | MinLength: 1 <br />Required: \{\} <br /> |
| `readOnly` _boolean_ | ReadOnly specifies whether the secret should be mounted read-only | true |  |


#### SecurityMode

_Underlying type:_ _string_

SecurityMode defines the security mode for Kepler metrics access



_Appears in:_
- [PowerMonitorKeplerDeploymentSecuritySpec](#powermonitorkeplerdeploymentsecurityspec)

| Field | Description |
| --- | --- |
| `none` | SecurityModeNone disables RBAC-based access control for Kepler metrics<br /> |
| `rbac` | SecurityModeRBAC enables RBAC-based access control for Kepler metrics<br /> |


