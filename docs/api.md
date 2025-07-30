# API Reference

Packages:

- [kepler.system.sustainable.computing.io/v1alpha1](#keplersystemsustainablecomputingiov1alpha1)

# kepler.system.sustainable.computing.io/v1alpha1

Resource Types:

- [PowerMonitorInternal](#powermonitorinternal)

- [PowerMonitor](#powermonitor)




## PowerMonitorInternal
<sup><sup>[↩ Parent](#keplersystemsustainablecomputingiov1alpha1 )</sup></sup>






PowerMonitorInternal is the Schema for the internal kepler 2 API

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
      <td><b>apiVersion</b></td>
      <td>string</td>
      <td>kepler.system.sustainable.computing.io/v1alpha1</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>PowerMonitorInternal</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#powermonitorinternalspec">spec</a></b></td>
        <td>object</td>
        <td>
          PowerMonitorInternalSpec defines the desired state of PowerMonitorInternalSpec<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#powermonitorinternalstatus">status</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### PowerMonitorInternal.spec
<sup><sup>[↩ Parent](#powermonitorinternal)</sup></sup>



PowerMonitorInternalSpec defines the desired state of PowerMonitorInternalSpec

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#powermonitorinternalspeckepler">kepler</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#powermonitorinternalspecopenshift">openshift</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### PowerMonitorInternal.spec.kepler
<sup><sup>[↩ Parent](#powermonitorinternalspec)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#powermonitorinternalspeckeplerdeployment">deployment</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#powermonitorinternalspeckeplerconfig">config</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### PowerMonitorInternal.spec.kepler.deployment
<sup><sup>[↩ Parent](#powermonitorinternalspeckepler)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>image</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>namespace</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>kubeRbacProxyImage</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>nodeSelector</b></td>
        <td>map[string]string</td>
        <td>
          Defines which Nodes the Pod is scheduled on<br/>
          <br/>
            <i>Default</i>: map[kubernetes.io/os:linux]<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#powermonitorinternalspeckeplerdeploymentsecurity">security</a></b></td>
        <td>object</td>
        <td>
          If set, defines the security mode and allowed SANames<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#powermonitorinternalspeckeplerdeploymenttolerationsindex">tolerations</a></b></td>
        <td>[]object</td>
        <td>
          If specified, define Pod's tolerations<br/>
          <br/>
            <i>Default</i>: [map[effect: key: operator:Exists value:]]<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### PowerMonitorInternal.spec.kepler.deployment.security
<sup><sup>[↩ Parent](#powermonitorinternalspeckeplerdeployment)</sup></sup>



If set, defines the security mode and allowed SANames

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>allowedSANames</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>mode</b></td>
        <td>enum</td>
        <td>
          <br/>
          <br/>
            <i>Enum</i>: none, rbac<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### PowerMonitorInternal.spec.kepler.deployment.tolerations[index]
<sup><sup>[↩ Parent](#powermonitorinternalspeckeplerdeployment)</sup></sup>



The pod this Toleration is attached to tolerates any taint that matches
the triple <key,value,effect> using the matching operator <operator>.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>effect</b></td>
        <td>string</td>
        <td>
          Effect indicates the taint effect to match. Empty means match all taint effects.
When specified, allowed values are NoSchedule, PreferNoSchedule and NoExecute.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          Key is the taint key that the toleration applies to. Empty means match all taint keys.
If the key is empty, operator must be Exists; this combination means to match all values and all keys.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          Operator represents a key's relationship to the value.
Valid operators are Exists and Equal. Defaults to Equal.
Exists is equivalent to wildcard for value, so that a pod can
tolerate all taints of a particular category.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>tolerationSeconds</b></td>
        <td>integer</td>
        <td>
          TolerationSeconds represents the period of time the toleration (which must be
of effect NoExecute, otherwise this field is ignored) tolerates the taint. By default,
it is not set, which means tolerate the taint forever (do not evict). Zero and
negative values will be treated as 0 (evict immediately) by the system.<br/>
          <br/>
            <i>Format</i>: int64<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          Value is the taint value the toleration matches to.
If the operator is Exists, the value should be empty, otherwise just a regular string.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### PowerMonitorInternal.spec.kepler.config
<sup><sup>[↩ Parent](#powermonitorinternalspeckepler)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#powermonitorinternalspeckeplerconfigadditionalconfigmapsindex">additionalConfigMaps</a></b></td>
        <td>[]object</td>
        <td>
          AdditionalConfigMaps is a list of ConfigMap names that will be merged with the default ConfigMap
These AdditionalConfigMaps must exist in the same namespace as PowerMonitor components<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>logLevel</b></td>
        <td>string</td>
        <td>
          <br/>
          <br/>
            <i>Default</i>: info<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>maxTerminated</b></td>
        <td>integer</td>
        <td>
          MaxTerminated controls terminated workload tracking behavior
Negative values: track unlimited terminated workloads (no capacity limit)
Zero: disable terminated workload tracking completely
Positive values: track top N terminated workloads by energy consumption<br/>
          <br/>
            <i>Format</i>: int32<br/>
            <i>Default</i>: 500<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>metricLevels</b></td>
        <td>[]enum</td>
        <td>
          MetricLevels specifies which metrics levels to export
Valid values are combinations of: node, process, container, vm, pod<br/>
          <br/>
            <i>Default</i>: [node pod vm]<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>sampleRate</b></td>
        <td>string</td>
        <td>
          SampleRate specifies the interval for monitoring resources (processes, containers, vms, etc.)
Must be a positive duration (e.g., "5s", "1m", "30s"). Negative values are not allowed.<br/>
          <br/>
            <i>Default</i>: 5s<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>staleness</b></td>
        <td>string</td>
        <td>
          Staleness specifies how long to wait before considering calculated power values as stale
Must be a positive duration (e.g., "500ms", "5s", "1h"). Negative values are not allowed.<br/>
          <br/>
            <i>Default</i>: 500ms<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### PowerMonitorInternal.spec.kepler.config.additionalConfigMaps[index]
<sup><sup>[↩ Parent](#powermonitorinternalspeckeplerconfig)</sup></sup>



ConfigMapRef defines a reference to a ConfigMap

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the ConfigMap<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### PowerMonitorInternal.spec.openshift
<sup><sup>[↩ Parent](#powermonitorinternalspec)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>enabled</b></td>
        <td>boolean</td>
        <td>
          <br/>
          <br/>
            <i>Default</i>: true<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#powermonitorinternalspecopenshiftdashboard">dashboard</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### PowerMonitorInternal.spec.openshift.dashboard
<sup><sup>[↩ Parent](#powermonitorinternalspecopenshift)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>enabled</b></td>
        <td>boolean</td>
        <td>
          <br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### PowerMonitorInternal.status
<sup><sup>[↩ Parent](#powermonitorinternal)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#powermonitorinternalstatusconditionsindex">conditions</a></b></td>
        <td>[]object</td>
        <td>
          conditions represent the latest available observations of power-monitor-internal<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#powermonitorinternalstatuskepler">kepler</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### PowerMonitorInternal.status.conditions[index]
<sup><sup>[↩ Parent](#powermonitorinternalstatus)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>lastTransitionTime</b></td>
        <td>string</td>
        <td>
          lastTransitionTime is the last time the condition transitioned from one status to another.
This should be when the underlying condition changed.  If that is not known, then using the time when the API field changed is acceptable.<br/>
          <br/>
            <i>Format</i>: date-time<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>message</b></td>
        <td>string</td>
        <td>
          message is a human readable message indicating details about the transition.
This may be an empty string.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>reason</b></td>
        <td>string</td>
        <td>
          reason contains a programmatic identifier indicating the reason for the condition's last transition.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>status</b></td>
        <td>string</td>
        <td>
          status of the condition, one of True, False, Unknown.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          Type of Kepler Condition - Reconciled, Available ...<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>observedGeneration</b></td>
        <td>integer</td>
        <td>
          observedGeneration represents the .metadata.generation that the condition was set based upon.
For instance, if .metadata.generation is currently 12, but the .status.conditions[x].observedGeneration is 9, the condition is out of date
with respect to the current state of the instance.<br/>
          <br/>
            <i>Format</i>: int64<br/>
            <i>Minimum</i>: 0<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### PowerMonitorInternal.status.kepler
<sup><sup>[↩ Parent](#powermonitorinternalstatus)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>currentNumberScheduled</b></td>
        <td>integer</td>
        <td>
          The number of nodes that are running at least 1 power-monitor-internal pod and are
supposed to run the power-monitor-internal pod.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>desiredNumberScheduled</b></td>
        <td>integer</td>
        <td>
          The total number of nodes that should be running the power-monitor-internal
pod (including nodes correctly running the power-monitor-internal pod).<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>numberMisscheduled</b></td>
        <td>integer</td>
        <td>
          The number of nodes that are running the power-monitor-internal pod, but are not supposed
to run the power-monitor-internal pod.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>numberReady</b></td>
        <td>integer</td>
        <td>
          numberReady is the number of nodes that should be running the power-monitor-internal pod
and have one or more of the power-monitor-internal pod running with a Ready Condition.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>numberAvailable</b></td>
        <td>integer</td>
        <td>
          The number of nodes that should be running the power-monitor-internal pod and have one or
more of the power-monitor-internal pod running and available<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>numberUnavailable</b></td>
        <td>integer</td>
        <td>
          The number of nodes that should be running the
power-monitor-internal pod and have none of the power-monitor-internal pod running and available<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>updatedNumberScheduled</b></td>
        <td>integer</td>
        <td>
          The total number of nodes that are running updated power-monitor-internal pod<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

## PowerMonitor
<sup><sup>[↩ Parent](#keplersystemsustainablecomputingiov1alpha1 )</sup></sup>






PowerMonitor is the Schema for the PowerMonitor API

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
      <td><b>apiVersion</b></td>
      <td>string</td>
      <td>kepler.system.sustainable.computing.io/v1alpha1</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>PowerMonitor</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#powermonitorspec">spec</a></b></td>
        <td>object</td>
        <td>
          PowerMonitorSpec defines the desired state of Power Monitor<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#powermonitorstatus">status</a></b></td>
        <td>object</td>
        <td>
          PowerMonitorStatus defines the observed state of Power Monitor<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### PowerMonitor.spec
<sup><sup>[↩ Parent](#powermonitor)</sup></sup>



PowerMonitorSpec defines the desired state of Power Monitor

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#powermonitorspeckepler">kepler</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### PowerMonitor.spec.kepler
<sup><sup>[↩ Parent](#powermonitorspec)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#powermonitorspeckeplerconfig">config</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#powermonitorspeckeplerdeployment">deployment</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### PowerMonitor.spec.kepler.config
<sup><sup>[↩ Parent](#powermonitorspeckepler)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#powermonitorspeckeplerconfigadditionalconfigmapsindex">additionalConfigMaps</a></b></td>
        <td>[]object</td>
        <td>
          AdditionalConfigMaps is a list of ConfigMap names that will be merged with the default ConfigMap
These AdditionalConfigMaps must exist in the same namespace as PowerMonitor components<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>logLevel</b></td>
        <td>string</td>
        <td>
          <br/>
          <br/>
            <i>Default</i>: info<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>maxTerminated</b></td>
        <td>integer</td>
        <td>
          MaxTerminated controls terminated workload tracking behavior
Negative values: track unlimited terminated workloads (no capacity limit)
Zero: disable terminated workload tracking completely
Positive values: track top N terminated workloads by energy consumption<br/>
          <br/>
            <i>Format</i>: int32<br/>
            <i>Default</i>: 500<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>metricLevels</b></td>
        <td>[]enum</td>
        <td>
          MetricLevels specifies which metrics levels to export
Valid values are combinations of: node, process, container, vm, pod<br/>
          <br/>
            <i>Default</i>: [node pod vm]<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>sampleRate</b></td>
        <td>string</td>
        <td>
          SampleRate specifies the interval for monitoring resources (processes, containers, vms, etc.)
Must be a positive duration (e.g., "5s", "1m", "30s"). Negative values are not allowed.<br/>
          <br/>
            <i>Default</i>: 5s<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>staleness</b></td>
        <td>string</td>
        <td>
          Staleness specifies how long to wait before considering calculated power values as stale
Must be a positive duration (e.g., "500ms", "5s", "1h"). Negative values are not allowed.<br/>
          <br/>
            <i>Default</i>: 500ms<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### PowerMonitor.spec.kepler.config.additionalConfigMaps[index]
<sup><sup>[↩ Parent](#powermonitorspeckeplerconfig)</sup></sup>



ConfigMapRef defines a reference to a ConfigMap

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the ConfigMap<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### PowerMonitor.spec.kepler.deployment
<sup><sup>[↩ Parent](#powermonitorspeckepler)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>nodeSelector</b></td>
        <td>map[string]string</td>
        <td>
          Defines which Nodes the Pod is scheduled on<br/>
          <br/>
            <i>Default</i>: map[kubernetes.io/os:linux]<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#powermonitorspeckeplerdeploymentsecurity">security</a></b></td>
        <td>object</td>
        <td>
          If set, defines the security mode and allowed SANames<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#powermonitorspeckeplerdeploymenttolerationsindex">tolerations</a></b></td>
        <td>[]object</td>
        <td>
          If specified, define Pod's tolerations<br/>
          <br/>
            <i>Default</i>: [map[effect: key: operator:Exists value:]]<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### PowerMonitor.spec.kepler.deployment.security
<sup><sup>[↩ Parent](#powermonitorspeckeplerdeployment)</sup></sup>



If set, defines the security mode and allowed SANames

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>allowedSANames</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>mode</b></td>
        <td>enum</td>
        <td>
          <br/>
          <br/>
            <i>Enum</i>: none, rbac<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### PowerMonitor.spec.kepler.deployment.tolerations[index]
<sup><sup>[↩ Parent](#powermonitorspeckeplerdeployment)</sup></sup>



The pod this Toleration is attached to tolerates any taint that matches
the triple <key,value,effect> using the matching operator <operator>.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>effect</b></td>
        <td>string</td>
        <td>
          Effect indicates the taint effect to match. Empty means match all taint effects.
When specified, allowed values are NoSchedule, PreferNoSchedule and NoExecute.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          Key is the taint key that the toleration applies to. Empty means match all taint keys.
If the key is empty, operator must be Exists; this combination means to match all values and all keys.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          Operator represents a key's relationship to the value.
Valid operators are Exists and Equal. Defaults to Equal.
Exists is equivalent to wildcard for value, so that a pod can
tolerate all taints of a particular category.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>tolerationSeconds</b></td>
        <td>integer</td>
        <td>
          TolerationSeconds represents the period of time the toleration (which must be
of effect NoExecute, otherwise this field is ignored) tolerates the taint. By default,
it is not set, which means tolerate the taint forever (do not evict). Zero and
negative values will be treated as 0 (evict immediately) by the system.<br/>
          <br/>
            <i>Format</i>: int64<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          Value is the taint value the toleration matches to.
If the operator is Exists, the value should be empty, otherwise just a regular string.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### PowerMonitor.status
<sup><sup>[↩ Parent](#powermonitor)</sup></sup>



PowerMonitorStatus defines the observed state of Power Monitor

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#powermonitorstatusconditionsindex">conditions</a></b></td>
        <td>[]object</td>
        <td>
          conditions represent the latest available observations of power-monitor<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#powermonitorstatuskepler">kepler</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### PowerMonitor.status.conditions[index]
<sup><sup>[↩ Parent](#powermonitorstatus)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>lastTransitionTime</b></td>
        <td>string</td>
        <td>
          lastTransitionTime is the last time the condition transitioned from one status to another.
This should be when the underlying condition changed.  If that is not known, then using the time when the API field changed is acceptable.<br/>
          <br/>
            <i>Format</i>: date-time<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>message</b></td>
        <td>string</td>
        <td>
          message is a human readable message indicating details about the transition.
This may be an empty string.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>reason</b></td>
        <td>string</td>
        <td>
          reason contains a programmatic identifier indicating the reason for the condition's last transition.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>status</b></td>
        <td>string</td>
        <td>
          status of the condition, one of True, False, Unknown.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          Type of Kepler Condition - Reconciled, Available ...<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>observedGeneration</b></td>
        <td>integer</td>
        <td>
          observedGeneration represents the .metadata.generation that the condition was set based upon.
For instance, if .metadata.generation is currently 12, but the .status.conditions[x].observedGeneration is 9, the condition is out of date
with respect to the current state of the instance.<br/>
          <br/>
            <i>Format</i>: int64<br/>
            <i>Minimum</i>: 0<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### PowerMonitor.status.kepler
<sup><sup>[↩ Parent](#powermonitorstatus)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>currentNumberScheduled</b></td>
        <td>integer</td>
        <td>
          The number of nodes that are running at least 1 power-monitor pod and are
supposed to run the power-monitor pod.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>desiredNumberScheduled</b></td>
        <td>integer</td>
        <td>
          The total number of nodes that should be running the power-monitor
pod (including nodes correctly running the power-monitor pod).<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>numberMisscheduled</b></td>
        <td>integer</td>
        <td>
          The number of nodes that are running the power-monitor pod, but are not supposed
to run the power-monitor pod.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>numberReady</b></td>
        <td>integer</td>
        <td>
          numberReady is the number of nodes that should be running the power-monitor pod
and have one or more of the power-monitor pod running with a Ready Condition.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>numberAvailable</b></td>
        <td>integer</td>
        <td>
          The number of nodes that should be running the power-monitor pod and have one or
more of the power-monitor pod running and available<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>numberUnavailable</b></td>
        <td>integer</td>
        <td>
          The number of nodes that should be running the
power-monitor pod and have none of the power-monitor pod running and available<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>updatedNumberScheduled</b></td>
        <td>integer</td>
        <td>
          The total number of nodes that are running updated power-monitor pod<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>