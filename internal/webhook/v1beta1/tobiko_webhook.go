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
	// ErrInvalidTobikoType is returned when an unexpected object type is passed to the webhook
	ErrInvalidTobikoType = errors.New("invalid object type for Tobiko webhook")
)

// nolint:unused
// log is for logging in this package.
var tobikolog = logf.Log.WithName("tobiko-resource")

// SetupTobikoWebhookWithManager registers the webhook for Tobiko in the manager.
func SetupTobikoWebhookWithManager(mgr ctrl.Manager) error {
	// Set up webhookClient for API webhook functions
	testv1beta1.SetupWebhookClient(mgr.GetClient())

	return ctrl.NewWebhookManagedBy(mgr).For(&testv1beta1.Tobiko{}).
		WithValidator(&TobikoCustomValidator{}).
		WithDefaulter(&TobikoCustomDefaulter{}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate-test-openstack-org-v1beta1-tobiko,mutating=true,failurePolicy=fail,sideEffects=None,groups=test.openstack.org,resources=tobikoes,verbs=create;update,versions=v1beta1,name=mtobiko-v1beta1.kb.io,admissionReviewVersions=v1

// TobikoCustomDefaulter struct is responsible for setting default values on the custom resource of the
// Kind Tobiko when those are created or updated.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as it is used only for temporary operations and does not need to be deeply copied.
type TobikoCustomDefaulter struct {
	// TODO(user): Add more fields as needed for defaulting
}

var _ webhook.CustomDefaulter = &TobikoCustomDefaulter{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind Tobiko.
func (d *TobikoCustomDefaulter) Default(_ context.Context, obj runtime.Object) error {
	tobiko, ok := obj.(*testv1beta1.Tobiko)

	if !ok {
		return fmt.Errorf("expected an Tobiko object but got %T: %w", obj, ErrInvalidTobikoType)
	}
	tobikolog.Info("Defaulting for Tobiko", "name", tobiko.GetName())

	// Call the default function from api/v1beta1
	tobiko.Default()

	return nil
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-test-openstack-org-v1beta1-tobiko,mutating=false,failurePolicy=fail,sideEffects=None,groups=test.openstack.org,resources=tobikoes,verbs=create;update,versions=v1beta1,name=vtobiko-v1beta1.kb.io,admissionReviewVersions=v1

// TobikoCustomValidator struct is responsible for validating the Tobiko resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type TobikoCustomValidator struct {
	// TODO(user): Add more fields as needed for validation
}

var _ webhook.CustomValidator = &TobikoCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type Tobiko.
func (v *TobikoCustomValidator) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	tobiko, ok := obj.(*testv1beta1.Tobiko)
	if !ok {
		return nil, fmt.Errorf("expected a Tobiko object but got %T: %w", obj, ErrInvalidTobikoType)
	}
	tobikolog.Info("Validation for Tobiko upon creation", "name", tobiko.GetName())

	// Call the validation function from api/v1beta1
	return tobiko.ValidateCreate()
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type Tobiko.
func (v *TobikoCustomValidator) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	tobiko, ok := newObj.(*testv1beta1.Tobiko)
	if !ok {
		return nil, fmt.Errorf("expected a Tobiko object for the newObj but got %T: %w", newObj, ErrInvalidTobikoType)
	}
	tobikolog.Info("Validation for Tobiko upon update", "name", tobiko.GetName())

	// Call the validation function from api/v1beta1
	return tobiko.ValidateUpdate(oldObj)
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type Tobiko.
func (v *TobikoCustomValidator) ValidateDelete(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	tobiko, ok := obj.(*testv1beta1.Tobiko)
	if !ok {
		return nil, fmt.Errorf("expected a Tobiko object but got %T: %w", obj, ErrInvalidTobikoType)
	}
	tobikolog.Info("Validation for Tobiko upon deletion", "name", tobiko.GetName())

	// Call the validation function from api/v1beta1
	return tobiko.ValidateDelete()
}
