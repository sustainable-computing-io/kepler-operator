// SPDX-FileCopyrightText: 2025 The Kepler Authors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"context"
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

type SecurityConfig struct {
	AllowedSANames []string
	Mode           SecurityMode
}

var DefaultSecurityConfig = SecurityConfig{
	AllowedSANames: nil,
	Mode:           SecurityModeNone,
}

var pmonLog = logf.Log.WithName("power-monitor-resource")

// SetupWebhookWithManager registers the webhook for PowerMonitor in the manager.
func SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&PowerMonitor{}).
		WithValidator(&PowerMonitorCustomValidator{}).
		WithDefaulter(&PowerMonitorCustomDefaulter{}).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-kepler-system-sustainable-computing-io-v1alpha1-powermonitor,mutating=true,failurePolicy=fail,sideEffects=None,groups=kepler.system.sustainable.computing.io,resources=powermonitors,verbs=create;update,versions=v1alpha1,name=mpowermonitor.kb.io,admissionReviewVersions=v1

// PowerMonitorCustomDefaulter struct is responsible for setting default values on the custom resource of the
// Kind PowerMonitor when those are created or updated.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as it is used only for temporary operations and does not need to be deeply copied.
type PowerMonitorCustomDefaulter struct {
	// TODO(user): Add more fields as needed for defaulting
}

var _ webhook.CustomDefaulter = &PowerMonitorCustomDefaulter{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind PowerMonitor.
func (d *PowerMonitorCustomDefaulter) Default(_ context.Context, obj runtime.Object) error {
	powerMonitor, ok := obj.(*PowerMonitor)

	if !ok {
		return fmt.Errorf("expected a PowerMonitor object but got %T", obj)
	}
	pmonLog.Info("Defaulting for PowerMonitor", "name", powerMonitor.GetName())

	if powerMonitor.Spec.Kepler.Deployment.Security.Mode == "" {
		pmonLog.Info("default", "mode", DefaultSecurityConfig.Mode)
		powerMonitor.Spec.Kepler.Deployment.Security.Mode = DefaultSecurityConfig.Mode
	}
	if powerMonitor.Spec.Kepler.Deployment.Security.AllowedSANames == nil {
		pmonLog.Info("default", "allowed sa", DefaultSecurityConfig.AllowedSANames)
		powerMonitor.Spec.Kepler.Deployment.Security.AllowedSANames = DefaultSecurityConfig.AllowedSANames
	}

	return nil
}

// +kubebuilder:webhook:path=/validate-kepler-system-sustainable-computing-io-v1alpha1-powermonitor,mutating=false,failurePolicy=fail,sideEffects=None,groups=kepler.system.sustainable.computing.io,resources=powermonitors,verbs=create;update;delete,versions=v1alpha1,name=vpowermonitor.kb.io,admissionReviewVersions=v1

// PowerMonitorCustomValidator struct is responsible for validating the PowerMonitor resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type PowerMonitorCustomValidator struct {
	// TODO(user): Add more fields as needed for validation
}

var _ webhook.CustomValidator = &PowerMonitorCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type PowerMonitor.
func (v *PowerMonitorCustomValidator) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	powerMonitor, ok := obj.(*PowerMonitor)
	if !ok {
		return nil, fmt.Errorf("expected a PowerMonitor object but got %T", obj)
	}
	pmonLog.Info("Validation for PowerMonitor upon creation", "name", powerMonitor.GetName())

	// Enforce a specific name if needed
	if powerMonitor.Name != PowerMonitorInstanceName {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid name %q; name must be %q", powerMonitor.Name, PowerMonitorInstanceName))
	}

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type PowerMonitor.
func (v *PowerMonitorCustomValidator) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	powerMonitor, ok := newObj.(*PowerMonitor)
	if !ok {
		return nil, fmt.Errorf("expected a PowerMonitor object for the newObj but got %T", newObj)
	}
	pmonLog.Info("Validation for PowerMonitor upon update", "name", powerMonitor.GetName())

	// TODO(user): fill in your validation logic upon object update.

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type PowerMonitor.
func (v *PowerMonitorCustomValidator) ValidateDelete(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	powerMonitor, ok := obj.(*PowerMonitor)
	if !ok {
		return nil, fmt.Errorf("expected a PowerMonitor object but got %T", obj)
	}
	pmonLog.Info("Validation for PowerMonitor upon deletion", "name", powerMonitor.GetName())

	// TODO(user): fill in your validation logic upon object deletion.

	return nil, nil
}
