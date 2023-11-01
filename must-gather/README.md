
power monitoring must-gather
============================

power monitoring `must-gather` is a tool built on top of [Openshift must-gather](https://github.com/openshift/must-gather) that lets users to gather information about power monitoring components.

### Usage
```sh
oc adm must-gather --image=$(oc -n openshift-operators get deployment.apps/kepler-operator-controller -o jsonpath='{.spec.template.spec.containers[?(@.name == "manager")].image}') -- /usr/bin/gather
```
or
```sh
oc adm must-gather --image=quay.io/sustainable_computing_io/kepler-operator:v1alpha1
```

The above command will gather information about power monitoring and dump that information in a new directory.


