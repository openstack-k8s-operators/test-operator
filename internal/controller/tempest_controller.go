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

package controller

import (
	"context"
	"strconv"

	"github.com/go-logr/logr"
	"github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	"github.com/openstack-k8s-operators/lib-common/modules/common/configmap"
	"github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	"github.com/openstack-k8s-operators/lib-common/modules/common/labels"
	"github.com/openstack-k8s-operators/lib-common/modules/common/util"
	testv1beta1 "github.com/openstack-k8s-operators/test-operator/api/v1beta1"
	"github.com/openstack-k8s-operators/test-operator/internal/tempest"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// TempestReconciler reconciles a Tempest object
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
func (r *TempestReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	instance := &testv1beta1.Tempest{}

	config := FrameworkConfig[*testv1beta1.Tempest]{
		ServiceName:             tempest.ServiceName,
		NeedsNetworkAttachments: true,
		NeedsConfigMaps:         true,
		NeedsFinalizer:          true,
		SupportsWorkflow:        true,

		GenerateServiceConfigMaps: func(ctx context.Context, helper *helper.Helper, instance *testv1beta1.Tempest, workflowStep int) error {
			return r.generateServiceConfigMaps(ctx, helper, instance, workflowStep)
		},

		BuildPod: func(ctx context.Context, instance *testv1beta1.Tempest, labels, annotations map[string]string, workflowStepNum int, pvcIndex int) (*corev1.Pod, error) {
			return r.buildTempestPod(ctx, instance, labels, annotations, workflowStepNum, pvcIndex)
		},

		GetInitialConditions: func() []*condition.Condition {
			return []*condition.Condition{
				condition.UnknownCondition(condition.ReadyCondition, condition.InitReason, condition.ReadyInitMessage),
				condition.UnknownCondition(condition.InputReadyCondition, condition.InitReason, condition.InputReadyInitMessage),
				condition.UnknownCondition(condition.ServiceConfigReadyCondition, condition.InitReason, condition.ServiceConfigReadyInitMessage),
				condition.UnknownCondition(condition.DeploymentReadyCondition, condition.InitReason, condition.DeploymentReadyInitMessage),
				condition.UnknownCondition(condition.NetworkAttachmentsReadyCondition, condition.InitReason, condition.NetworkAttachmentsReadyInitMessage),
			}
		},

		ValidateInputs: func(ctx context.Context, instance *testv1beta1.Tempest) error {
			if err := r.ValidateOpenstackInputs(ctx, instance, instance.Spec.OpenStackConfigMap, instance.Spec.OpenStackConfigSecret); err != nil {
				return err
			}
			return r.ValidateSecretWithKeys(ctx, instance, instance.Spec.SSHKeySecretName, []string{})
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

		GetNetworkAttachmentStatus: func(instance *testv1beta1.Tempest) *map[string][]string {
			return &instance.Status.NetworkAttachments
		},

		SetObservedGeneration: func(instance *testv1beta1.Tempest) {
			instance.Status.ObservedGeneration = instance.Generation
		},
	}

	return CommonReconcile(ctx, &r.Reconciler, req, instance, config, r.GetLogger(ctx))
}

func (r *TempestReconciler) buildTempestPod(
	ctx context.Context,
	instance *testv1beta1.Tempest,
	labels, annotations map[string]string,
	workflowStepNum int,
	pvcIndex int,
) (*corev1.Pod, error) {
	mountSSHKey := len(instance.Spec.SSHKeySecretName) != 0
	mountCerts := r.CheckSecretExists(ctx, instance, "combined-ca-bundle")

	customDataConfigMapName := GetCustomDataConfigMapName(instance, workflowStepNum)
	envVarsConfigMapName := GetEnvVarsConfigMapName(instance, workflowStepNum)

	podName := r.GetPodName(instance, workflowStepNum)
	logsPVCName := r.GetPVCLogsName(instance, pvcIndex)

	containerImage, err := r.GetContainerImage(ctx, instance)
	if err != nil {
		return nil, err
	}

	return tempest.Pod(
		instance,
		labels,
		annotations,
		podName,
		envVarsConfigMapName,
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

func (r *TempestReconciler) setTempestConfigVars(envVars map[string]string,
	customData map[string]string,
	instance *testv1beta1.Tempest,
	workflowStepNum int,
) {
	tRun := instance.Spec.TempestRun

	// Files
	SetFileEnvVar(customData, envVars, tRun.WorkerFile, "worker_file.yaml", "TEMPEST_WORKER_FILE")
	SetFileEnvVar(customData, envVars, tRun.IncludeList, "include.txt", "TEMPEST_INCLUDE_LIST")
	SetFileEnvVar(customData, envVars, tRun.ExcludeList, "exclude.txt", "TEMPEST_EXCLUDE_LIST")
	SetFileEnvVar(customData, envVars, tRun.ExpectedFailuresList, "expected_failures.txt", "TEMPEST_EXPECTED_FAILURES_LIST")

	// Bool
	tempestBoolEnvVars := map[string]bool{
		"TEMPEST_SERIAL":     tRun.Serial,
		"TEMPEST_PARALLEL":   tRun.Parallel,
		"TEMPEST_SMOKE":      tRun.Smoke,
		"USE_EXTERNAL_FILES": true,
	}

	for key, value := range tempestBoolEnvVars {
		envVars[key] = strconv.FormatBool(value)
	}

	// Int
	if tRun.Concurrency > 0 {
		envVars["TEMPEST_CONCURRENCY"] = strconv.FormatInt(tRun.Concurrency, 10)
	}

	// Dictionary
	for _, plugin := range tRun.ExternalPlugin {
		SetDictEnvVar(envVars, map[string]string{
			"TEMPEST_EXTERNAL_PLUGIN_GIT_URL":    plugin.Repository,
			"TEMPEST_EXTERNAL_PLUGIN_CHANGE_URL": StringOrPlaceholder(plugin.ChangeRepository, "-"),
			"TEMPEST_EXTERNAL_PLUGIN_REFSPEC":    StringOrPlaceholder(plugin.ChangeRefspec, "-"),
		})
	}

	envVars["TEMPEST_WORKFLOW_STEP_DIR_NAME"] = r.GetPodName(instance, workflowStepNum)

	for _, img := range tRun.ExtraImages {
		SetDictEnvVar(envVars, map[string]string{
			"TEMPEST_EXTRA_IMAGES_URL":              img.URL,
			"TEMPEST_EXTRA_IMAGES_OS_CLOUD":         img.OsCloud,
			"TEMPEST_EXTRA_IMAGES_CONTAINER_FORMAT": img.ContainerFormat,
			"TEMPEST_EXTRA_IMAGES_ID":               img.ID,
			"TEMPEST_EXTRA_IMAGES_NAME":             img.Name,
			"TEMPEST_EXTRA_IMAGES_DISK_FORMAT":      img.DiskFormat,
			"TEMPEST_EXTRA_IMAGES_CREATE_TIMEOUT":   Int64OrPlaceholder(img.ImageCreationTimeout, ""),

			"TEMPEST_EXTRA_IMAGES_FLAVOR_ID":       img.Flavor.ID,
			"TEMPEST_EXTRA_IMAGES_FLAVOR_NAME":     img.Flavor.Name,
			"TEMPEST_EXTRA_IMAGES_FLAVOR_OS_CLOUD": img.Flavor.OsCloud,
			"TEMPEST_EXTRA_IMAGES_FLAVOR_RAM":      Int64OrPlaceholder(img.Flavor.RAM, "-"),
			"TEMPEST_EXTRA_IMAGES_FLAVOR_DISK":     Int64OrPlaceholder(img.Flavor.Disk, "-"),
			"TEMPEST_EXTRA_IMAGES_FLAVOR_VCPUS":    Int64OrPlaceholder(img.Flavor.Vcpus, "-"),
		})
	}

	for _, rpm := range tRun.ExtraRPMs {
		SetDictEnvVar(envVars, map[string]string{
			"TEMPEST_EXTRA_RPMS": rpm,
		})
	}
}

func (r *TempestReconciler) setTempestconfConfigVars(
	envVars map[string]string,
	customData map[string]string,
	instance *testv1beta1.Tempest,
) {
	tcRun := instance.Spec.TempestconfRun

	// Files
	SetFileEnvVar(customData, envVars, tcRun.DeployerInput, "deployer_input.ini", "TEMPESTCONF_DEPLOYER_INPUT")
	SetFileEnvVar(customData, envVars, tcRun.TestAccounts, "accounts.yaml", "TEMPESTCONF_TEST_ACCOUNTS")
	SetFileEnvVar(customData, envVars, tcRun.Profile, "profile.yaml", "TEMPESTCONF_PROFILE")

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
		envVars[key] = strconv.FormatBool(value)
	}

	tempestconfIntEnvVars := map[string]int64{
		"TEMPESTCONF_TIMEOUT":         tcRun.Timeout,
		"TEMPESTCONF_FLAVOR_MIN_MEM":  tcRun.FlavorMinMem,
		"TEMPESTCONF_FLAVOR_MIN_DISK": tcRun.FlavorMinDisk,
	}

	for key, value := range tempestconfIntEnvVars {
		envVars[key] = Int64OrPlaceholder(value, "")
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
//   - %-env-vars contains all the environment variables that are needed for
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

	for key, data := range instance.Spec.ConfigOverwrite {
		customData[key] = data
	}

	envVars["TEMPEST_DEBUG_MODE"] = strconv.FormatBool(instance.Spec.Debug)
	envVars["TEMPEST_CLEANUP"] = strconv.FormatBool(instance.Spec.Cleanup)
	envVars["TEMPEST_RERUN_FAILED_TESTS"] = strconv.FormatBool(instance.Spec.RerunFailedTests)
	envVars["TEMPEST_RERUN_OVERRIDE_STATUS"] = strconv.FormatBool(instance.Spec.RerunOverrideStatus)
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

// GetEnvVarsConfigMapName returns the name of the environment variables ConfigMap for the given workflow step
func GetEnvVarsConfigMapName(instance *testv1beta1.Tempest, workflowStepNum int) string {
	return instance.Name + envVarsConfigMapInfix + strconv.Itoa(workflowStepNum)
}

// GetCustomDataConfigMapName returns the name of the custom data ConfigMap for the given workflow step
func GetCustomDataConfigMapName(instance *testv1beta1.Tempest, workflowStepNum int) string {
	return instance.Name + customDataConfigMapInfix + strconv.Itoa(workflowStepNum)
}
