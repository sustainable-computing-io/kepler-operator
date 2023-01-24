/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"strings"

	mcfgv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
)

func (r *collectorReconciler) ensureMachineConfig(l klog.Logger) (bool, error) {

	logger := l.WithValues("machineConfig", types.NamespacedName{Name: "50-master-cgroupv2", Namespace: ""})

	var master_labels_map = make(map[string]string)
	master_labels_map["machineconfiguration.openshift.io/role"] = "master"

	var worker_labels_map = make(map[string]string)
	worker_labels_map["machineconfiguration.openshift.io/role"] = "worker"

	mc_cgroupv2_master := &mcfgv1.MachineConfig{
		TypeMeta: metav1.TypeMeta{
			Kind:       "MachineConfig",
			APIVersion: "machineconfiguration.openshift.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      MachineConfigCGroupKernelArgMasterNameSuffix,
			Namespace: MachineConfigCGroupKernelArgMasterNameSpaceSuffix,
			Labels:    master_labels_map,
		},
		Spec: mcfgv1.MachineConfigSpec{
			KernelArguments: []string{"systemd.unified_cgroup_hierarchy=1", "cgroup_no_v1='all'"},
		},
	}

	found := &mcfgv1.MachineConfig{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: MachineConfigCGroupKernelArgMasterNameSuffix, Namespace: MachineConfigCGroupKernelArgMasterNameSpaceSuffix}, found)
	if err != nil {
		if strings.Contains(err.Error(), "no matches for kind") {
			fmt.Printf("resulting error not a timeout: %s", err)
			logger.V(1).Info("Not OpenShift skipping MachineConfig")
			return true, nil
		}
	}

	if err != nil && !apierrors.IsNotFound(err) {

		return false, err
	}
	if apierrors.IsNotFound(err) {
		err = r.Client.Create(context.TODO(), mc_cgroupv2_master)
		if err != nil {
			return false, err
		}
	}
	logger.V(1).Info("MachineConfig", "MachineConfig", mc_cgroupv2_master)

	mc_cgroupv2_worker := &mcfgv1.MachineConfig{
		TypeMeta: metav1.TypeMeta{
			Kind:       "MachineConfig",
			APIVersion: "machineconfiguration.openshift.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      MachineConfigCGroupKernelArgWorkerNameSuffix,
			Namespace: MachineConfigCGroupKernelArgWorkerNameSpaceSuffix,
			Labels:    worker_labels_map,
		},
		Spec: mcfgv1.MachineConfigSpec{
			KernelArguments: []string{"systemd.unified_cgroup_hierarchy=1", "cgroup_no_v1='all'"},
		},
	}

	err = r.Client.Get(context.TODO(), types.NamespacedName{Name: MachineConfigCGroupKernelArgWorkerNameSuffix, Namespace: MachineConfigCGroupKernelArgWorkerNameSpaceSuffix}, found)

	if err != nil && !apierrors.IsNotFound(err) {

		return false, err
	}
	if apierrors.IsNotFound(err) {
		err = r.Client.Create(context.TODO(), mc_cgroupv2_worker)
		if err != nil {
			return false, err
		}
	}
	logger.V(1).Info("MachineConfig", "MachineConfig", mc_cgroupv2_worker)

	mc_kernel_devel_master := &mcfgv1.MachineConfig{
		TypeMeta: metav1.TypeMeta{
			Kind:       "MachineConfig",
			APIVersion: "machineconfiguration.openshift.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      MachineConfigDevelMasterNameSuffix,
			Namespace: MachineConfigDevelMasterNameSpaceSuffix,
			Labels:    master_labels_map,
		},
		Spec: mcfgv1.MachineConfigSpec{
			Extensions: []string{"kernel-devel"},
		},
	}

	err = r.Client.Get(context.TODO(), types.NamespacedName{Name: MachineConfigDevelMasterNameSuffix, Namespace: MachineConfigDevelMasterNameSpaceSuffix}, found)

	if err != nil && !apierrors.IsNotFound(err) {

		return false, err
	}
	if apierrors.IsNotFound(err) {
		err = r.Client.Create(context.TODO(), mc_kernel_devel_master)
		if err != nil {
			return false, err
		}
	}
	logger.V(1).Info("MachineConfig", "MachineConfig", mc_kernel_devel_master)

	mc_kernel_devel_worker := &mcfgv1.MachineConfig{
		TypeMeta: metav1.TypeMeta{
			Kind:       "MachineConfig",
			APIVersion: "machineconfiguration.openshift.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      MachineConfigDevelWorkerNameSuffix,
			Namespace: MachineConfigDevelWorkerNameSpaceSuffix,
			Labels:    worker_labels_map,
		},
		Spec: mcfgv1.MachineConfigSpec{
			Extensions: []string{"kernel-devel"},
		},
	}

	err = r.Client.Get(context.TODO(), types.NamespacedName{Name: MachineConfigDevelWorkerNameSuffix, Namespace: MachineConfigDevelWorkerNameSpaceSuffix}, found)

	if err != nil && !apierrors.IsNotFound(err) {

		return false, err
	}
	if apierrors.IsNotFound(err) {
		err = r.Client.Create(context.TODO(), mc_kernel_devel_worker)
		if err != nil {
			return false, err
		}
	}
	logger.V(1).Info("MachineConfig", "MachineConfig", mc_kernel_devel_worker)

	return true, err
}
