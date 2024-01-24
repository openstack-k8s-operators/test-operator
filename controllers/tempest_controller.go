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
	"time"

	"github.com/openstack-k8s-operators/lib-common/modules/common"
	"github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	"github.com/openstack-k8s-operators/lib-common/modules/common/configmap"
	"github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	"github.com/openstack-k8s-operators/lib-common/modules/common/job"
	"github.com/openstack-k8s-operators/lib-common/modules/common/labels"
	common_rbac "github.com/openstack-k8s-operators/lib-common/modules/common/rbac"
	"github.com/openstack-k8s-operators/lib-common/modules/common/util"
	testv1beta1 "github.com/openstack-k8s-operators/test-operator/api/v1beta1"
	"github.com/openstack-k8s-operators/test-operator/pkg/tempest"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type TempestReconciler struct {
	Reconciler
}

// +kubebuilder:rbac:groups=test.openstack.org,resources=tempests,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=test.openstack.org,resources=tempests/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=test.openstack.org,resources=tempests/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete;
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete;
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;patch;update;delete;
// +kubebuilder:rbac:groups=k8s.cni.cncf.io,resources=network-attachment-definitions,verbs=get;list;watch

// service account, role, rolebinding
// +kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;list;watch;create;update
// +kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=roles,verbs=get;list;watch;create;update
// +kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=rolebindings,verbs=get;list;watch;create;update
// service account permissions that are needed to grant permission to the above
// +kubebuilder:rbac:groups="security.openshift.io",resourceNames=anyuid;privileged,resources=securitycontextconstraints,verbs=use
// +kubebuilder:rbac:groups="",resources=pods,verbs=create;delete;get;list;patch;update;watch
// +kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;create;update;watch;patch

// Reconcile - Tempest
func (r *TempestReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, _err error) {
	_ = r.Log.WithValues("tempest", req.NamespacedName)

	// Fetch the Tempest instance
	instance := &testv1beta1.Tempest{}
	err := r.Client.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if k8s_errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	logging := log.FromContext(ctx)
	if r.CompletedJobExists(ctx, instance) {
		// The job created by the instance was completed. Release the lock
		// so that other instances can spawn a job.
		logging.Info("Job completed")
		r.ReleaseLock(ctx, instance)
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
		)

		instance.Status.Conditions.Init(&cl)

		// Register overall status immediately to have an early feedback
		// e.g. in the cli
		return ctrl.Result{}, nil
	}

	// Handle service delete
	if !instance.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, instance, helper)
	}

	// Handle non-deleted clusters
	return r.reconcileNormal(ctx, instance, helper)
}

func (r *TempestReconciler) reconcileNormal(ctx context.Context, instance *testv1beta1.Tempest, helper *helper.Helper) (ctrl.Result, error) {
	r.Log.Info("Reconciling Service")

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

	// Generate ConfigMaps
	err = r.generateServiceConfigMaps(ctx, helper, instance)
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

	//
	// TODO check when/if Init, Update, or Upgrade should/could be skipped
	//

	// Create PersistentVolumeClaim
	ctrlResult, err := r.EnsureLogsPVCExists(ctx, instance, helper, instance.Name)
	if err != nil {
		return ctrlResult, err
	} else if (ctrlResult != ctrl.Result{}) {
		return ctrlResult, nil
	}
	// Create PersistentVolumeClaim - end

	// Create a new job
	serviceLabels := map[string]string{
		common.AppSelector: tempest.ServiceName,
	}

	mountCerts := r.CheckSecretExists(ctx, instance, "combined-ca-bundle")
	jobDef := tempest.Job(instance, serviceLabels, mountCerts)
	tempestJob := job.NewJob(
		jobDef,
		testv1beta1.ConfigHash,
		false,
		time.Duration(5)*time.Second,
		"",
	)

	// If there is a job that is completed do not try to create
	// another one
	if r.JobExists(ctx, instance) {
		return ctrl.Result{}, nil
	}

	// We are about to start job that spawns the pod with tests.
	// This lock ensures that there is always only one pod running.
	logging := log.FromContext(ctx)
	if !r.AcquireLock(ctx, instance, helper, instance.Spec.Parallel) {
		logging.Info("Can not acquire lock")
		requeueAfter := time.Second * 60
		return ctrl.Result{RequeueAfter: requeueAfter}, nil
	} else {
		logging.Info("Lock acquired")
	}

	ctrlResult, err = tempestJob.DoJob(ctx, helper)
	if err != nil {
		// Creation of the tempest job was not successfull.
		// Release the lock and allow other controllers to spawn
		// a job.
		r.ReleaseLock(ctx, instance)
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

	r.Log.Info("Reconciled Service successfully")
	return ctrl.Result{}, nil
}

func (r *TempestReconciler) reconcileDelete(ctx context.Context, instance *testv1beta1.Tempest, helper *helper.Helper) (ctrl.Result, error) {
	r.Log.Info("Reconciling Service delete")

	// remove the finalizer
	controllerutil.RemoveFinalizer(instance, helper.GetFinalizer())

	r.Log.Info("Reconciled Service delete successfully")

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
	tempestRun *testv1beta1.TempestRunSpec,
	ctx context.Context) {

	testOperatorDir := "/etc/test_operator/"
	if tempestRun == nil {
		includeListFile := "include.txt"
		customData[includeListFile] = "tempest.api.identity.v3"
		envVars["TEMPEST_INCLUDE_LIST"] = testOperatorDir + includeListFile
		envVars["TEMPEST_PARALLEL"] = "true"
		return
	}

	// Files
	if len(tempestRun.WorkerFile) != 0 {
		workerFile := "worker_file.yaml"
		customData[workerFile] = tempestRun.WorkerFile
		envVars["TEMPEST_WORKER_FILE"] = testOperatorDir + workerFile
	}

	if len(tempestRun.IncludeList) != 0 {
		includeListFile := "include.txt"
		customData[includeListFile] = tempestRun.IncludeList
		envVars["TEMPEST_INCLUDE_LIST"] = testOperatorDir + includeListFile
	}

	if len(tempestRun.ExcludeList) != 0 {
		excludeListFile := "exclude.txt"
		customData[excludeListFile] = tempestRun.ExcludeList
		envVars["TEMPEST_EXCLUDE_LIST"] = testOperatorDir + excludeListFile
	}

	// Bool
	tempestBoolEnvVars := make(map[string]bool)
	tempestBoolEnvVars = map[string]bool{
		"TEMPEST_SERIAL":     tempestRun.Serial,
		"TEMPEST_PARALLEL":   tempestRun.Parallel,
		"TEMPEST_SMOKE":      tempestRun.Smoke,
		"USE_EXTERNAL_FILES": true,
	}

	for key, value := range tempestBoolEnvVars {
		envVars[key] = r.GetDefaultBool(value)
	}

	// Int
	envVars["TEMPEST_CONCURRENCY"] = r.GetDefaultInt(tempestRun.Concurrency)

	// Dictionary
	for _, externalPluginDictionary := range tempestRun.ExternalPlugin {
		envVars["TEMPEST_EXTERNAL_PLUGIN_GIT_URL"] += externalPluginDictionary.Repository + ","

		if len(externalPluginDictionary.ChangeRepository) == 0 || len(externalPluginDictionary.ChangeRefspec) == 0 {
			envVars["TEMPEST_EXTERNAL_PLUGIN_CHANGE_URL"] += "-,"
			envVars["TEMPEST_EXTERNAL_PLUGIN_REFSPEC"] += "-,"
			continue
		}

		envVars["TEMPEST_EXTERNAL_PLUGIN_CHANGE_URL"] += externalPluginDictionary.ChangeRepository + ","
		envVars["TEMPEST_EXTERNAL_PLUGIN_REFSPEC"] += externalPluginDictionary.ChangeRefspec + ","
	}
}

func (r *TempestReconciler) setTempestconfConfigVars(envVars map[string]string,
	customData map[string]string,
	tempestconfRun *testv1beta1.TempestconfRunSpec) {

	if tempestconfRun == nil {
		envVars["TEMPESTCONF_CREATE"] = "true"
		envVars["TEMPESTCONF_OVERRIDES"] = "identity.v3_endpoint_type public"
		return
	}

	// Files
	testOperatorDir := "/etc/test_operator/"
	if len(tempestconfRun.DeployerInput) != 0 {
		deployerInputFile := "deployer_input.yaml"
		customData[deployerInputFile] = tempestconfRun.DeployerInput
		envVars["TEMPESTCONF_DEPLOYER_INPUT"] = testOperatorDir + deployerInputFile
	}

	if len(tempestconfRun.TestAccounts) != 0 {
		accountsFile := "accounts.yaml"
		customData[accountsFile] = tempestconfRun.TestAccounts
		envVars["TEMPESTCONF_TEST_ACCOUNTS"] = testOperatorDir + accountsFile
	}

	if len(tempestconfRun.Profile) != 0 {
		profileFile := "profile.yaml"
		customData[profileFile] = tempestconfRun.Profile
		envVars["TEMPESTCONF_PROFILE"] = testOperatorDir + profileFile
	}

	// Bool
	tempestconfBoolEnvVars := make(map[string]bool)
	tempestconfBoolEnvVars = map[string]bool{
		"TEMPESTCONF_CREATE":              tempestconfRun.Create,
		"TEMPESTCONF_COLLECT_TIMING":      tempestconfRun.CollectTiming,
		"TEMPESTCONF_INSECURE":            tempestconfRun.Insecure,
		"TEMPESTCONF_NO_DEFAULT_DEPLOYER": tempestconfRun.NoDefaultDeployer,
		"TEMPESTCONF_DEBUG":               tempestconfRun.Debug,
		"TEMPESTCONF_VERBOSE":             tempestconfRun.Verbose,
		"TEMPESTCONF_NON_ADMIN":           tempestconfRun.NonAdmin,
		"TEMPESTCONF_RETRY_IMAGE":         tempestconfRun.RetryImage,
		"TEMPESTCONF_CONVERT_TO_RAW":      tempestconfRun.ConvertToRaw,
	}

	for key, value := range tempestconfBoolEnvVars {
		envVars[key] = r.GetDefaultBool(value)
	}

	// Int
	tempestconfIntEnvVars := make(map[string]int64)
	tempestconfIntEnvVars = map[string]int64{
		"TEMPESTCONF_TIMEOUT":         tempestconfRun.Timeout,
		"TEMPESTCONF_FLAVOR_MIN_MEM":  tempestconfRun.FlavorMinMem,
		"TEMPESTCONF_FLAVOR_MIN_DISK": tempestconfRun.FlavorMinDisk,
	}

	for key, value := range tempestconfIntEnvVars {
		envVars[key] = r.GetDefaultInt(value)
	}

	// String
	envVars["TEMPESTCONF_OUT"] = tempestconfRun.Out
	envVars["TEMPESTCONF_CREATE_ACCOUNTS_FILE"] = tempestconfRun.CreateAccountsFile
	envVars["TEMPESTCONF_GENERATE_PROFILE"] = tempestconfRun.GenerateProfile
	envVars["TEMPESTCONF_IMAGE_DISK_FORMAT"] = tempestconfRun.ImageDiskFormat
	envVars["TEMPESTCONF_IMAGE"] = tempestconfRun.Image
	envVars["TEMPESTCONF_NETWORK_ID"] = tempestconfRun.NetworkID
	envVars["TEMPESTCONF_APPEND"] = tempestconfRun.Append
	envVars["TEMPESTCONF_REMOVE"] = tempestconfRun.Remove
	envVars["TEMPESTCONF_OVERRIDES"] = tempestconfRun.Overrides
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
) error {
	// Create/update configmaps from template
	cmLabels := labels.GetLabels(instance, labels.GetGroupLabel(tempest.ServiceName), map[string]string{})

	templateParameters := make(map[string]interface{})
	customData := make(map[string]string)
	envVars := make(map[string]string)

	r.setTempestConfigVars(envVars, customData, instance.Spec.TempestRun, ctx)
	r.setTempestconfConfigVars(envVars, customData, instance.Spec.TempestconfRun)

	cms := []util.Template{
		// ConfigMap
		{
			Name:          fmt.Sprintf("%s-config-data", instance.Name),
			Namespace:     instance.Namespace,
			InstanceType:  instance.Kind,
			Labels:        cmLabels,
			ConfigOptions: templateParameters,
			CustomData:    customData,
		},
		// configMap - EnvVars
		{
			Name:          fmt.Sprintf("%s-env-vars", instance.Name),
			Namespace:     instance.Namespace,
			InstanceType:  instance.Kind,
			Labels:        cmLabels,
			ConfigOptions: templateParameters,
			CustomData:    envVars,
		},
	}

	return configmap.EnsureConfigMaps(ctx, h, instance, cms, nil)
}
