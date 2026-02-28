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

// Package controller implements the Kubernetes controllers for managing test framework operations
package controller

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	"github.com/openstack-k8s-operators/lib-common/modules/common/env"
	testv1beta1 "github.com/openstack-k8s-operators/test-operator/api/v1beta1"
	"github.com/openstack-k8s-operators/test-operator/internal/ansibletest"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// AnsibleTestReconciler reconciles an AnsibleTest object
type AnsibleTestReconciler struct {
	Reconciler
}

// GetLogger returns a logger object with a prefix of "controller.name" and additional controller context fields
func (r *AnsibleTestReconciler) GetLogger(ctx context.Context) logr.Logger {
	return log.FromContext(ctx).WithName("Controllers").WithName("AnsibleTest")
}

// +kubebuilder:rbac:groups=test.openstack.org,resources=ansibletests,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=test.openstack.org,resources=ansibletests/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=test.openstack.org,resources=ansibletests/finalizers,verbs=update;patch
// +kubebuilder:rbac:groups=k8s.cni.cncf.io,resources=network-attachment-definitions,verbs=get;list;watch
// +kubebuilder:rbac:groups="security.openshift.io",resourceNames=anyuid;privileged;nonroot;nonroot-v2,resources=securitycontextconstraints,verbs=use
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete;
// +kubebuilder:rbac:groups="",resources=pods,verbs=create;delete;get;list;patch;update;watch
// +kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;create;update;watch;patch;delete

// Reconcile - AnsibleTest
func (r *AnsibleTestReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	instance := &testv1beta1.AnsibleTest{}

	config := FrameworkConfig[*testv1beta1.AnsibleTest]{
		ServiceName:             ansibletest.ServiceName,
		NeedsNetworkAttachments: false,
		NeedsConfigMaps:         false,
		NeedsFinalizer:          false,
		SupportsWorkflow:        true,

		BuildPod: func(ctx context.Context, instance *testv1beta1.AnsibleTest, labels, annotations map[string]string, workflowStepNum int, pvcIndex int) (*corev1.Pod, error) {
			return r.buildAnsibleTestPod(ctx, instance, labels, annotations, workflowStepNum, pvcIndex)
		},

		GetInitialConditions: func() []*condition.Condition {
			return []*condition.Condition{
				condition.UnknownCondition(condition.ReadyCondition, condition.InitReason, condition.ReadyInitMessage),
				condition.UnknownCondition(condition.InputReadyCondition, condition.InitReason, condition.InputReadyInitMessage),
				condition.UnknownCondition(condition.DeploymentReadyCondition, condition.InitReason, condition.DeploymentReadyInitMessage),
			}
		},

		ValidateInputs: func(ctx context.Context, instance *testv1beta1.AnsibleTest) error {
			return r.ValidateOpenstackInputs(ctx, instance, instance.Spec.OpenStackConfigMap, instance.Spec.OpenStackConfigSecret)
		},

		GetSpec: func(instance *testv1beta1.AnsibleTest) interface{} {
			return &instance.Spec
		},

		GetWorkflowStep: func(instance *testv1beta1.AnsibleTest, step int) interface{} {
			return instance.Spec.Workflow[step]
		},

		GetWorkflowLength: func(instance *testv1beta1.AnsibleTest) int {
			return len(instance.Spec.Workflow)
		},

		GetStorageClass: func(instance *testv1beta1.AnsibleTest) string {
			return instance.Spec.StorageClass
		},

		SetObservedGeneration: func(instance *testv1beta1.AnsibleTest) {
			instance.Status.ObservedGeneration = instance.Generation
		},
	}

	return CommonReconcile(ctx, &r.Reconciler, req, instance, config, r.GetLogger(ctx))
}

func (r *AnsibleTestReconciler) buildAnsibleTestPod(
	ctx context.Context,
	instance *testv1beta1.AnsibleTest,
	labels, _ map[string]string,
	workflowStepNum int,
	pvcIndex int,
) (*corev1.Pod, error) {
	mountCerts := r.CheckSecretExists(ctx, instance, "combined-ca-bundle")
	envVars := r.PrepareAnsibleEnv(instance)

	podName := r.GetPodName(instance, workflowStepNum)
	logsPVCName := r.GetPVCLogsName(instance, pvcIndex)

	containerImage, err := r.GetContainerImage(ctx, instance)
	if err != nil {
		return nil, err
	}

	return ansibletest.Pod(
		instance,
		labels,
		podName,
		logsPVCName,
		mountCerts,
		envVars,
		workflowStepNum,
		containerImage,
	), nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AnsibleTestReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&testv1beta1.AnsibleTest{}).
		Owns(&corev1.Pod{}).
		Owns(&corev1.Secret{}).
		Owns(&corev1.ConfigMap{}).
		Complete(r)
}

// PrepareAnsibleEnv prepares environment variables for a single workflow step
func (r *AnsibleTestReconciler) PrepareAnsibleEnv(
	instance *testv1beta1.AnsibleTest,
) map[string]env.Setter {
	// Prepare env vars
	envVars := make(map[string]env.Setter)

	// Bool
	SetBoolEnvVars(envVars, map[string]bool{
		"POD_DEBUG": instance.Spec.Debug,
	})

	// Strings
	SetStringEnvVars(envVars, map[string]string{
		"POD_ANSIBLE_EXTRA_VARS":      instance.Spec.AnsibleExtraVars,
		"POD_ANSIBLE_FILE_EXTRA_VARS": instance.Spec.AnsibleVarFiles,
		"POD_ANSIBLE_INVENTORY":       instance.Spec.AnsibleInventory,
		"POD_ANSIBLE_GIT_REPO":        instance.Spec.AnsibleGitRepo,
		"POD_ANSIBLE_GIT_BRANCH":      instance.Spec.AnsibleGitBranch,
		"POD_ANSIBLE_PLAYBOOK":        instance.Spec.AnsiblePlaybookPath,
		"POD_INSTALL_COLLECTIONS":     instance.Spec.AnsibleCollections,
	})

	return envVars
}
