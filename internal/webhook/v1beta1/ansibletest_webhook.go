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

// Package v1beta1 contains webhook implementations for v1beta1 API resources
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
	// ErrInvalidAnsibleTestType is returned when an unexpected object type is passed to the webhook
	ErrInvalidAnsibleTestType = errors.New("invalid object type for AnsibleTest webhook")
)

// nolint:unused
// log is for logging in this package.
var ansibletestlog = logf.Log.WithName("ansibletest-resource")

// SetupAnsibleTestWebhookWithManager registers the webhook for AnsibleTest in the manager.
func SetupAnsibleTestWebhookWithManager(mgr ctrl.Manager) error {
	// Set up webhookClient for API webhook functions
	testv1beta1.SetupWebhookClient(mgr.GetClient())

	return ctrl.NewWebhookManagedBy(mgr).For(&testv1beta1.AnsibleTest{}).
		WithValidator(&AnsibleTestCustomValidator{}).
		WithDefaulter(&AnsibleTestCustomDefaulter{}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate-test-openstack-org-v1beta1-ansibletest,mutating=true,failurePolicy=fail,sideEffects=None,groups=test.openstack.org,resources=ansibletests,verbs=create;update,versions=v1beta1,name=mansibletest-v1beta1.kb.io,admissionReviewVersions=v1

// AnsibleTestCustomDefaulter struct is responsible for setting default values on the custom resource of the
// Kind AnsibleTest when those are created or updated.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as it is used only for temporary operations and does not need to be deeply copied.
type AnsibleTestCustomDefaulter struct {
	// TODO(user): Add more fields as needed for defaulting
}

var _ webhook.CustomDefaulter = &AnsibleTestCustomDefaulter{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind AnsibleTest.
func (d *AnsibleTestCustomDefaulter) Default(_ context.Context, obj runtime.Object) error {
	ansibletest, ok := obj.(*testv1beta1.AnsibleTest)

	if !ok {
		return fmt.Errorf("expected an AnsibleTest object but got %T: %w", obj, ErrInvalidAnsibleTestType)
	}
	ansibletestlog.Info("Defaulting for AnsibleTest", "name", ansibletest.GetName())

	// Call the default function from api/v1beta1
	ansibletest.Default()

	return nil
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-test-openstack-org-v1beta1-ansibletest,mutating=false,failurePolicy=fail,sideEffects=None,groups=test.openstack.org,resources=ansibletests,verbs=create;update,versions=v1beta1,name=vansibletest-v1beta1.kb.io,admissionReviewVersions=v1

// AnsibleTestCustomValidator struct is responsible for validating the AnsibleTest resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type AnsibleTestCustomValidator struct {
	// TODO(user): Add more fields as needed for validation
}

var _ webhook.CustomValidator = &AnsibleTestCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type AnsibleTest.
func (v *AnsibleTestCustomValidator) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	ansibletest, ok := obj.(*testv1beta1.AnsibleTest)
	if !ok {
		return nil, fmt.Errorf("expected a AnsibleTest object but got %T: %w", obj, ErrInvalidAnsibleTestType)
	}
	ansibletestlog.Info("Validation for AnsibleTest upon creation", "name", ansibletest.GetName())

	// Call the validation function from api/v1beta1
	return ansibletest.ValidateCreate()
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type AnsibleTest.
func (v *AnsibleTestCustomValidator) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	ansibletest, ok := newObj.(*testv1beta1.AnsibleTest)
	if !ok {
		return nil, fmt.Errorf("expected a AnsibleTest object for the newObj but got %T: %w", newObj, ErrInvalidAnsibleTestType)
	}
	ansibletestlog.Info("Validation for AnsibleTest upon update", "name", ansibletest.GetName())

	// Call the validation function from api/v1beta1
	return ansibletest.ValidateUpdate(oldObj)
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type AnsibleTest.
func (v *AnsibleTestCustomValidator) ValidateDelete(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	ansibletest, ok := obj.(*testv1beta1.AnsibleTest)
	if !ok {
		return nil, fmt.Errorf("expected a AnsibleTest object but got %T: %w", obj, ErrInvalidAnsibleTestType)
	}
	ansibletestlog.Info("Validation for AnsibleTest upon deletion", "name", ansibletest.GetName())

	// Call the validation function from api/v1beta1
	return ansibletest.ValidateDelete()
}
