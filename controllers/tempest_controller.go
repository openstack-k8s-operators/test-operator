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

	"github.com/go-logr/logr"
	"github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	"github.com/openstack-k8s-operators/lib-common/modules/common/configmap"
	"github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	"github.com/openstack-k8s-operators/lib-common/modules/common/labels"
	"github.com/openstack-k8s-operators/lib-common/modules/common/util"
	testv1beta1 "github.com/openstack-k8s-operators/test-operator/api/v1beta1"
	"github.com/openstack-k8s-operators/test-operator/pkg/tempest"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// TempestReconciler reconciles a Tempest object
type TempestReconciler struct {
	Reconciler
}

// GetLogger returns a logger object with a prefix of "controller.name"
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
func (r *TempestReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Fetch the Tempest instance
	instance := &testv1beta1.Tempest{}

	config := FrameworkConfig[*testv1beta1.Tempest]{
		ServiceName:               tempest.ServiceName,
		NeedsNetworkAttachments:   true,
		NeedsConfigMaps:           true,
		GenerateServiceConfigMaps: generateTempestServiceConfigMaps,
		BuildPod:                  buildTempestPod,

		GetInitialConditions: func() []*condition.Condition {
			return []*condition.Condition{
				condition.UnknownCondition(condition.ReadyCondition, condition.InitReason, condition.ReadyInitMessage),
				condition.UnknownCondition(condition.ServiceConfigReadyCondition, condition.InitReason, condition.ServiceConfigReadyInitMessage),
				condition.UnknownCondition(condition.DeploymentReadyCondition, condition.InitReason, condition.DeploymentReadyInitMessage),
				condition.UnknownCondition(condition.NetworkAttachmentsReadyCondition, condition.InitReason, condition.NetworkAttachmentsReadyInitMessage),
			}
		},

		GetSpec: func(instance *testv1beta1.Tempest) interface{} {
			return &instance.Spec
		},

		GetWorkflowStep: func(instance *testv1beta1.Tempest, step int) interface{} {
			return instance.Spec.Workflow[step]
		},

		GetWorkflowLength: func(instance *testv1beta1.Tempest) int {
			return len(instance.Spec.Workflow)
		},

		GetParallel: func(instance *testv1beta1.Tempest) bool {
			return instance.Spec.Parallel
		},

		GetStorageClass: func(instance *testv1beta1.Tempest) string {
			return instance.Spec.StorageClass
		},

		GetNetworkAttachments: func(instance *testv1beta1.Tempest) []string {
			return instance.Spec.NetworkAttachments
		},

		GetNetworkAttachmentStatus: func(instance *testv1beta1.Tempest) map[string][]string {
			return instance.Status.NetworkAttachments
		},

		SetNetworkAttachmentStatus: func(instance *testv1beta1.Tempest, status map[string][]string) {
			instance.Status.NetworkAttachments = status
		},
	}

	return CommonReconcile(ctx, &r.Reconciler, req, instance, config, r.GetLogger(ctx))
}

func buildTempestPod(
	ctx context.Context,
	r *Reconciler,
	instance *testv1beta1.Tempest,
	labels, annotations map[string]string,
	workflowStep int,
) (*corev1.Pod, error) {
	mountSSHKey := false
	if instance.Spec.SSHKeySecretName != "" {
		mountSSHKey = r.CheckSecretExists(ctx, instance, instance.Spec.SSHKeySecretName)
	}

	mountCerts := r.CheckSecretExists(ctx, instance, "combined-ca-bundle")
	customDataConfigMapName := GetCustomDataConfigMapName(instance, workflowStep)
	EnvVarsConfigMapName := GetEnvVarsConfigMapName(instance, workflowStep)
	podName := r.GetPodName(instance, workflowStep)

	workflowStepNum := 0
	if instance.Spec.Parallel && workflowStep < len(instance.Spec.Workflow) {
		workflowStepNum = workflowStep
	}
	logsPVCName := r.GetPVCLogsName(instance, workflowStepNum)

	containerImage, err := r.GetContainerImage(ctx, instance.Spec.ContainerImage, instance)
	if err != nil {
		return nil, err
	}

	return tempest.Pod(
		instance,
		labels,
		annotations,
		podName,
		EnvVarsConfigMapName,
		customDataConfigMapName,
		logsPVCName,
		mountCerts,
		mountSSHKey,
		containerImage,
	), nil
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

func setTempestConfigVars(
	r *Reconciler,
	envVars map[string]string,
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

func setTempestconfConfigVars(
	r *Reconciler,
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
func generateTempestServiceConfigMaps(
	ctx context.Context,
	r *Reconciler,
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

	setTempestConfigVars(r, envVars, customData, instance, workflowStepNum)
	setTempestconfConfigVars(r, envVars, customData, instance)
	r.setConfigOverwrite(customData, instance.Spec.ConfigOverwrite)

	envVars["TEMPEST_DEBUG_MODE"] = r.GetDefaultBool(instance.Spec.Debug)
	envVars["TEMPEST_CLEANUP"] = r.GetDefaultBool(instance.Spec.Cleanup)
	envVars["TEMPEST_RERUN_FAILED_TESTS"] = r.GetDefaultBool(instance.Spec.RerunFailedTests)
	envVars["TEMPEST_RERUN_OVERRIDE_STATUS"] = r.GetDefaultBool(instance.Spec.RerunOverrideStatus)
	envVars["TEMPEST_TIMING_DATA_URL"] = instance.Spec.TimingDataUrl

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
