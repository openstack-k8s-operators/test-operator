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
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/go-logr/logr"
	networkv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	"github.com/openstack-k8s-operators/lib-common/modules/common"
	"github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	"github.com/openstack-k8s-operators/lib-common/modules/common/configmap"
	"github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	"github.com/openstack-k8s-operators/lib-common/modules/common/labels"
	nad "github.com/openstack-k8s-operators/lib-common/modules/common/networkattachment"
	"github.com/openstack-k8s-operators/lib-common/modules/common/util"
	testv1beta1 "github.com/openstack-k8s-operators/test-operator/api/v1beta1"
	"github.com/openstack-k8s-operators/test-operator/pkg/tempest"
	corev1 "k8s.io/api/core/v1"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
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
// +kubebuilder:rbac:groups=k8s.cni.cncf.io,resources=network-attachment-definitions,verbs=get;list;watch
// +kubebuilder:rbac:groups="security.openshift.io",resourceNames=anyuid;privileged;nonroot;nonroot-v2,resources=securitycontextconstraints,verbs=use
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete;
// +kubebuilder:rbac:groups="",resources=pods,verbs=create;delete;get;list;patch;update;watch
// +kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;create;update;watch;patch;delete

// Reconcile - Tempest
func (r *TempestReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, _err error) {
	Log := r.GetLogger(ctx)

	// Fetch the Tempest instance
	instance := &testv1beta1.Tempest{}
	err := r.Client.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if k8s_errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
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
			condition.UnknownCondition(condition.ServiceConfigReadyCondition, condition.InitReason, condition.ServiceConfigReadyInitMessage),
			condition.UnknownCondition(condition.DeploymentReadyCondition, condition.InitReason, condition.DeploymentReadyInitMessage),
			condition.UnknownCondition(condition.NetworkAttachmentsReadyCondition, condition.InitReason, condition.NetworkAttachmentsReadyInitMessage),
		)
		instance.Status.Conditions.Init(&cl)

		// Register overall status immediately to have an early feedback
		// e.g. in the cli
		return ctrl.Result{}, nil
	}

	// If we're not deleting this and the service object doesn't have our
	// finalizer, add it.
	if instance.DeletionTimestamp.IsZero() && controllerutil.AddFinalizer(instance, helper.GetFinalizer()) {
		return ctrl.Result{}, nil
	}

	if instance.Status.NetworkAttachments == nil {
		instance.Status.NetworkAttachments = map[string][]string{}
	}

	// Handle service delete
	if !instance.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, instance, helper)
	}

	workflowLength := len(instance.Spec.Workflow)
	nextAction, nextWorkflowStep, err := r.NextAction(ctx, instance, workflowLength)
	if nextWorkflowStep < workflowLength {
		MergeSections(&instance.Spec, instance.Spec.Workflow[nextWorkflowStep])
	}

	switch nextAction {
	case Failure:
		return ctrl.Result{}, err

	case Wait:
		Log.Info(InfoWaitingOnPod)
		return ctrl.Result{RequeueAfter: RequeueAfterValue}, nil

	case EndTesting:
		// All pods created by the instance were completed. Release the lock
		// so that other instances can spawn their pods.
		if lockReleased, err := r.ReleaseLock(ctx, instance); !lockReleased {
			Log.Info(fmt.Sprintf(InfoCanNotReleaseLock, testOperatorLockName))
			return ctrl.Result{RequeueAfter: RequeueAfterValue}, err
		}

		instance.Status.Conditions.MarkTrue(
			condition.DeploymentReadyCondition,
			condition.DeploymentReadyMessage)

		Log.Info(InfoTestingCompleted)
		return ctrl.Result{}, nil

	case CreateFirstPod:
		lockAcquired, err := r.AcquireLock(ctx, instance, helper, instance.Spec.Parallel)
		if !lockAcquired {
			Log.Info(fmt.Sprintf(InfoCanNotAcquireLock, testOperatorLockName))
			return ctrl.Result{RequeueAfter: RequeueAfterValue}, err
		}

		Log.Info(fmt.Sprintf(InfoCreatingFirstPod, nextWorkflowStep))

	case CreateNextPod:
		// Confirm that we still hold the lock. This is useful to check if for
		// example somebody / something deleted the lock and it got claimed by
		// another instance. This is considered to be an error state.
		lockAcquired, err := r.AcquireLock(ctx, instance, helper, instance.Spec.Parallel)
		if !lockAcquired {
			Log.Error(err, ErrConfirmLockOwnership, testOperatorLockName)
			return ctrl.Result{RequeueAfter: RequeueAfterValue}, err
		}

		Log.Info(fmt.Sprintf(InfoCreatingNextPod, nextWorkflowStep))

	default:
		return ctrl.Result{}, errors.New(ErrReceivedUnexpectedAction)
	}

	serviceLabels := map[string]string{
		common.AppSelector: tempest.ServiceName,
		workflowStepLabel:  strconv.Itoa(nextWorkflowStep),
		instanceNameLabel:  instance.Name,
		operatorNameLabel:  "test-operator",
	}

	workflowStepNum := 0
	// Create multiple PVCs for parallel execution
	if instance.Spec.Parallel && nextWorkflowStep < len(instance.Spec.Workflow) {
		workflowStepNum = nextWorkflowStep
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

	mountSSHKey := false
	if instance.Spec.SSHKeySecretName != "" {
		mountSSHKey = r.CheckSecretExists(ctx, instance, instance.Spec.SSHKeySecretName)
	}

	// Generate ConfigMaps
	err = r.generateServiceConfigMaps(ctx, helper, instance, nextWorkflowStep)
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

	nadList := []networkv1.NetworkAttachmentDefinition{}
	for _, netAtt := range instance.Spec.NetworkAttachments {
		nad, err := nad.GetNADWithName(ctx, helper, netAtt, instance.Namespace)
		if err != nil {
			if k8s_errors.IsNotFound(err) {
				Log.Info(fmt.Sprintf("network-attachment-definition %s not found", netAtt))
				instance.Status.Conditions.Set(condition.FalseCondition(
					condition.NetworkAttachmentsReadyCondition,
					condition.RequestedReason,
					condition.SeverityInfo,
					condition.NetworkAttachmentsReadyWaitingMessage,
					netAtt))
				return ctrl.Result{RequeueAfter: time.Second * 10}, nil
			}
			instance.Status.Conditions.Set(condition.FalseCondition(
				condition.NetworkAttachmentsReadyCondition,
				condition.ErrorReason,
				condition.SeverityWarning,
				condition.NetworkAttachmentsReadyErrorMessage,
				err.Error()))
			return ctrl.Result{}, err
		}

		if nad != nil {
			nadList = append(nadList, *nad)
		}
	}

	serviceAnnotations, err := nad.EnsureNetworksAnnotation(nadList)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed create network annotation from %s: %w",
			instance.Spec.NetworkAttachments, err)
	}

	// Create a new pod
	mountCerts := r.CheckSecretExists(ctx, instance, "combined-ca-bundle")
	customDataConfigMapName := GetCustomDataConfigMapName(instance, nextWorkflowStep)
	EnvVarsConfigMapName := GetEnvVarsConfigMapName(instance, nextWorkflowStep)
	podName := r.GetPodName(instance, nextWorkflowStep)
	logsPVCName := r.GetPVCLogsName(instance, workflowStepNum)
	containerImage, err := r.GetContainerImage(ctx, instance.Spec.ContainerImage, instance)
	if err != nil {
		return ctrl.Result{}, err
	}

	podDef := tempest.Pod(
		instance,
		serviceLabels,
		serviceAnnotations,
		podName,
		EnvVarsConfigMapName,
		customDataConfigMapName,
		logsPVCName,
		mountCerts,
		mountSSHKey,
		containerImage,
	)

	ctrlResult, err = r.CreatePod(ctx, *helper, podDef)
	if err != nil {
		// Creation of the tempest pod was not successfull.
		// Release the lock and allow other controllers to spawn
		// a pod.
		if lockReleased, lockErr := r.ReleaseLock(ctx, instance); lockReleased {
			return ctrl.Result{RequeueAfter: RequeueAfterValue}, lockErr
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
	// Create a new pod - end

	// NetworkAttachments
	if r.PodExists(ctx, instance, nextWorkflowStep) {
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
		Owns(&corev1.Pod{}).
		Owns(&corev1.Secret{}).
		Owns(&corev1.ConfigMap{}).
		Complete(r)
}

func (r *TempestReconciler) setTempestConfigVars(envVars map[string]string,
	customData map[string]string,
	instance *testv1beta1.Tempest,
	workflowStepNum int,
) {
	tRun := instance.Spec.TempestRun

	testOperatorDir := "/etc/test_operator/"

	// Files
	value := tRun.WorkerFile
	if len(value) != 0 {
		workerFile := "worker_file.yaml"
		customData[workerFile] = value
		envVars["TEMPEST_WORKER_FILE"] = testOperatorDir + workerFile
	}

	value = tRun.IncludeList
	if len(value) != 0 {
		includeListFile := "include.txt"
		customData[includeListFile] = value
		envVars["TEMPEST_INCLUDE_LIST"] = testOperatorDir + includeListFile
	}

	value = tRun.ExcludeList
	if len(value) != 0 {
		excludeListFile := "exclude.txt"
		customData[excludeListFile] = value
		envVars["TEMPEST_EXCLUDE_LIST"] = testOperatorDir + excludeListFile
	}

	value = tRun.ExpectedFailuresList
	if len(value) != 0 {
		expectedFailuresListFile := "expected_failures.txt"
		customData[expectedFailuresListFile] = value
		envVars["TEMPEST_EXPECTED_FAILURES_LIST"] = testOperatorDir + expectedFailuresListFile
	}

	// Bool
	tempestBoolEnvVars := map[string]bool{
		"TEMPEST_SERIAL":     tRun.Serial,
		"TEMPEST_PARALLEL":   tRun.Parallel,
		"TEMPEST_SMOKE":      tRun.Smoke,
		"USE_EXTERNAL_FILES": true,
	}

	for key, value := range tempestBoolEnvVars {
		envVars[key] = r.GetDefaultBool(value)
	}

	// Int
	numValue := tRun.Concurrency
	envVars["TEMPEST_CONCURRENCY"] = r.GetDefaultInt(numValue)

	// Dictionary
	dictValue := tRun.ExternalPlugin
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

	envVars["TEMPEST_WORKFLOW_STEP_DIR_NAME"] = r.GetPodName(instance, workflowStepNum)

	extraImages := tRun.ExtraImages
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

	extraRPMs := tRun.ExtraRPMs
	for _, extraRPMURL := range extraRPMs {
		envVars["TEMPEST_EXTRA_RPMS"] += extraRPMURL + ","
	}
}

func (r *TempestReconciler) setTempestconfConfigVars(
	envVars map[string]string,
	customData map[string]string,
	instance *testv1beta1.Tempest,
) {
	tcRun := instance.Spec.TempestconfRun

	testOperatorDir := "/etc/test_operator/"
	value := tcRun.DeployerInput
	if len(value) != 0 {
		deployerInputFile := "deployer_input.ini"
		customData[deployerInputFile] = value
		envVars["TEMPESTCONF_DEPLOYER_INPUT"] = testOperatorDir + deployerInputFile
	}

	value = tcRun.TestAccounts
	if len(value) != 0 {
		accountsFile := "accounts.yaml"
		customData[accountsFile] = value
		envVars["TEMPESTCONF_TEST_ACCOUNTS"] = testOperatorDir + accountsFile
	}

	value = tcRun.Profile
	if len(value) != 0 {
		profileFile := "profile.yaml"
		customData[profileFile] = value
		envVars["TEMPESTCONF_PROFILE"] = testOperatorDir + profileFile
	}

	// Bool
	tempestconfBoolEnvVars := map[string]bool{
		"TEMPESTCONF_CREATE":              tcRun.Create,
		"TEMPESTCONF_COLLECT_TIMING":      tcRun.CollectTiming,
		"TEMPESTCONF_INSECURE":            tcRun.Insecure,
		"TEMPESTCONF_NO_DEFAULT_DEPLOYER": tcRun.NoDefaultDeployer,
		"TEMPESTCONF_DEBUG":               tcRun.Debug,
		"TEMPESTCONF_VERBOSE":             tcRun.Verbose,
		"TEMPESTCONF_NON_ADMIN":           tcRun.NonAdmin,
		"TEMPESTCONF_RETRY_IMAGE":         tcRun.RetryImage,
		"TEMPESTCONF_CONVERT_TO_RAW":      tcRun.ConvertToRaw,
	}

	for key, value := range tempestconfBoolEnvVars {
		envVars[key] = r.GetDefaultBool(value)
	}

	tempestconfIntEnvVars := map[string]int64{
		"TEMPESTCONF_TIMEOUT":         tcRun.Timeout,
		"TEMPESTCONF_FLAVOR_MIN_MEM":  tcRun.FlavorMinMem,
		"TEMPESTCONF_FLAVOR_MIN_DISK": tcRun.FlavorMinDisk,
	}

	for key, value := range tempestconfIntEnvVars {
		envVars[key] = r.GetDefaultInt(value)
	}

	// String
	envVars["TEMPESTCONF_OUT"] = tcRun.Out
	envVars["TEMPESTCONF_CREATE_ACCOUNTS_FILE"] = tcRun.CreateAccountsFile
	envVars["TEMPESTCONF_GENERATE_PROFILE"] = tcRun.GenerateProfile
	envVars["TEMPESTCONF_IMAGE_DISK_FORMAT"] = tcRun.ImageDiskFormat
	envVars["TEMPESTCONF_IMAGE"] = tcRun.Image
	envVars["TEMPESTCONF_NETWORK_ID"] = tcRun.NetworkID
	envVars["TEMPESTCONF_APPEND"] = tcRun.Append
	envVars["TEMPESTCONF_REMOVE"] = tcRun.Remove
	envVars["TEMPESTCONF_OVERRIDES"] = tcRun.Overrides
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
		operatorNameLabel: "test-operator",
		instanceNameLabel: instance.Name,
	}

	// Combine labels
	for key, value := range operatorLabels {
		cmLabels[key] = value
	}

	templateParameters := make(map[string]interface{})
	customData := make(map[string]string)
	envVars := make(map[string]string)

	r.setTempestConfigVars(envVars, customData, instance, workflowStepNum)
	r.setTempestconfConfigVars(envVars, customData, instance)
	r.setConfigOverwrite(customData, instance.Spec.ConfigOverwrite)

	envVars["TEMPEST_DEBUG_MODE"] = r.GetDefaultBool(instance.Spec.Debug)
	envVars["TEMPEST_CLEANUP"] = r.GetDefaultBool(instance.Spec.Cleanup)
	envVars["TEMPEST_RERUN_FAILED_TESTS"] = r.GetDefaultBool(instance.Spec.RerunFailedTests)
	envVars["TEMPEST_RERUN_OVERRIDE_STATUS"] = r.GetDefaultBool(instance.Spec.RerunOverrideStatus)

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
