// SPDX-FileCopyrightText: 2025 The Kepler Authors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	PowerMonitorInstanceName = "power-monitor" // Enforce a specific name if needed
)

var pmonLog = logf.Log.WithName("power-monitor-resource")

func (r *PowerMonitor) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-kepler-system-sustainable-computing-io-v1alpha1-powermonitor,mutating=true,failurePolicy=fail,sideEffects=None,groups=kepler.system.sustainable.computing.io,resources=powermonitors,verbs=create;update,versions=v1alpha1,name=mpowermonitor.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &PowerMonitor{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *PowerMonitor) Default() {
	pmonLog.Info("default", "name", r.Name)
}

// +kubebuilder:webhook:path=/validate-kepler-system-sustainable-computing-io-v1alpha1-powermonitor,mutating=false,failurePolicy=fail,sideEffects=None,groups=kepler.system.sustainable.computing.io,resources=powermonitors,verbs=create;update;delete,versions=v1alpha1,name=vpowermonitor.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &PowerMonitor{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *PowerMonitor) ValidateCreate() (admission.Warnings, error) {
	pmonLog.Info("validate create", "name", r.Name)

	// Example: Enforce a specific name if needed
	if r.Name != PowerMonitorInstanceName {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid name %q; name must be %q", r.Name, PowerMonitorInstanceName))
	}

	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *PowerMonitor) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	pmonLog.Info("validate update", "name", r.Name)

	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *PowerMonitor) ValidateDelete() (admission.Warnings, error) {
	pmonLog.Info("validate delete", "name", r.Name)

	return nil, nil
}
