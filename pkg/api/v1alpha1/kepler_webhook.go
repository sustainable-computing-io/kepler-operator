/*
Copyright 2023.

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
	KeplerInstanceName = "kepler"
)

// log is for logging in this package.
var keplerlog = logf.Log.WithName("kepler-resource")

func (r *Kepler) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-kepler-system-sustainable-computing-io-v1alpha1-kepler,mutating=true,failurePolicy=fail,sideEffects=None,groups=kepler.system.sustainable.computing.io,resources=keplers,verbs=create;update,versions=v1alpha1,name=mkepler.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &Kepler{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *Kepler) Default() {
	keplerlog.Info("default", "name", r.Name)

	// TODO(user): fill in your defaulting logic.
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-kepler-system-sustainable-computing-io-v1alpha1-kepler,mutating=false,failurePolicy=fail,sideEffects=None,groups=kepler.system.sustainable.computing.io,resources=keplers,verbs=create;update,versions=v1alpha1,name=vkepler.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &Kepler{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Kepler) ValidateCreate() (admission.Warnings, error) {
	keplerlog.Info("validate create", "name", r.Name)
	if r.Name != KeplerInstanceName {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid name %q; name must be %q", r.Name, KeplerInstanceName))
	}

	// TODO(user): fill in your validation logic upon object creation.
	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Kepler) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	keplerlog.Info("validate update", "name", r.Name)

	// TODO(user): fill in your validation logic upon object update.
	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Kepler) ValidateDelete() (admission.Warnings, error) {
	keplerlog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil, nil
}
