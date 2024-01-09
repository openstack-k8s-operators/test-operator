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
	"github.com/openstack-k8s-operators/lib-common/modules/common/pvc"
	common_rbac "github.com/openstack-k8s-operators/lib-common/modules/common/rbac"
	"github.com/openstack-k8s-operators/lib-common/modules/common/util"
	testv1beta1 "github.com/openstack-k8s-operators/test-operator/api/v1beta1"
	"github.com/openstack-k8s-operators/test-operator/pkg/tempest"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	k8sresource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// TempestReconciler reconciles a Tempest object
type TempestReconciler struct {
	client.Client
	Kclient kubernetes.Interface
	Log     logr.Logger
	Scheme  *runtime.Scheme
}

// GetClient -
func (r *TempestReconciler) GetClient() client.Client {
	return r.Client
}

// GetLogger -
func (r *TempestReconciler) GetLogger() logr.Logger {
	return r.Log
}

// GetScheme -
func (r *TempestReconciler) GetScheme() *runtime.Scheme {
	return r.Scheme
}

func SecretExists(r *TempestReconciler, ctx context.Context, instance *testv1beta1.Tempest, SecretName string) bool {
	secret := &corev1.Secret{}
	err := r.Get(ctx, client.ObjectKey{Namespace: instance.Namespace, Name: SecretName}, secret)
	if err != nil && k8s_errors.IsNotFound(err) {
		return false
	} else {
		return true
	}
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

// Reconcile - Tempest
func (r *TempestReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, _err error) {
	_ = r.Log.WithValues("tempest", req.NamespacedName)

	// Fetch the Tempest instance
	instance := &testv1beta1.Tempest{}
	err := r.Client.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if k8s_errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected.
			// For additional cleanup logic use finalizers. Return and don't requeue.
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
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
	// Always patch the instance status when exiting this function so we can persist any changes.
	defer func() {
		err := helper.PatchInstance(ctx, instance)
		if err != nil {
			_err = err
			return
		}
	}()

	// If we're not deleting this and the service object doesn't have our finalizer, add it.
	if instance.DeletionTimestamp.IsZero() && controllerutil.AddFinalizer(instance, helper.GetFinalizer()) {
		return ctrl.Result{}, nil
	}

	//
	// initialize status
	//
	if instance.Status.Conditions == nil {
		instance.Status.Conditions = condition.Conditions{}
		// initialize conditions used later as Status=Unknown
		cl := condition.CreateList(
			condition.UnknownCondition(condition.InputReadyCondition, condition.InitReason, condition.InputReadyInitMessage),
			condition.UnknownCondition(condition.ServiceConfigReadyCondition, condition.InitReason, condition.ServiceConfigReadyInitMessage),
			condition.UnknownCondition(condition.DeploymentReadyCondition, condition.InitReason, condition.DeploymentReadyInitMessage),
			condition.UnknownCondition(condition.NetworkAttachmentsReadyCondition, condition.InitReason, condition.NetworkAttachmentsReadyInitMessage),
		)

		instance.Status.Conditions.Init(&cl)

		// Register overall status immediately to have an early feedback e.g. in the cli
		return ctrl.Result{}, nil
	}
	if instance.Status.NetworkAttachments == nil {
		instance.Status.NetworkAttachments = map[string][]string{}
	}

	// Handle service delete
	if !instance.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, instance, helper)
	}

	// Handle non-deleted clusters
	return r.reconcileNormal(ctx, instance, helper)
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

func (r *TempestReconciler) reconcileDelete(ctx context.Context, instance *testv1beta1.Tempest, helper *helper.Helper) (ctrl.Result, error) {
	r.Log.Info("Reconciling Service delete")

	// remove the finalizer
	controllerutil.RemoveFinalizer(instance, helper.GetFinalizer())
	r.Log.Info("Reconciled Service delete successfully")

	return ctrl.Result{}, nil
}

func (r *TempestReconciler) reconcileInit(
	ctx context.Context,
	instance *testv1beta1.Tempest,
	helper *helper.Helper,
	serviceLabels map[string]string,
	serviceAnnotations map[string]string,
) (ctrl.Result, error) {
	r.Log.Info("Reconciling Service init")

	r.Log.Info("Reconciled Service init successfully")
	return ctrl.Result{}, nil
}

func (r *TempestReconciler) reconcileUpdate(ctx context.Context, instance *testv1beta1.Tempest, helper *helper.Helper) (ctrl.Result, error) {
	r.Log.Info("Reconciling Service update")

	// TODO(slaweq): is that needed at all?

	r.Log.Info("Reconciled Service update successfully")
	return ctrl.Result{}, nil
}

func (r *TempestReconciler) reconcileUpgrade(ctx context.Context, instance *testv1beta1.Tempest, helper *helper.Helper) (ctrl.Result, error) {
	r.Log.Info("Reconciling Service upgrade")

	// TODO(slaweq): is that needed at all?

	r.Log.Info("Reconciled Service upgrade successfully")
	return ctrl.Result{}, nil
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
	}
	rbacResult, err := common_rbac.ReconcileRbac(ctx, helper, instance, rbacRules)
	if err != nil {
		return rbacResult, err
	} else if (rbacResult != ctrl.Result{}) {
		return rbacResult, nil
	}

	instance.Status.Conditions.MarkTrue(condition.InputReadyCondition, condition.InputReadyMessage)
	// run check OpenStack secret - end

	//
	// Create ConfigMaps and Secrets required as input for the Service and calculate an overall hash of hashes
	//

	//
	// create Configmap required for neutron input
	// - %-scripts configmap holding scripts to e.g. bootstrap the service
	// - %-config configmap holding minimal neutron config required to get the service up, user can add additional files to be added to the service
	// - parameters which has passwords gets added from the OpenStack secret via the init container
	//
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

	//
	// TODO check when/if Init, Update, or Upgrade should/could be skipped
	//

	serviceLabels := map[string]string{
		common.AppSelector: tempest.ServiceName,
	}

	// networks to attach to
	for _, netAtt := range instance.Spec.NetworkAttachments {
		_, err := nad.GetNADWithName(ctx, helper, netAtt, instance.Namespace)
		if err != nil {
			if k8s_errors.IsNotFound(err) {
				instance.Status.Conditions.Set(condition.FalseCondition(
					condition.NetworkAttachmentsReadyCondition,
					condition.RequestedReason,
					condition.SeverityInfo,
					condition.NetworkAttachmentsReadyWaitingMessage,
					netAtt))
				return ctrl.Result{RequeueAfter: time.Second * 10}, fmt.Errorf("network-attachment-definition %s not found", netAtt)
			}
			instance.Status.Conditions.Set(condition.FalseCondition(
				condition.NetworkAttachmentsReadyCondition,
				condition.ErrorReason,
				condition.SeverityWarning,
				condition.NetworkAttachmentsReadyErrorMessage,
				err.Error()))
			return ctrl.Result{}, err
		}
	}

	serviceAnnotations, err := nad.CreateNetworksAnnotation(instance.Namespace, instance.Spec.NetworkAttachments)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed create network annotation from %s: %w",
			instance.Spec.NetworkAttachments, err)
	}

	// Handle service init
	ctrlResult, err := r.reconcileInit(ctx, instance, helper, serviceLabels, serviceAnnotations)
	if err != nil {
		return ctrlResult, err
	} else if (ctrlResult != ctrl.Result{}) {
		return ctrlResult, nil
	}

	// Handle service update
	ctrlResult, err = r.reconcileUpdate(ctx, instance, helper)
	if err != nil {
		return ctrlResult, err
	} else if (ctrlResult != ctrl.Result{}) {
		return ctrlResult, nil
	}

	// Handle service upgrade
	ctrlResult, err = r.reconcileUpgrade(ctx, instance, helper)
	if err != nil {
		return ctrlResult, err
	} else if (ctrlResult != ctrl.Result{}) {
		return ctrlResult, nil
	}

	// verify if network attachment matches expectations
	networkReady, networkAttachmentStatus, err := nad.VerifyNetworkStatusFromAnnotation(ctx, helper, instance.Spec.NetworkAttachments, serviceLabels, 1)
	if err != nil {
		return ctrl.Result{}, err
	}

	instance.Status.NetworkAttachments = networkAttachmentStatus
	if networkReady {
		instance.Status.Conditions.MarkTrue(condition.NetworkAttachmentsReadyCondition, condition.NetworkAttachmentsReadyMessage)
	} else {
		err := fmt.Errorf("not all pods have interfaces with ips as configured in NetworkAttachments: %s", instance.Spec.NetworkAttachments)
		instance.Status.Conditions.Set(condition.FalseCondition(
			condition.NetworkAttachmentsReadyCondition,
			condition.ErrorReason,
			condition.SeverityWarning,
			condition.NetworkAttachmentsReadyErrorMessage,
			err.Error()))

		return ctrl.Result{}, err
	}

	// Create pvc
	testOperatorPvcDef := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-operator-logs",
			Namespace: instance.Namespace,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
				corev1.ReadWriteMany,
			},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: k8sresource.MustParse("1Gi"),
				},
			},
		},
	}

	timeDuration, _ := time.ParseDuration("2m")
	testOperatorPvc := pvc.NewPvc(testOperatorPvcDef, timeDuration)
	ctrlResult, err = testOperatorPvc.CreateOrPatch(ctx, helper)
	if err != nil {
		return ctrlResult, err
	} else if (ctrlResult != ctrl.Result{}) {
		return ctrlResult, nil
	}

	// Define a new Job object
	mountCerts := SecretExists(r, ctx, instance, "combined-ca-bundle")
	jobDef := tempest.Job(instance, serviceLabels, mountCerts)
	tempestJob := job.NewJob(
		jobDef,
		testv1beta1.ConfigHash,
		false,
		time.Duration(5)*time.Second,
		"",
	)

	ctrlResult, err = tempestJob.DoJob(ctx, helper)
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
	r.Log.Info("Reconciled Service successfully")
	return ctrl.Result{}, nil
}

func getDefaultBool(variable bool) string {
	if variable {
		return "true"
	} else {
		return "false"
	}
}

func getDefaultInt(variable int64) string {
	if variable != -1 {
		return strconv.FormatInt(variable, 10)
	} else {
		return ""
	}
}

func setTempestConfigVars(envVars map[string]string,
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
		envVars[key] = getDefaultBool(value)
	}

	// Int
	envVars["TEMPEST_CONCURRENCY"] = getDefaultInt(tempestRun.Concurrency)

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

func setTempestconfConfigVars(envVars map[string]string,
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
		envVars[key] = getDefaultBool(value)
	}

	// Int
	tempestconfIntEnvVars := make(map[string]int64)
	tempestconfIntEnvVars = map[string]int64{
		"TEMPESTCONF_TIMEOUT":         tempestconfRun.Timeout,
		"TEMPESTCONF_FLAVOR_MIN_MEM":  tempestconfRun.FlavorMinMem,
		"TEMPESTCONF_FLAVOR_MIN_DISK": tempestconfRun.FlavorMinDisk,
	}

	for key, value := range tempestconfIntEnvVars {
		envVars[key] = getDefaultInt(value)
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

// generateServiceConfigMaps - create create configmaps which hold scripts and service configuration
// TODO add DefaultConfigOverwrite
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

	setTempestConfigVars(envVars, customData, instance.Spec.TempestRun, ctx)
	setTempestconfConfigVars(envVars, customData, instance.Spec.TempestconfRun)

	/* Tempestconf - end */
	cms := []util.Template{
		// ScriptsConfigMap
		{
			Name:         fmt.Sprintf("%s-scripts", instance.Name),
			Namespace:    instance.Namespace,
			Type:         util.TemplateTypeScripts,
			InstanceType: instance.Kind,
			Labels:       cmLabels,
		},
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
