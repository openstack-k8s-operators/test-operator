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

package controllers

import (
	"context"

	"reflect"

	"github.com/go-logr/logr"
	"github.com/openstack-k8s-operators/lib-common/modules/common/env"
	"github.com/openstack-k8s-operators/test-operator/api/v1beta1"
	testv1beta1 "github.com/openstack-k8s-operators/test-operator/api/v1beta1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type AnsibleTestReconciler struct {
	Reconciler
}

// GetLogger returns a logger object with a prefix of "controller.name" and additional controller context fields
func (r *AnsibleTestReconciler) GetLogger(ctx context.Context) logr.Logger {
	return log.FromContext(ctx).WithName("Controllers").WithName("Tobiko")
}

// +kubebuilder:rbac:groups=test.openstack.org,resources=ansibletests,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=test.openstack.org,resources=ansibletests/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=test.openstack.org,resources=ansibletests/finalizers,verbs=update;patch
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete;
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete;
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;patch;update;delete;
// +kubebuilder:rbac:groups=k8s.cni.cncf.io,resources=network-attachment-definitions,verbs=get;list;watch

// service account, role, rolebinding
// +kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=roles,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=rolebindings,verbs=get;list;watch;create;update;patch
// service account permissions that are needed to grant permission to the above
// +kubebuilder:rbac:groups="security.openshift.io",resourceNames=anyuid;privileged,resources=securitycontextconstraints,verbs=use
// +kubebuilder:rbac:groups="",resources=pods,verbs=create;delete;get;list;patch;update;watch
// +kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;create;update;watch;patch

// Reconcile - AnsibleTestReconciler
func (r *AnsibleTestReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, _err error) {
	// TODO(lpiwowar)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AnsibleTestReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&testv1beta1.AnsibleTest{}).
		Owns(&batchv1.Job{}).
		Owns(&corev1.Secret{}).
		Owns(&corev1.ConfigMap{}).
		Complete(r)
}

func (r *Reconciler) OverwriteAnsibleWithWorkflow(
	instance v1beta1.AnsibleTestSpec,
	sectionName string,
	workflowValueType string,
	workflowStepNum int,
) interface{} {
	if len(instance.Workflow)-1 < workflowStepNum {
		reflected := reflect.ValueOf(instance)
		fieldValue := reflected.FieldByName(sectionName)
		return fieldValue.Interface()
	}

	reflected := reflect.ValueOf(instance)
	SpecValue := reflected.FieldByName(sectionName).Interface()

	reflected = reflect.ValueOf(instance.Workflow[workflowStepNum])
	WorkflowValue := reflected.FieldByName(sectionName).Interface()

	if workflowValueType == "pbool" {
		if val, ok := WorkflowValue.(*bool); ok && val != nil {
			return *(WorkflowValue.(*bool))
		}
		return SpecValue.(bool)
	} else if workflowValueType == "puint8" {
		if val, ok := WorkflowValue.(*uint8); ok && val != nil {
			return *(WorkflowValue.(*uint8))
		}
		return SpecValue
	} else if workflowValueType == "string" {
		if val, ok := WorkflowValue.(string); ok && val != "" {
			return WorkflowValue
		}
		return SpecValue
	}

	return nil
}

// This function prepares env variables for a single workflow step.
func (r *AnsibleTestReconciler) PrepareAnsibleEnv(
	instance *testv1beta1.AnsibleTest,
	step int,
) (map[string]env.Setter, map[string]string) {
	// Prepare env vars
	envVars := make(map[string]env.Setter)
	workflowOverrideParams := make(map[string]string)

	// volumes workflow override
	workflowOverrideParams["WorkloadSSHKeySecretName"] = r.OverwriteAnsibleWithWorkflow(instance.Spec, "WorkloadSSHKeySecretName", "string", step).(string)
	workflowOverrideParams["ComputesSSHKeySecretName"] = r.OverwriteAnsibleWithWorkflow(instance.Spec, "ComputesSSHKeySecretName", "string", step).(string)
	workflowOverrideParams["ContainerImage"] = r.OverwriteAnsibleWithWorkflow(instance.Spec, "ContainerImage", "string", step).(string)

	// bool
	debug := r.OverwriteAnsibleWithWorkflow(instance.Spec, "Debug", "pbool", step).(bool)
	if debug {
		envVars["POD_DEBUG"] = env.SetValue("true")
	}

	// strings
	extraVars := r.OverwriteAnsibleWithWorkflow(instance.Spec, "AnsibleExtraVars", "string", step).(string)
	envVars["POD_ANSIBLE_EXTRA_VARS"] = env.SetValue(extraVars)

	extraVarsFile := r.OverwriteAnsibleWithWorkflow(instance.Spec, "AnsibleVarFiles", "string", step).(string)
	envVars["POD_ANSIBLE_FILE_EXTRA_VARS"] = env.SetValue(extraVarsFile)

	inventory := r.OverwriteAnsibleWithWorkflow(instance.Spec, "AnsibleInventory", "string", step).(string)
	envVars["POD_ANSIBLE_INVENTORY"] = env.SetValue(inventory)

	gitRepo := r.OverwriteAnsibleWithWorkflow(instance.Spec, "AnsibleGitRepo", "string", step).(string)
	envVars["POD_ANSIBLE_GIT_REPO"] = env.SetValue(gitRepo)

	playbookPath := r.OverwriteAnsibleWithWorkflow(instance.Spec, "AnsiblePlaybookPath", "string", step).(string)
	envVars["POD_ANSIBLE_PLAYBOOK"] = env.SetValue(playbookPath)

	ansibleCollections := r.OverwriteAnsibleWithWorkflow(instance.Spec, "AnsibleCollections", "string", step).(string)
	envVars["POD_INSTALL_COLLECTIONS"] = env.SetValue(ansibleCollections)

	return envVars, workflowOverrideParams
}
