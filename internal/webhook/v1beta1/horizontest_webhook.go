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
	// ErrInvalidHorizonTestType is returned when an unexpected object type is passed to the webhook
	ErrInvalidHorizonTestType = errors.New("invalid object type for HorizonTest webhook")
)

// horizontestlog is for logging in this package.
var horizontestlog = logf.Log.WithName("horizontest-resource")

// SetupHorizonTestWebhookWithManager registers the webhook for HorizonTest in the manager.
func SetupHorizonTestWebhookWithManager(mgr ctrl.Manager) error {
	// Set up webhookClient for API webhook functions
	testv1beta1.SetupWebhookClient(mgr.GetClient())

	return ctrl.NewWebhookManagedBy(mgr).For(&testv1beta1.HorizonTest{}).
		WithValidator(&HorizonTestCustomValidator{}).
		WithDefaulter(&HorizonTestCustomDefaulter{}).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-test-openstack-org-v1beta1-horizontest,mutating=true,failurePolicy=fail,sideEffects=None,groups=test.openstack.org,resources=horizontests,verbs=create;update,versions=v1beta1,name=mhorizontest-v1beta1.kb.io,admissionReviewVersions=v1

// HorizonTestCustomDefaulter struct is responsible for setting default values on the custom resource of the
// Kind HorizonTest when those are created or updated.
type HorizonTestCustomDefaulter struct {
}

var _ webhook.CustomDefaulter = &HorizonTestCustomDefaulter{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind HorizonTest.
func (d *HorizonTestCustomDefaulter) Default(_ context.Context, obj runtime.Object) error {
	horizontest, ok := obj.(*testv1beta1.HorizonTest)

	if !ok {
		return fmt.Errorf("expected an HorizonTest object but got %T: %w", obj, ErrInvalidHorizonTestType)
	}
	horizontestlog.Info("Defaulting for HorizonTest", "name", horizontest.GetName())

	// Call the default function from api/v1beta1
	horizontest.Default()

	return nil
}

// +kubebuilder:webhook:path=/validate-test-openstack-org-v1beta1-horizontest,mutating=false,failurePolicy=fail,sideEffects=None,groups=test.openstack.org,resources=horizontests,verbs=create;update,versions=v1beta1,name=vhorizontest-v1beta1.kb.io,admissionReviewVersions=v1

// HorizonTestCustomValidator struct is responsible for validating the HorizonTest resource
// when it is created, updated, or deleted.
type HorizonTestCustomValidator struct {
}

var _ webhook.CustomValidator = &HorizonTestCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type HorizonTest.
func (v *HorizonTestCustomValidator) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	horizontest, ok := obj.(*testv1beta1.HorizonTest)
	if !ok {
		return nil, fmt.Errorf("expected a HorizonTest object but got %T: %w", obj, ErrInvalidHorizonTestType)
	}
	horizontestlog.Info("Validation for HorizonTest upon creation", "name", horizontest.GetName())

	// Call the validation function from api/v1beta1
	return horizontest.ValidateCreate()
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type HorizonTest.
func (v *HorizonTestCustomValidator) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	horizontest, ok := newObj.(*testv1beta1.HorizonTest)
	if !ok {
		return nil, fmt.Errorf("expected a HorizonTest object for the newObj but got %T: %w", newObj, ErrInvalidHorizonTestType)
	}
	horizontestlog.Info("Validation for HorizonTest upon update", "name", horizontest.GetName())

	// Call the validation function from api/v1beta1
	return horizontest.ValidateUpdate(oldObj)
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type HorizonTest.
func (v *HorizonTestCustomValidator) ValidateDelete(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	horizontest, ok := obj.(*testv1beta1.HorizonTest)
	if !ok {
		return nil, fmt.Errorf("expected a HorizonTest object but got %T: %w", obj, ErrInvalidHorizonTestType)
	}
	horizontestlog.Info("Validation for HorizonTest upon deletion", "name", horizontest.GetName())

	// Call the validation function from api/v1beta1
	return horizontest.ValidateDelete()
}
