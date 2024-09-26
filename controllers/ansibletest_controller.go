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
	"strconv"
	"time"

	"reflect"

	"github.com/go-logr/logr"
	"github.com/openstack-k8s-operators/lib-common/modules/common"
	"github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	"github.com/openstack-k8s-operators/lib-common/modules/common/env"
	"github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	"github.com/openstack-k8s-operators/lib-common/modules/common/job"
	common_rbac "github.com/openstack-k8s-operators/lib-common/modules/common/rbac"
	"github.com/openstack-k8s-operators/test-operator/api/v1beta1"
	testv1beta1 "github.com/openstack-k8s-operators/test-operator/api/v1beta1"
	"github.com/openstack-k8s-operators/test-operator/pkg/ansibletest"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
	Log := r.GetLogger(ctx)

	// How much time should we wait before calling Reconcile loop when there is a failure
	requeueAfter := time.Second * 60

	// Fetch the ansible instance
	instance := &testv1beta1.AnsibleTest{}
	err := r.Client.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if k8s_errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	workflowActive := false
	if len(instance.Spec.Workflow) > 0 {
		workflowActive = true
	}

	// Create a helper
	helper, err := helper.NewHelper(
		instance,
		r.Client,
		r.Kclient,
		r.Scheme,
		r.Log,
	)
	if err != nil {
		return ctrl.Result{}, err
	}

	// initialize status
	isNewInstance := instance.Status.Conditions == nil
	if isNewInstance {
		instance.Status.Conditions = condition.Conditions{}
	}

	// Save a copy of the condtions so that we can restore the LastTransitionTime
	// when a condition's state doesn't change.
	savedConditions := instance.Status.Conditions.DeepCopy()

	// Always patch the instance status when exiting this function so we
	// can persist any changes.
	defer func() {
		// update the overall status condition if service is ready
		if instance.Status.Conditions.AllSubConditionIsTrue() {
			instance.Status.Conditions.MarkTrue(condition.ReadyCondition, condition.ReadyMessage)
		}
		condition.RestoreLastTransitionTimes(&instance.Status.Conditions, savedConditions)
		if instance.Status.Conditions.IsUnknown(condition.ReadyCondition) {
			instance.Status.Conditions.Set(
				instance.Status.Conditions.Mirror(condition.ReadyCondition))
		}
		err := helper.PatchInstance(ctx, instance)
		if err != nil {
			_err = err
			return
		}
	}()

	if isNewInstance {
		// Initialize conditions used later as Status=Unknown
		cl := condition.CreateList(
			condition.UnknownCondition(condition.ReadyCondition, condition.InitReason, condition.ReadyInitMessage),
			condition.UnknownCondition(condition.ServiceConfigReadyCondition, condition.InitReason, condition.ServiceConfigReadyMessage),
			condition.UnknownCondition(condition.DeploymentReadyCondition, condition.InitReason, condition.DeploymentReadyInitMessage),
		)
		instance.Status.Conditions.Init(&cl)

		// Register overall status immediately to have an early feedback
		// e.g. in the cli
		return ctrl.Result{}, nil

	}

	// Ensure that there is an external counter and read its value
	// We use the external counter to keep track of the workflow steps
	r.WorkflowStepCounterCreate(ctx, instance, helper)
	externalWorkflowCounter := r.WorkflowStepCounterRead(ctx, instance)
	if externalWorkflowCounter == -1 {
		return ctrl.Result{RequeueAfter: requeueAfter}, nil
	}

	// Each job that is being executed by the test operator has
	currentWorkflowStep := 0
	runningAnsibleJob := &batchv1.Job{}
	runningJobName := r.GetJobName(instance, externalWorkflowCounter-1)
	err = r.Client.Get(ctx, client.ObjectKey{Namespace: instance.GetNamespace(), Name: runningJobName}, runningAnsibleJob)
	if err != nil && !k8s_errors.IsNotFound(err) {
		return ctrl.Result{}, err
	} else if err == nil {
		currentWorkflowStep, _ = strconv.Atoi(runningAnsibleJob.Labels["workflowStep"])
	}

	if r.CompletedJobExists(ctx, instance, currentWorkflowStep) {
		// The job created by the instance was completed. Release the lock
		// so that other instances can spawn a job.
		instance.Status.Conditions.MarkTrue(condition.DeploymentReadyCondition, condition.DeploymentReadyMessage)
		Log.Info("Job completed")
		if lockReleased, err := r.ReleaseLock(ctx, instance); !lockReleased {
			return ctrl.Result{}, err
		}
	}

	// Service account, role, binding
	rbacRules := []rbacv1.PolicyRule{
		{
			APIGroups:     []string{"security.openshift.io"},
			ResourceNames: []string{"anyuid", "privileged"},
			Resources:     []string{"securitycontextconstraints"},
			Verbs:         []string{"use"},
		},
		{
			APIGroups: []string{""},
			Resources: []string{"pods"},
			Verbs:     []string{"create", "get", "list", "watch", "update", "patch", "delete"},
		},
		{
			APIGroups: []string{""},
			Resources: []string{"persistentvolumeclaims"},
			Verbs:     []string{"get", "list", "create", "update", "watch", "patch"},
		},
	}
	rbacResult, err := common_rbac.ReconcileRbac(ctx, helper, instance, rbacRules)
	if err != nil {
		return rbacResult, err
	} else if (rbacResult != ctrl.Result{}) {
		return rbacResult, nil
	}

	// Service account, role, binding - end

	serviceLabels := map[string]string{
		common.AppSelector: ansibletest.ServiceName,
		"workflowStep":     strconv.Itoa(externalWorkflowCounter),
		"instanceName":     instance.Name,
		"operator":         "test-operator",
	}

	// Create PersistentVolumeClaim
	ctrlResult, err := r.EnsureLogsPVCExists(
		ctx,
		instance,
		helper,
		serviceLabels,
		instance.Spec.StorageClass,
		0,
	)
	if err != nil {
		return ctrlResult, err
	} else if (ctrlResult != ctrl.Result{}) {
		return ctrlResult, nil
	}
	// Create PersistentVolumeClaim - end

	// If the current job is executing the last workflow step -> do not create another job
	if workflowActive && externalWorkflowCounter >= len(instance.Spec.Workflow) {
		return ctrl.Result{}, nil
	} else if !workflowActive && r.JobExists(ctx, instance, currentWorkflowStep) {
		return ctrl.Result{}, nil
	}

	// We are about to start job that spawns the pod with tests.
	// This lock ensures that there is always only one pod running.
	lockAcquired, err := r.AcquireLock(ctx, instance, helper, false)
	if !lockAcquired {
		Log.Info("Can not acquire lock")
		requeueAfter := time.Second * 60
		return ctrl.Result{RequeueAfter: requeueAfter}, err
	}
	Log.Info("Lock acquired")

	if workflowActive {
		r.WorkflowStepCounterIncrease(ctx, instance, helper)
	}

	instance.Status.Conditions.MarkTrue(condition.ServiceConfigReadyCondition, condition.ServiceConfigReadyMessage)

	// Create a new job
	mountCerts := r.CheckSecretExists(ctx, instance, "combined-ca-bundle")
	jobName := r.GetJobName(instance, externalWorkflowCounter)
	envVars, workflowOverrideParams := r.PrepareAnsibleEnv(instance, externalWorkflowCounter)
	logsPVCName := r.GetPVCLogsName(instance, 0)
	containerImage, err := r.GetContainerImage(ctx, workflowOverrideParams["ContainerImage"], instance)
	privileged := r.OverwriteAnsibleWithWorkflow(instance.Spec, "Privileged", "pbool", externalWorkflowCounter).(bool)
	if err != nil {
		return ctrl.Result{}, err
	}

	if externalWorkflowCounter < len(instance.Spec.Workflow) {
		if instance.Spec.Workflow[externalWorkflowCounter].NodeSelector != nil {
			instance.Spec.NodeSelector = *instance.Spec.Workflow[externalWorkflowCounter].NodeSelector
		}

		if instance.Spec.Workflow[externalWorkflowCounter].Tolerations != nil {
			instance.Spec.Tolerations = *instance.Spec.Workflow[externalWorkflowCounter].Tolerations
		}
	}

	jobDef := ansibletest.Job(
		instance,
		serviceLabels,
		jobName,
		logsPVCName,
		mountCerts,
		envVars,
		workflowOverrideParams,
		externalWorkflowCounter,
		containerImage,
		privileged,
	)
	ansibleTestsJob := job.NewJob(
		jobDef,
		testv1beta1.ConfigHash,
		true,
		time.Duration(5)*time.Second,
		"",
	)

	ctrlResult, err = ansibleTestsJob.DoJob(ctx, helper)
	if err != nil {
		// Creation of the ansibleTests job was not successfull.
		// Release the lock and allow other controllers to spawn
		// a job.
		if lockReleased, err := r.ReleaseLock(ctx, instance); !lockReleased {
			return ctrl.Result{}, err
		}

		instance.Status.Conditions.Set(condition.FalseCondition(
			condition.DeploymentReadyCondition,
			condition.ErrorReason,
			condition.SeverityWarning,
			condition.DeploymentReadyErrorMessage,
			err.Error()))
		return ctrlResult, err
	} else if (ctrlResult != ctrl.Result{}) {
		instance.Status.Conditions.Set(condition.FalseCondition(
			condition.DeploymentReadyCondition,
			condition.RequestedReason,
			condition.SeverityInfo,
			condition.DeploymentReadyRunningMessage))
		return ctrlResult, nil
	}
	// Create a new job - end

	Log.Info("Reconciled Service successfully")
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
