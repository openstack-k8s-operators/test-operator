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

	"github.com/go-logr/logr"
	"github.com/openstack-k8s-operators/lib-common/modules/common"
	"github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	"github.com/openstack-k8s-operators/lib-common/modules/common/env"
	"github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	testv1beta1 "github.com/openstack-k8s-operators/test-operator/api/v1beta1"
	"github.com/openstack-k8s-operators/test-operator/internal/horizontest"
	corev1 "k8s.io/api/core/v1"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// HorizonTestReconciler reconciles a HorizonTest object
type HorizonTestReconciler struct {
	Reconciler
}

// GetLogger returns a logger object with a prefix of "controller.name" and additional controller context fields
func (r *HorizonTestReconciler) GetLogger(ctx context.Context) logr.Logger {
	return log.FromContext(ctx).WithName("Controllers").WithName("HorizonTest")
}

// +kubebuilder:rbac:groups=test.openstack.org,resources=horizontests,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=test.openstack.org,resources=horizontests/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=test.openstack.org,resources=horizontests/finalizers,verbs=update;patch
// +kubebuilder:rbac:groups=k8s.cni.cncf.io,resources=network-attachment-definitions,verbs=get;list;watch
// +kubebuilder:rbac:groups="security.openshift.io",resourceNames=anyuid;privileged;nonroot;nonroot-v2,resources=securitycontextconstraints,verbs=use
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete;
// +kubebuilder:rbac:groups="",resources=pods,verbs=create;delete;get;list;patch;update;watch
// +kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;create;update;watch;patch;delete

// Reconcile - HorizonTest
func (r *HorizonTestReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, _err error) {
	Log := r.GetLogger(ctx)
	instance := &testv1beta1.HorizonTest{}
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
		)
		instance.Status.Conditions.Init(&cl)

		// Register overall status immediately to have an early feedback
		// e.g. in the cli
		return ctrl.Result{}, nil
	}
	instance.Status.ObservedGeneration = instance.Generation

	workflowLength := 0
	nextAction, nextWorkflowStep, err := r.NextAction(ctx, instance, workflowLength)

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
		return ctrl.Result{}, ErrReceivedUnexpectedAction
	}

	serviceLabels := map[string]string{
		common.AppSelector: horizontest.ServiceName,
		instanceNameLabel:  instance.Name,
		operatorNameLabel:  "test-operator",

		// NOTE(lpiwowar):  This is a workaround since the Horizontest CR does not support
		//                  workflows. However, the label might be required by automation that
		//                  consumes the test-operator (e.g., ci-framework).
		workflowStepLabel: "0",
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

	// Create Pod
	mountCerts := r.CheckSecretExists(ctx, instance, "combined-ca-bundle")
	mountKubeconfig := len(instance.Spec.KubeconfigSecretName) != 0

	// Prepare HorizonTest env vars
	envVars := r.PrepareHorizonTestEnvVars(instance)
	podName := r.GetPodName(instance, 0)
	logsPVCName := r.GetPVCLogsName(instance, 0)
	containerImage, err := r.GetContainerImage(ctx, instance)
	if err != nil {
		return ctrl.Result{}, err
	}

	podDef := horizontest.Pod(
		instance,
		serviceLabels,
		podName,
		logsPVCName,
		mountCerts,
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
func (r *HorizonTestReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&testv1beta1.HorizonTest{}).
		Owns(&corev1.Pod{}).
		Owns(&corev1.Secret{}).
		Owns(&corev1.ConfigMap{}).
		Complete(r)
}

// PrepareHorizonTestEnvVars prepares environment variables for HorizonTest execution
func (r *HorizonTestReconciler) PrepareHorizonTestEnvVars(
	instance *testv1beta1.HorizonTest,
) map[string]env.Setter {
	// Prepare env vars
	envVars := make(map[string]env.Setter)

	// Bool
	SetBoolEnvVars(envVars, map[string]bool{
		"HORIZONTEST_DEBUG_MODE": instance.Spec.Debug,
	})

	// String
	SetStringEnvVars(envVars, map[string]string{
		"USE_EXTERNAL_FILES":    "True",
		"HORIZON_LOGS_DIR_NAME": "horizon",

		// Mandatory variables
		"ADMIN_USERNAME":      instance.Spec.AdminUsername,
		"ADMIN_PASSWORD":      instance.Spec.AdminPassword,
		"DASHBOARD_URL":       instance.Spec.DashboardUrl,
		"AUTH_URL":            instance.Spec.AuthUrl,
		"REPO_URL":            instance.Spec.RepoUrl,
		"HORIZON_REPO_BRANCH": instance.Spec.HorizonRepoBranch,

		// Horizon specific configuration
		"IMAGE_FILE":          "/var/lib/horizontest/cirros-0.6.2-x86_64-disk.img",
		"IMAGE_FILE_NAME":     "cirros-0.6.2-x86_64-disk",
		"IMAGE_URL":           "http://download.cirros-cloud.net/0.6.2/cirros-0.6.2-x86_64-disk.img",
		"PROJECT_NAME":        "horizontest",
		"USER_NAME":           "horizontest",
		"PASSWORD":            "horizontest",
		"FLAVOR_NAME":         "m1.tiny",
		"HORIZON_KEYS_FOLDER": "/etc/test_operator",
		"EXTRA_FLAG":          instance.Spec.ExtraFlag,
		"PROJECT_NAME_XPATH":  instance.Spec.ProjectNameXpath,
	})

	return envVars
}
