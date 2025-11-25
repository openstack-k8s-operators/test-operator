/*
Copyright 2025.

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

package v1beta1

import (
	"context"
	"errors"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	testv1beta1 "github.com/openstack-k8s-operators/test-operator/api/v1beta1"
)

var (
	// ErrInvalidTempestType is returned when an unexpected object type is passed to the webhook
	ErrInvalidTempestType = errors.New("invalid object type for Tempest webhook")
)

// nolint:unused
// log is for logging in this package.
var tempestlog = logf.Log.WithName("tempest-resource")

// SetupTempestWebhookWithManager registers the webhook for Tempest in the manager.
func SetupTempestWebhookWithManager(mgr ctrl.Manager) error {
	// Set up webhookClient for API webhook functions
	testv1beta1.SetupWebhookClient(mgr.GetClient())

	return ctrl.NewWebhookManagedBy(mgr).For(&testv1beta1.Tempest{}).
		WithValidator(&TempestCustomValidator{}).
		WithDefaulter(&TempestCustomDefaulter{}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate-test-openstack-org-v1beta1-tempest,mutating=true,failurePolicy=fail,sideEffects=None,groups=test.openstack.org,resources=tempests,verbs=create;update,versions=v1beta1,name=mtempest-v1beta1.kb.io,admissionReviewVersions=v1

// TempestCustomDefaulter struct is responsible for setting default values on the custom resource of the
// Kind Tempest when those are created or updated.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as it is used only for temporary operations and does not need to be deeply copied.
type TempestCustomDefaulter struct {
	// TODO(user): Add more fields as needed for defaulting
}

var _ webhook.CustomDefaulter = &TempestCustomDefaulter{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind Tempest.
func (d *TempestCustomDefaulter) Default(_ context.Context, obj runtime.Object) error {
	tempest, ok := obj.(*testv1beta1.Tempest)

	if !ok {
		return fmt.Errorf("expected an Tempest object but got %T: %w", obj, ErrInvalidTempestType)
	}
	tempestlog.Info("Defaulting for Tempest", "name", tempest.GetName())

	// Call the default function from api/v1beta1
	tempest.Default()

	return nil
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-test-openstack-org-v1beta1-tempest,mutating=false,failurePolicy=fail,sideEffects=None,groups=test.openstack.org,resources=tempests,verbs=create;update,versions=v1beta1,name=vtempest-v1beta1.kb.io,admissionReviewVersions=v1

// TempestCustomValidator struct is responsible for validating the Tempest resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type TempestCustomValidator struct {
	// TODO(user): Add more fields as needed for validation
}

var _ webhook.CustomValidator = &TempestCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type Tempest.
func (v *TempestCustomValidator) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	tempest, ok := obj.(*testv1beta1.Tempest)
	if !ok {
		return nil, fmt.Errorf("expected a Tempest object but got %T: %w", obj, ErrInvalidTempestType)
	}
	tempestlog.Info("Validation for Tempest upon creation", "name", tempest.GetName())

	// Call the validation function from api/v1beta1
	return tempest.ValidateCreate()
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type Tempest.
func (v *TempestCustomValidator) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	tempest, ok := newObj.(*testv1beta1.Tempest)
	if !ok {
		return nil, fmt.Errorf("expected a Tempest object for the newObj but got %T: %w", newObj, ErrInvalidTempestType)
	}
	tempestlog.Info("Validation for Tempest upon update", "name", tempest.GetName())

	// Call the validation function from api/v1beta1
	return tempest.ValidateUpdate(oldObj)
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type Tempest.
func (v *TempestCustomValidator) ValidateDelete(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	tempest, ok := obj.(*testv1beta1.Tempest)
	if !ok {
		return nil, fmt.Errorf("expected a Tempest object but got %T: %w", obj, ErrInvalidTempestType)
	}
	tempestlog.Info("Validation for Tempest upon deletion", "name", tempest.GetName())

	// Call the validation function from api/v1beta1
	return tempest.ValidateDelete()
}
