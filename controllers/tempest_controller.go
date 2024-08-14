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
	"github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	"github.com/openstack-k8s-operators/lib-common/modules/common/job"
	"github.com/openstack-k8s-operators/lib-common/modules/common/labels"
	nad "github.com/openstack-k8s-operators/lib-common/modules/common/networkattachment"
	common_rbac "github.com/openstack-k8s-operators/lib-common/modules/common/rbac"
	"github.com/openstack-k8s-operators/lib-common/modules/common/util"
	testv1beta1 "github.com/openstack-k8s-operators/test-operator/api/v1beta1"
	"github.com/openstack-k8s-operators/test-operator/pkg/tempest"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type TempestReconciler struct {
	Reconciler
}

// GetLogger returns a logger object with a prefix of "controller.name" and additional controller context fields
func (r *TempestReconciler) GetLogger(ctx context.Context) logr.Logger {
	return log.FromContext(ctx).WithName("Controllers").WithName("Tempest")
}

// +kubebuilder:rbac:groups=test.openstack.org,resources=tempests,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=test.openstack.org,resources=tempests/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=test.openstack.org,resources=tempests/finalizers,verbs=update;patch
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

// Reconcile - Tempest
func (r *TempestReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, _err error) {
	Log := r.GetLogger(ctx)

	// How much time should we wait before calling Reconcile loop when there is a failure
	requeueAfter := time.Second * 60

	// Fetch the Tempest instance
	instance := &testv1beta1.Tempest{}
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

	// Ensure that there is an external counter and read its value
	// We use the external counter to keep track of the workflow steps
	r.WorkflowStepCounterCreate(ctx, instance, helper)
	externalWorkflowCounter := r.WorkflowStepCounterRead(ctx, instance, helper)
	if externalWorkflowCounter == -1 {
		return ctrl.Result{RequeueAfter: requeueAfter}, nil
	}

	// Each job that is being executed by the test operator has
	currentWorkflowStep := 0
	runningTobikoJob := &batchv1.Job{}
	runningJobName := r.GetJobName(instance, externalWorkflowCounter-1)
	err = r.Client.Get(ctx, client.ObjectKey{Namespace: instance.GetNamespace(), Name: runningJobName}, runningTobikoJob)
	if err == nil {
		currentWorkflowStep, err = strconv.Atoi(runningTobikoJob.Labels["workflowStep"])
	}

	if r.CompletedJobExists(ctx, instance, currentWorkflowStep) {
		// The job created by the instance was completed. Release the lock
		// so that other instances can spawn a job.
		Log.Info("Job completed")
		if lockReleased, err := r.ReleaseLock(ctx, instance); !lockReleased {
			return ctrl.Result{}, err
		}
	}

	// Always patch the instance status when exiting this function so we
	// can persist any changes.
	defer func() {
		err := helper.PatchInstance(ctx, instance)
		if err != nil {
			_err = err
			return
		}
	}()

	// If we're not deleting this and the service object doesn't have our
	// finalizer, add it.
	if instance.DeletionTimestamp.IsZero() && controllerutil.AddFinalizer(instance, helper.GetFinalizer()) {
		return ctrl.Result{}, nil
	}

	// Initialize conditions used later as Status=Unknown
	if instance.Status.Conditions == nil {
		instance.Status.Conditions = condition.Conditions{}
		cl := condition.CreateList(
			condition.UnknownCondition(condition.InputReadyCondition, condition.InitReason, condition.InputReadyInitMessage),
			condition.UnknownCondition(condition.ServiceConfigReadyCondition, condition.InitReason, condition.ServiceConfigReadyInitMessage),
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

	// Handle service delete
	if !instance.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, instance, helper)
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
	instance.Status.Conditions.MarkTrue(condition.InputReadyCondition, condition.InputReadyMessage)
	// Service account, role, binding - end

	serviceLabels := map[string]string{
		common.AppSelector: tempest.ServiceName,
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
		instance.Spec.Parallel,
	)

	if err != nil {
		return ctrlResult, err
	} else if (ctrlResult != ctrl.Result{}) {
		return ctrlResult, nil
	}
	// Create PersistentVolumeClaim - end

	mountSSHKey := false
	if instance.Spec.SSHKeySecretName != "" {
		mountSSHKey = r.CheckSecretExists(ctx, instance, instance.Spec.SSHKeySecretName)
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
		requeueAfter := time.Second * 60
		return ctrl.Result{RequeueAfter: requeueAfter}, err
	}
	Log.Info("Lock acquired")

	if workflowActive {
		r.WorkflowStepCounterIncrease(ctx, instance, helper)
	}

	// Generate ConfigMaps
	err = r.generateServiceConfigMaps(ctx, helper, instance, externalWorkflowCounter)
	if err != nil {
		instance.Status.Conditions.Set(condition.FalseCondition(
			condition.ServiceConfigReadyCondition,
			condition.ErrorReason,
			condition.SeverityWarning,
			condition.ServiceConfigReadyErrorMessage,
			err.Error()))
		return ctrl.Result{}, err
	}
	instance.Status.Conditions.MarkTrue(condition.ServiceConfigReadyCondition, condition.ServiceConfigReadyMessage)
	// Generate ConfigMaps - end

	serviceAnnotations, err := nad.CreateNetworksAnnotation(instance.Namespace, instance.Spec.NetworkAttachments)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed create network annotation from %s: %w",
			instance.Spec.NetworkAttachments, err)
	}

	// NetworkAttachments
	networkReady, networkAttachmentStatus, err := nad.VerifyNetworkStatusFromAnnotation(ctx, helper, instance.Spec.NetworkAttachments, serviceLabels, 1)
	if err != nil {
		return ctrl.Result{}, err
	}
	instance.Status.NetworkAttachments = networkAttachmentStatus

	if networkReady {
		instance.Status.Conditions.MarkTrue(condition.NetworkAttachmentsReadyCondition, condition.NetworkAttachmentsReadyMessage)
	} else if r.JobExists(ctx, instance, externalWorkflowCounter) {
		err := fmt.Errorf("not all pods have interfaces with ips as configured in NetworkAttachments: %s", instance.Spec.NetworkAttachments)
		instance.Status.Conditions.Set(condition.FalseCondition(
			condition.NetworkAttachmentsReadyCondition,
			condition.ErrorReason,
			condition.SeverityWarning,
			condition.NetworkAttachmentsReadyErrorMessage,
			err.Error()))

		return ctrl.Result{}, err
	}
	// NetworkAttachments - end

	// Create a new job
	mountCerts := r.CheckSecretExists(ctx, instance, "combined-ca-bundle")
	customDataConfigMapName := GetCustomDataConfigMapName(instance, externalWorkflowCounter)
	EnvVarsConfigMapName := GetEnvVarsConfigMapName(instance, externalWorkflowCounter)
	jobName := r.GetJobName(instance, externalWorkflowCounter)
	logsPVCName := r.GetPVCLogsName(instance)

	// Note(lpiwowar): Remove all the workflow merge code to webhook once it is done.
	//                 It will simplify the logic and duplicite code (Tempest vs Tobiko)
	if externalWorkflowCounter < len(instance.Spec.Workflow) {
		if instance.Spec.Workflow[externalWorkflowCounter].NodeSelector != nil {
			instance.Spec.NodeSelector = *instance.Spec.Workflow[externalWorkflowCounter].NodeSelector
		}

		if instance.Spec.Workflow[externalWorkflowCounter].Tolerations != nil {
			instance.Spec.Tolerations = *instance.Spec.Workflow[externalWorkflowCounter].Tolerations
		}
	}

	jobDef := tempest.Job(
		instance,
		serviceLabels,
		serviceAnnotations,
		jobName,
		EnvVarsConfigMapName,
		customDataConfigMapName,
		logsPVCName,
		mountCerts,
		mountSSHKey,
	)
	tempestJob := job.NewJob(
		jobDef,
		testv1beta1.ConfigHash,
		true,
		time.Duration(5)*time.Second,
		"",
	)

	ctrlResult, err = tempestJob.DoJob(ctx, helper)
	if err != nil {
		// Creation of the tempest job was not successfull.
		// Release the lock and allow other controllers to spawn
		// a job.
		if lockReleased, lockErr := r.ReleaseLock(ctx, instance); lockReleased {
			return ctrl.Result{}, lockErr
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

func (r *TempestReconciler) reconcileDelete(
	ctx context.Context,
	instance *testv1beta1.Tempest,
	helper *helper.Helper,
) (ctrl.Result, error) {
	Log := r.GetLogger(ctx)
	Log.Info("Reconciling Service delete")

	// remove the finalizer
	controllerutil.RemoveFinalizer(instance, helper.GetFinalizer())

	Log.Info("Reconciled Service delete successfully")

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *TempestReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&testv1beta1.Tempest{}).
		Owns(&batchv1.Job{}).
		Owns(&corev1.Secret{}).
		Owns(&corev1.ConfigMap{}).
		Complete(r)
}

func (r *TempestReconciler) setTempestConfigVars(envVars map[string]string,
	customData map[string]string,
	instance *testv1beta1.Tempest,
	ctx context.Context,
	workflowStepNum int,
) {
	tRun := instance.Spec.TempestRun
	wtRun := testv1beta1.WorkflowTempestRunSpec{}
	if workflowStepNum < len(instance.Spec.Workflow) {
		wtRun = instance.Spec.Workflow[workflowStepNum].TempestRun
	}

	testOperatorDir := "/etc/test_operator/"

	// Files
	value := mergeWithWorkflow(tRun.WorkerFile, wtRun.WorkerFile)
	if len(value) != 0 {
		workerFile := "worker_file.yaml"
		customData[workerFile] = value
		envVars["TEMPEST_WORKER_FILE"] = testOperatorDir + workerFile
	}

	value = mergeWithWorkflow(tRun.IncludeList, wtRun.IncludeList)
	if len(value) != 0 {
		includeListFile := "include.txt"
		customData[includeListFile] = value
		envVars["TEMPEST_INCLUDE_LIST"] = testOperatorDir + includeListFile
	}

	value = mergeWithWorkflow(tRun.ExcludeList, wtRun.ExcludeList)
	if len(value) != 0 {
		excludeListFile := "exclude.txt"
		customData[excludeListFile] = value
		envVars["TEMPEST_EXCLUDE_LIST"] = testOperatorDir + excludeListFile
	}

	// Bool
	tempestBoolEnvVars := make(map[string]bool)
	tempestBoolEnvVars = map[string]bool{
		"TEMPEST_SERIAL":     mergeWithWorkflow(tRun.Serial, wtRun.Serial),
		"TEMPEST_PARALLEL":   mergeWithWorkflow(tRun.Parallel, wtRun.Parallel),
		"TEMPEST_SMOKE":      mergeWithWorkflow(tRun.Smoke, wtRun.Smoke),
		"USE_EXTERNAL_FILES": true,
	}

	for key, value := range tempestBoolEnvVars {
		envVars[key] = r.GetDefaultBool(value)
	}

	// Int
	numValue := mergeWithWorkflow(tRun.Concurrency, wtRun.Concurrency)
	envVars["TEMPEST_CONCURRENCY"] = r.GetDefaultInt(numValue)

	// Dictionary
	dictValue := mergeWithWorkflow(tRun.ExternalPlugin, wtRun.ExternalPlugin)
	for _, externalPluginDictionary := range dictValue {
		envVars["TEMPEST_EXTERNAL_PLUGIN_GIT_URL"] += externalPluginDictionary.Repository + ","

		if len(externalPluginDictionary.ChangeRepository) == 0 || len(externalPluginDictionary.ChangeRefspec) == 0 {
			envVars["TEMPEST_EXTERNAL_PLUGIN_CHANGE_URL"] += "-,"
			envVars["TEMPEST_EXTERNAL_PLUGIN_REFSPEC"] += "-,"
			continue
		}

		envVars["TEMPEST_EXTERNAL_PLUGIN_CHANGE_URL"] += externalPluginDictionary.ChangeRepository + ","
		envVars["TEMPEST_EXTERNAL_PLUGIN_REFSPEC"] += externalPluginDictionary.ChangeRefspec + ","
	}

	envVars["TEMPEST_WORKFLOW_STEP_DIR_NAME"] = r.GetJobName(instance, workflowStepNum)

	extraImages := mergeWithWorkflow(tRun.ExtraImages, wtRun.ExtraImages)
	for _, extraImageDict := range extraImages {
		envVars["TEMPEST_EXTRA_IMAGES_URL"] += extraImageDict.URL + ","
		envVars["TEMPEST_EXTRA_IMAGES_OS_CLOUD"] += extraImageDict.OsCloud + ","
		envVars["TEMPEST_EXTRA_IMAGES_CONTAINER_FORMAT"] += extraImageDict.ContainerFormat + ","
		envVars["TEMPEST_EXTRA_IMAGES_ID"] += extraImageDict.ID + ","
		envVars["TEMPEST_EXTRA_IMAGES_NAME"] += extraImageDict.Name + ","
		envVars["TEMPEST_EXTRA_IMAGES_DISK_FORMAT"] += extraImageDict.DiskFormat + ","
		envVars["TEMPEST_EXTRA_IMAGES_CREATE_TIMEOUT"] += r.GetDefaultInt(extraImageDict.ImageCreationTimeout) + ","

		envVars["TEMPEST_EXTRA_IMAGES_FLAVOR_ID"] += extraImageDict.Flavor.ID + ","
		envVars["TEMPEST_EXTRA_IMAGES_FLAVOR_NAME"] += extraImageDict.Flavor.Name + ","
		envVars["TEMPEST_EXTRA_IMAGES_FLAVOR_OS_CLOUD"] += extraImageDict.Flavor.OsCloud + ","
		envVars["TEMPEST_EXTRA_IMAGES_FLAVOR_RAM"] += r.GetDefaultInt(extraImageDict.Flavor.RAM, "-") + ","
		envVars["TEMPEST_EXTRA_IMAGES_FLAVOR_DISK"] += r.GetDefaultInt(extraImageDict.Flavor.Disk, "-") + ","
		envVars["TEMPEST_EXTRA_IMAGES_FLAVOR_VCPUS"] += r.GetDefaultInt(extraImageDict.Flavor.Vcpus, "-") + ","
	}

	extraRPMs := mergeWithWorkflow(tRun.ExtraRPMs, wtRun.ExtraRPMs)
	for _, extraRPMURL := range extraRPMs {
		envVars["TEMPEST_EXTRA_RPMS"] += extraRPMURL + ","
	}
}

func mergeWithWorkflow[T any](value T, workflowValue *T) T {
	if workflowValue == nil {
		return value
	}

	return *workflowValue
}

func (r *TempestReconciler) setTempestconfConfigVars(
	envVars map[string]string,
	customData map[string]string,
	instance *testv1beta1.Tempest,
	ctx context.Context,
	workflowStepNum int,
) {
	tcRun := instance.Spec.TempestconfRun
	wtcRun := testv1beta1.WorkflowTempestconfRunSpec{}
	if workflowStepNum < len(instance.Spec.Workflow) {
		wtcRun = instance.Spec.Workflow[workflowStepNum].TempestconfRun
	}

	testOperatorDir := "/etc/test_operator/"
	value := mergeWithWorkflow(tcRun.DeployerInput, wtcRun.DeployerInput)
	if len(value) != 0 {
		deployerInputFile := "deployer_input.ini"
		customData[deployerInputFile] = value
		envVars["TEMPESTCONF_DEPLOYER_INPUT"] = testOperatorDir + deployerInputFile
	}

	value = mergeWithWorkflow(tcRun.TestAccounts, wtcRun.TestAccounts)
	if len(value) != 0 {
		accountsFile := "accounts.yaml"
		customData[accountsFile] = value
		envVars["TEMPESTCONF_TEST_ACCOUNTS"] = testOperatorDir + accountsFile
	}

	value = mergeWithWorkflow(tcRun.Profile, wtcRun.Profile)
	if len(value) != 0 {
		profileFile := "profile.yaml"
		customData[profileFile] = value
		envVars["TEMPESTCONF_PROFILE"] = testOperatorDir + profileFile
	}

	// Bool
	tempestconfBoolEnvVars := make(map[string]bool)
	tempestconfBoolEnvVars = map[string]bool{
		"TEMPESTCONF_CREATE":              mergeWithWorkflow(tcRun.Create, wtcRun.Create),
		"TEMPESTCONF_COLLECT_TIMING":      mergeWithWorkflow(tcRun.CollectTiming, wtcRun.CollectTiming),
		"TEMPESTCONF_INSECURE":            mergeWithWorkflow(tcRun.Insecure, wtcRun.Insecure),
		"TEMPESTCONF_NO_DEFAULT_DEPLOYER": mergeWithWorkflow(tcRun.NoDefaultDeployer, wtcRun.NoDefaultDeployer),
		"TEMPESTCONF_DEBUG":               mergeWithWorkflow(tcRun.Debug, wtcRun.Debug),
		"TEMPESTCONF_VERBOSE":             mergeWithWorkflow(tcRun.Verbose, wtcRun.Verbose),
		"TEMPESTCONF_NON_ADMIN":           mergeWithWorkflow(tcRun.NonAdmin, wtcRun.NonAdmin),
		"TEMPESTCONF_RETRY_IMAGE":         mergeWithWorkflow(tcRun.RetryImage, wtcRun.RetryImage),
		"TEMPESTCONF_CONVERT_TO_RAW":      mergeWithWorkflow(tcRun.ConvertToRaw, wtcRun.ConvertToRaw),
	}

	for key, value := range tempestconfBoolEnvVars {
		envVars[key] = r.GetDefaultBool(value)
	}

	tempestconfIntEnvVars := make(map[string]int64)
	tempestconfIntEnvVars = map[string]int64{
		"TEMPESTCONF_TIMEOUT":         mergeWithWorkflow(tcRun.Timeout, wtcRun.Timeout),
		"TEMPESTCONF_FLAVOR_MIN_MEM":  mergeWithWorkflow(tcRun.FlavorMinMem, wtcRun.FlavorMinMem),
		"TEMPESTCONF_FLAVOR_MIN_DISK": mergeWithWorkflow(tcRun.FlavorMinDisk, wtcRun.FlavorMinDisk),
	}

	for key, value := range tempestconfIntEnvVars {
		envVars[key] = r.GetDefaultInt(value)
	}

	// String
	mValue := mergeWithWorkflow(tcRun.Out, wtcRun.Out)
	envVars["TEMPESTCONF_OUT"] = mValue

	mValue = mergeWithWorkflow(tcRun.CreateAccountsFile, wtcRun.CreateAccountsFile)
	envVars["TEMPESTCONF_CREATE_ACCOUNTS_FILE"] = mValue

	mValue = mergeWithWorkflow(tcRun.GenerateProfile, wtcRun.GenerateProfile)
	envVars["TEMPESTCONF_GENERATE_PROFILE"] = mValue

	mValue = mergeWithWorkflow(tcRun.ImageDiskFormat, wtcRun.ImageDiskFormat)
	envVars["TEMPESTCONF_IMAGE_DISK_FORMAT"] = mValue

	mValue = mergeWithWorkflow(tcRun.Image, wtcRun.Image)
	envVars["TEMPESTCONF_IMAGE"] = mValue

	mValue = mergeWithWorkflow(tcRun.NetworkID, wtcRun.NetworkID)
	envVars["TEMPESTCONF_NETWORK_ID"] = mValue

	mValue = mergeWithWorkflow(tcRun.Append, wtcRun.Append)
	envVars["TEMPESTCONF_APPEND"] = mValue

	mValue = mergeWithWorkflow(tcRun.Remove, wtcRun.Remove)
	envVars["TEMPESTCONF_REMOVE"] = mValue

	mValue = mergeWithWorkflow(tcRun.Overrides, wtcRun.Overrides)
	envVars["TEMPESTCONF_OVERRIDES"] = mValue
}

// Create ConfigMaps:
//   - %-env-vars contians all the environment variables that are needed for
//     execution of the tempest container
//   - %-config contains all the files that are needed for the execution of
//     the tempest container
func (r *TempestReconciler) generateServiceConfigMaps(
	ctx context.Context,
	h *helper.Helper,
	instance *testv1beta1.Tempest,
	workflowStepNum int,
) error {
	// Create/update configmaps from template
	cmLabels := labels.GetLabels(instance, labels.GetGroupLabel(tempest.ServiceName), map[string]string{})

	operatorLabels := map[string]string{
		"operator":     "test-operator",
		"instanceName": instance.Name,
	}

	// Combine labels
	for key, value := range operatorLabels {
		cmLabels[key] = value
	}

	templateParameters := make(map[string]interface{})
	customData := make(map[string]string)
	envVars := make(map[string]string)

	r.setTempestConfigVars(envVars, customData, instance, ctx, workflowStepNum)
	r.setTempestconfConfigVars(envVars, customData, instance, ctx, workflowStepNum)
	r.setConfigOverwrite(customData, instance.Spec.ConfigOverwrite)

	envVars["TEMPEST_DEBUG_MODE"] = r.GetDefaultBool(instance.Spec.Debug)

	cms := []util.Template{
		// ConfigMap
		{
			Name:          GetCustomDataConfigMapName(instance, workflowStepNum),
			Namespace:     instance.Namespace,
			InstanceType:  instance.Kind,
			Labels:        cmLabels,
			ConfigOptions: templateParameters,
			CustomData:    customData,
		},
		// configMap - EnvVars
		{
			Name:          GetEnvVarsConfigMapName(instance, workflowStepNum),
			Namespace:     instance.Namespace,
			InstanceType:  instance.Kind,
			Labels:        cmLabels,
			ConfigOptions: templateParameters,
			CustomData:    envVars,
		},
	}

	return configmap.EnsureConfigMaps(ctx, h, instance, cms, nil)
}
