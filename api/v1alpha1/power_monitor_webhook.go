/*
Copyright 2024.

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
	PowerMonitorInstanceName = "powermonitor" // Enforce a specific name if needed
)

var powermonitorlog = logf.Log.WithName("powermonitor-resource")

func (r *PowerMonitor) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-kepler-system-sustainable-computing-io-v1alpha1-powermonitor,mutating=true,failurePolicy=fail,sideEffects=None,groups=kepler.system.sustainable.computing.io,resources=powermonitors,verbs=create;update,versions=v1alpha1,name=mpowermonitor.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &PowerMonitor{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *PowerMonitor) Default() {
	powermonitorlog.Info("default", "name", r.Name)
}

// +kubebuilder:webhook:path=/validate-kepler-system-sustainable-computing-io-v1alpha1-powermonitor,mutating=false,failurePolicy=fail,sideEffects=None,groups=kepler.system.sustainable.computing.io,resources=powermonitors,verbs=create;update;delete,versions=v1alpha1,name=vpowermonitor.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &PowerMonitor{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *PowerMonitor) ValidateCreate() (admission.Warnings, error) {
	powermonitorlog.Info("validate create", "name", r.Name)

	// Example: Enforce a specific name if needed
	if r.Name != PowerMonitorInstanceName {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid name %q; name must be %q", r.Name, PowerMonitorInstanceName))
	}

	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *PowerMonitor) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	powermonitorlog.Info("validate update", "name", r.Name)

	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *PowerMonitor) ValidateDelete() (admission.Warnings, error) {
	powermonitorlog.Info("validate delete", "name", r.Name)

	return nil, nil
}
