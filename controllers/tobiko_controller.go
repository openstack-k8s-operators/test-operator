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
	"fmt"
	"strconv"
	"time"

	"github.com/go-logr/logr"
	"github.com/openstack-k8s-operators/lib-common/modules/common"
	"github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	"github.com/openstack-k8s-operators/lib-common/modules/common/configmap"
	"github.com/openstack-k8s-operators/lib-common/modules/common/env"
	"github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	"github.com/openstack-k8s-operators/lib-common/modules/common/job"
	nad "github.com/openstack-k8s-operators/lib-common/modules/common/networkattachment"
	common_rbac "github.com/openstack-k8s-operators/lib-common/modules/common/rbac"
	"github.com/openstack-k8s-operators/lib-common/modules/common/util"
	testv1beta1 "github.com/openstack-k8s-operators/test-operator/api/v1beta1"
	"github.com/openstack-k8s-operators/test-operator/pkg/tobiko"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// TobikoReconciler reconciles a Tobiko object
type TobikoReconciler struct {
	Reconciler
}

// GetLogger returns a logger object with a prefix of "controller.name" and additional controller context fields
func (r *TobikoReconciler) GetLogger(ctx context.Context) logr.Logger {
	return log.FromContext(ctx).WithName("Controllers").WithName("Tobiko")
}

//+kubebuilder:rbac:groups=test.openstack.org,resources=tobikoes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=test.openstack.org,resources=tobikoes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=test.openstack.org,resources=tobikoes/finalizers,verbs=update;patch
//+kubebuilder:rbac:groups="",resources=pods,verbs=create;delete;get;list;patch;update;watch
//+kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;create;update;watch;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *TobikoReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, _err error) {
	Log := r.GetLogger(ctx)

	// How much time should we wait before calling Reconcile loop when there is a failure
	requeueAfter := time.Second * 60

	instance := &testv1beta1.Tobiko{}
	err := r.Client.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if k8s_errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Check whether the user wants to execute workflow
	workflowActive := false
	if len(instance.Spec.Workflow) > 0 {
		workflowActive = true
	}

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
			condition.UnknownCondition(condition.DeploymentReadyCondition, condition.InitReason, condition.DeploymentReadyInitMessage),
			condition.UnknownCondition(condition.NetworkAttachmentsReadyCondition, condition.InitReason, condition.NetworkAttachmentsReadyInitMessage),
		)
		instance.Status.Conditions.Init(&cl)

		// Register overall status immediately to have an early feedback
		// e.g. in the cli
		return ctrl.Result{}, nil

	}

	if instance.Status.NetworkAttachments == nil {
		instance.Status.NetworkAttachments = map[string][]string{}
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
	runningTobikoJob := &batchv1.Job{}
	runningJobName := r.GetJobName(instance, externalWorkflowCounter-1)
	err = r.Client.Get(ctx, client.ObjectKey{Namespace: instance.GetNamespace(), Name: runningJobName}, runningTobikoJob)
	if err == nil {
		currentWorkflowStep, _ = strconv.Atoi(runningTobikoJob.Labels["workflowStep"])
	}

	if r.CompletedJobExists(ctx, instance, currentWorkflowStep) {
		instance.Status.Conditions.MarkTrue(condition.DeploymentReadyCondition, condition.DeploymentReadyMessage)
		// The job created by the instance was completed. Release the lock
		// so that other instances can spawn a job.
		Log.Info("Job completed")
		if lockReleased, err := r.ReleaseLock(ctx, instance); !lockReleased {
			return ctrl.Result{}, err
		}
	}

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
	}

	rbacResult, err := common_rbac.ReconcileRbac(ctx, helper, instance, rbacRules)
	if err != nil {
		return rbacResult, err
	} else if (rbacResult != ctrl.Result{}) {
		return rbacResult, nil
	}

	serviceLabels := map[string]string{
		common.AppSelector: tobiko.ServiceName,
		"workflowStep":     strconv.Itoa(externalWorkflowCounter),
		"instanceName":     instance.Name,
		"operator":         "test-operator",
	}

	yamlResult, err := EnsureCloudsConfigMapExists(ctx, instance, helper, serviceLabels)

	if err != nil {
		return yamlResult, err
	}

	workflowStepNum := 0

	// Create multiple PVCs for parallel execution
	if instance.Spec.Parallel && externalWorkflowCounter < len(instance.Spec.Workflow) {
		workflowStepNum = externalWorkflowCounter
	}

	// Create PersistentVolumeClaim
	ctrlResult, err := r.EnsureLogsPVCExists(
		ctx,
		instance,
		helper,
		serviceLabels,
		instance.Spec.StorageClass,
		workflowStepNum,
	)
	if err != nil {
		return ctrlResult, err
	} else if (ctrlResult != ctrl.Result{}) {
		return ctrlResult, nil
	}
	// Create PersistentVolumeClaim - end

	serviceAnnotations, err := nad.CreateNetworksAnnotation(instance.Namespace, instance.Spec.NetworkAttachments)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed create network annotation from %s: %w",
			instance.Spec.NetworkAttachments, err)
	}

	// NetworkAttachments
	if r.JobExists(ctx, instance, externalWorkflowCounter) {
		networkReady, networkAttachmentStatus, err := nad.VerifyNetworkStatusFromAnnotation(
			ctx,
			helper,
			instance.Spec.NetworkAttachments,
			serviceLabels,
			1,
		)
		if err != nil {
			return ctrl.Result{}, err
		}

		instance.Status.NetworkAttachments = networkAttachmentStatus

		if networkReady {
			instance.Status.Conditions.MarkTrue(
				condition.NetworkAttachmentsReadyCondition,
				condition.NetworkAttachmentsReadyMessage)
		} else {
			err := fmt.Errorf(ErrNetworkAttachments, instance.Spec.NetworkAttachments)
			instance.Status.Conditions.Set(condition.FalseCondition(
				condition.NetworkAttachmentsReadyCondition,
				condition.ErrorReason,
				condition.SeverityWarning,
				condition.NetworkAttachmentsReadyErrorMessage,
				err.Error()))

			return ctrl.Result{}, err
		}
	}
	// NetworkAttachments - end

	// Create Job
	mountCerts := r.CheckSecretExists(ctx, instance, "combined-ca-bundle")

	mountKeys := false
	if (len(instance.Spec.PublicKey) == 0) || (len(instance.Spec.PrivateKey) == 0) {
		Log.Info("Both values privateKey and publicKey need to be specified. Keys not mounted.")
	} else {
		mountKeys = true
	}

	mountKubeconfig := false
	if len(instance.Spec.KubeconfigSecretName) != 0 {
		mountKubeconfig = true
	}

	// If the current job is executing the last workflow step -> do not create another job
	if workflowActive && externalWorkflowCounter >= len(instance.Spec.Workflow) {
		return ctrl.Result{}, nil
	} else if !workflowActive && r.JobExists(ctx, instance, currentWorkflowStep) {
		return ctrl.Result{}, nil
	}

	// We are about to start job that spawns the pod with tests.
	// This lock ensures that there is always only one pod running.
	lockAcquired, err := r.AcquireLock(ctx, instance, helper, instance.Spec.Parallel)
	if !lockAcquired {
		Log.Info("Can not acquire lock")
		return ctrl.Result{RequeueAfter: requeueAfter}, err
	}
	Log.Info("Lock acquired")

	if workflowActive {
		r.WorkflowStepCounterIncrease(ctx, instance, helper)
	}

	// Prepare Tobiko env vars
	envVars := r.PrepareTobikoEnvVars(ctx, serviceLabels, instance, helper, externalWorkflowCounter)
	jobName := r.GetJobName(instance, externalWorkflowCounter)
	logsPVCName := r.GetPVCLogsName(instance, workflowStepNum)
	containerImage, err := r.GetContainerImage(ctx, instance.Spec.ContainerImage, instance)
	privileged := r.OverwriteValueWithWorkflow(instance.Spec, "Privileged", "pbool", externalWorkflowCounter).(bool)
	if err != nil {
		return ctrl.Result{}, err
	}

	jobDef := tobiko.Job(
		instance,
		serviceLabels,
		serviceAnnotations,
		jobName,
		logsPVCName,
		mountCerts,
		mountKeys,
		mountKubeconfig,
		envVars,
		containerImage,
		privileged,
	)
	tobikoJob := job.NewJob(
		jobDef,
		testv1beta1.ConfigHash,
		true,
		time.Duration(5)*time.Second,
		"",
	)

	ctrlResult, err = tobikoJob.DoJob(ctx, helper)
	if err != nil {
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
	// create Job - end

	Log.Info("Reconciled Service successfully")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *TobikoReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&testv1beta1.Tobiko{}).
		Owns(&batchv1.Job{}).
		Owns(&corev1.Secret{}).
		Owns(&corev1.ConfigMap{}).
		Complete(r)
}

// This function prepares env variables for a single workflow step.
func (r *TobikoReconciler) PrepareTobikoEnvVars(
	ctx context.Context,
	labels map[string]string,
	instance *testv1beta1.Tobiko,
	helper *helper.Helper,
	step int,
) map[string]env.Setter {

	// NOTE(lpiwowar): Move all the merge code to the webhook once it is completed.
	//                 It will clean up the workflow code and remove the duplicit code
	//                 (Tempest vs Tobiko)
	if step < len(instance.Spec.Workflow) {
		if instance.Spec.Workflow[step].NodeSelector != nil {
			instance.Spec.NodeSelector = *instance.Spec.Workflow[step].NodeSelector
		}

		if instance.Spec.Workflow[step].Tolerations != nil {
			instance.Spec.Tolerations = *instance.Spec.Workflow[step].Tolerations
		}
	}

	// Prepare env vars
	envVars := make(map[string]env.Setter)
	envVars["USE_EXTERNAL_FILES"] = env.SetValue("True")
	envVars["TOBIKO_LOGS_DIR_NAME"] = env.SetValue(r.GetJobName(instance, step))

	testenv := r.OverwriteValueWithWorkflow(instance.Spec, "Testenv", "string", step).(string)
	envVars["TOBIKO_TESTENV"] = env.SetValue(testenv)

	version := r.OverwriteValueWithWorkflow(instance.Spec, "Version", "string", step).(string)
	envVars["TOBIKO_VERSION"] = env.SetValue(version)

	pytestAddopts := r.OverwriteValueWithWorkflow(instance.Spec, "PytestAddopts", "string", step).(string)
	envVars["TOBIKO_PYTEST_ADDOPTS"] = env.SetValue(pytestAddopts)

	preventCreate := r.OverwriteValueWithWorkflow(instance.Spec, "PreventCreate", "pbool", step).(bool)
	if preventCreate {
		envVars["TOBIKO_PREVENT_CREATE"] = env.SetValue("True")
	}

	numProcesses := r.OverwriteValueWithWorkflow(instance.Spec, "NumProcesses", "puint8", step).(uint8)
	if numProcesses > 0 {
		envVars["TOX_NUM_PROCESSES"] = env.SetValue(strconv.Itoa(int(numProcesses)))
	}

	envVars["TOBIKO_KEYS_FOLDER"] = env.SetValue("/etc/test_operator")
	envVars["TOBIKO_DEBUG_MODE"] = env.SetValue(r.GetDefaultBool(instance.Spec.Debug))
	// Prepare env vars - end

	// Prepare custom data
	customData := make(map[string]string)
	tobikoConf := r.OverwriteValueWithWorkflow(instance.Spec, "Config", "string", step).(string)
	customData["tobiko.conf"] = tobikoConf

	privateKeyData := make(map[string]string)
	privateKey := r.OverwriteValueWithWorkflow(instance.Spec, "PrivateKey", "string", step).(string)
	privateKeyData["id_ecdsa"] = privateKey

	publicKeyData := make(map[string]string)
	publicKey := r.OverwriteValueWithWorkflow(instance.Spec, "PublicKey", "string", step).(string)
	publicKeyData["id_ecdsa.pub"] = publicKey

	cms := []util.Template{
		{
			Name:         instance.Name + "tobiko-config",
			Namespace:    instance.Namespace,
			InstanceType: instance.Kind,
			Labels:       labels,
			CustomData:   customData,
		},
		{
			Name:         instance.Name + "tobiko-private-key",
			Namespace:    instance.Namespace,
			InstanceType: instance.Kind,
			Labels:       labels,
			CustomData:   privateKeyData,
		},
		{
			Name:         instance.Name + "tobiko-public-key",
			Namespace:    instance.Namespace,
			InstanceType: instance.Kind,
			Labels:       labels,
			CustomData:   publicKeyData,
		},
	}

	err := configmap.EnsureConfigMaps(ctx, helper, instance, cms, nil)
	if err != nil {
		return map[string]env.Setter{}
	}

	return envVars
}
