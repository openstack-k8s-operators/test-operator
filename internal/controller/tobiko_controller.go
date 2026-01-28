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
	"fmt"
	"strconv"

	"github.com/go-logr/logr"
	"github.com/openstack-k8s-operators/lib-common/modules/common"
	"github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	"github.com/openstack-k8s-operators/lib-common/modules/common/configmap"
	"github.com/openstack-k8s-operators/lib-common/modules/common/env"
	"github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	"github.com/openstack-k8s-operators/lib-common/modules/common/util"
	testv1beta1 "github.com/openstack-k8s-operators/test-operator/api/v1beta1"
	"github.com/openstack-k8s-operators/test-operator/internal/tobiko"
	corev1 "k8s.io/api/core/v1"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
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

// +kubebuilder:rbac:groups=test.openstack.org,resources=tobikoes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=test.openstack.org,resources=tobikoes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=test.openstack.org,resources=tobikoes/finalizers,verbs=update;patch
// +kubebuilder:rbac:groups=k8s.cni.cncf.io,resources=network-attachment-definitions,verbs=get;list;watch
// +kubebuilder:rbac:groups="security.openshift.io",resourceNames=anyuid;privileged;nonroot;nonroot-v2,resources=securitycontextconstraints,verbs=use
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete;
// +kubebuilder:rbac:groups="",resources=pods,verbs=create;delete;get;list;patch;update;watch
// +kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;create;update;watch;patch;delete

// Reconcile - Tobiko
func (r *TobikoReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, _err error) {
	Log := r.GetLogger(ctx)

	instance := &testv1beta1.Tobiko{}
	err := r.Client.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if k8s_errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
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

	// initialize status
	isNewInstance := instance.Status.Conditions == nil
	if isNewInstance {
		instance.Status.Conditions = condition.Conditions{}
	}

	// Save a copy of the conditions so that we can restore the LastTransitionTime
	// when a condition's state doesn't change.
	savedConditions := instance.Status.Conditions.DeepCopy()

	// Always patch the instance status when exiting this function so we
	// can persist any changes.
	defer func() {
		// Don't update the status, if reconciler Panics
		if r := recover(); r != nil {
			Log.Info(fmt.Sprintf("panic during reconcile %v\n", r))
			panic(r)
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
	instance.Status.ObservedGeneration = instance.Generation

	if instance.Status.NetworkAttachments == nil {
		instance.Status.NetworkAttachments = map[string][]string{}
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

		if instance.Status.Conditions.AllSubConditionIsTrue() {
			instance.Status.Conditions.MarkTrue(condition.ReadyCondition, condition.ReadyMessage)
		}

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
		// Confirm that we still hold the lock. This needs to be checked in order
		// to prevent situation when somebody / something deleted the lock and it
		// got claimed by another instance.
		lockAcquired, err := r.AcquireLock(ctx, instance, helper, instance.Spec.Parallel)
		if !lockAcquired {
			Log.Error(err, ErrConfirmLockOwnership, testOperatorLockName)
			return ctrl.Result{RequeueAfter: RequeueAfterValue}, err
		}

		Log.Info(fmt.Sprintf(InfoCreatingNextPod, nextWorkflowStep))

	default:
		return ctrl.Result{}, ErrReceivedUnexpectedAction
	}

	serviceLabels := map[string]string{
		common.AppSelector: tobiko.ServiceName,
		workflowStepLabel:  strconv.Itoa(nextWorkflowStep),
		instanceNameLabel:  instance.Name,
		operatorNameLabel:  "test-operator",
	}

	yamlResult, err := EnsureCloudsConfigMapExists(
		ctx,
		instance,
		helper,
		serviceLabels,
		instance.Spec.OpenStackConfigMap,
	)

	if err != nil {
		return yamlResult, err
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

	serviceAnnotations, ctrlResult, err := r.EnsureNetworkAttachments(
		ctx,
		Log,
		helper,
		instance.Spec.NetworkAttachments,
		instance.Namespace,
		&instance.Status.Conditions,
	)
	if err != nil || (ctrlResult != ctrl.Result{}) {
		return ctrlResult, err
	}

	// NetworkAttachments
	ctrlResult, err = r.VerifyNetworkAttachments(
		ctx,
		helper,
		instance,
		instance.Spec.NetworkAttachments,
		serviceLabels,
		nextWorkflowStep,
		&instance.Status.Conditions,
		&instance.Status.NetworkAttachments,
	)
	if err != nil || (ctrlResult != ctrl.Result{}) {
		return ctrlResult, err
	}

	// Create Pod
	mountCerts := r.CheckSecretExists(ctx, instance, "combined-ca-bundle")

	mountKeys := false
	if (len(instance.Spec.PublicKey) == 0) || (len(instance.Spec.PrivateKey) == 0) {
		Log.Info("Both values privateKey and publicKey need to be specified. Keys not mounted.")
	} else {
		mountKeys = true
	}

	mountKubeconfig := len(instance.Spec.KubeconfigSecretName) != 0

	// Prepare Tobiko env vars
	envVars := r.PrepareTobikoEnvVars(ctx, serviceLabels, instance, helper, nextWorkflowStep)
	podName := r.GetPodName(instance, nextWorkflowStep)
	logsPVCName := r.GetPVCLogsName(instance, workflowStepNum)
	containerImage, err := r.GetContainerImage(ctx, instance)
	if err != nil {
		return ctrl.Result{}, err
	}

	podDef := tobiko.Pod(
		instance,
		serviceLabels,
		serviceAnnotations,
		podName,
		logsPVCName,
		mountCerts,
		mountKeys,
		mountKubeconfig,
		envVars,
		containerImage,
	)

	ctrlResult, err = r.CreatePod(ctx, *helper, podDef)
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
	// create Pod - end

	if instance.Status.Conditions.AllSubConditionIsTrue() {
		instance.Status.Conditions.MarkTrue(condition.ReadyCondition, condition.ReadyMessage)
	}

	Log.Info("Reconciled Service successfully")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *TobikoReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&testv1beta1.Tobiko{}).
		Owns(&corev1.Pod{}).
		Owns(&corev1.Secret{}).
		Owns(&corev1.ConfigMap{}).
		Complete(r)
}

// PrepareTobikoEnvVars prepares environment variables for a single workflow step
func (r *TobikoReconciler) PrepareTobikoEnvVars(
	ctx context.Context,
	labels map[string]string,
	instance *testv1beta1.Tobiko,
	helper *helper.Helper,
	workflowStepNum int,
) map[string]env.Setter {
	// Prepare env vars
	envVars := make(map[string]env.Setter)

	// Bool
	SetBoolEnvVars(envVars, map[string]bool{
		"TOBIKO_DEBUG_MODE":     instance.Spec.Debug,
		"TOBIKO_PREVENT_CREATE": instance.Spec.PreventCreate,
	})

	// Note(kstrenko): Remove after the TCIB is updated and takes bool
	if instance.Spec.PreventCreate {
		envVars["TOBIKO_PREVENT_CREATE"] = env.SetValue("True")
	} else {
		envVars["TOBIKO_PREVENT_CREATE"] = env.SetValue("")
	}

	// String
	SetStringEnvVars(envVars, map[string]string{
		"USE_EXTERNAL_FILES":    "True",
		"TOBIKO_LOGS_DIR_NAME":  r.GetPodName(instance, workflowStepNum),
		"TOBIKO_TESTENV":        instance.Spec.Testenv,
		"TOBIKO_VERSION":        instance.Spec.Version,
		"TOBIKO_PYTEST_ADDOPTS": instance.Spec.PytestAddopts,
		"TOBIKO_KEYS_FOLDER":    "/etc/test_operator",
	})

	numProcesses := instance.Spec.NumProcesses
	if numProcesses > 0 {
		envVars["TOX_NUM_PROCESSES"] = env.SetValue(strconv.Itoa(int(numProcesses)))
	}

	if instance.Spec.Patch != (testv1beta1.PatchType{}) {
		SetStringEnvVars(envVars, map[string]string{
			"TOBIKO_PATCH_REPOSITORY": instance.Spec.Patch.Repository,
			"TOBIKO_PATCH_REFSPEC":    instance.Spec.Patch.Refspec,
		})
	}

	// Prepare custom data
	customData := make(map[string]string)
	customData["tobiko.conf"] = instance.Spec.Config

	privateKeyData := make(map[string]string)
	privateKeyData["id_ecdsa"] = instance.Spec.PrivateKey

	publicKeyData := make(map[string]string)
	publicKeyData["id_ecdsa.pub"] = instance.Spec.PublicKey

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
