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

package v1beta1

import (
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"github.com/openstack-k8s-operators/lib-common/modules/common/util"
)

const (
	TestTobikoImage = "quay.io/podified-antelope-centos9/openstack-tobiko:current-podified"
)

// log is for logging in this package.
var tobikolog = logf.Log.WithName("tobiko-resource")

func (r *Tobiko) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-test-openstack-org-v1beta1-tobiko,mutating=true,failurePolicy=fail,sideEffects=None,groups=test.openstack.org,resources=tobikoes,verbs=create;update,versions=v1beta1,name=mtobiko.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &Tobiko{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *Tobiko) Default() {
	tobikolog.Info("default", "name", r.Name)

	if len(r.Spec.ContainerImage) == 0 {
		r.Spec.ContainerImage = util.GetEnvVar("RELATED_IMAGE_TEST_TOBIKO_IMAGE_URL_DEFAULT", TestTobikoImage)
	}
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-test-openstack-org-v1beta1-tobiko,mutating=false,failurePolicy=fail,sideEffects=None,groups=test.openstack.org,resources=tobikoes,verbs=create;update,versions=v1beta1,name=vtobiko.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &Tobiko{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Tobiko) ValidateCreate() (admission.Warnings, error) {
	tobikolog.Info("validate create", "name", r.Name)

	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Tobiko) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	tobikolog.Info("validate update", "name", r.Name)

	// TODO(user): fill in your validation logic upon object update.
	return nil, nil 
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Tobiko) ValidateDelete() (admission.Warnings, error) {
	tobikolog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil, nil
}
