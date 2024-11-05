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
	"time"

	"github.com/go-logr/logr"
	"github.com/openstack-k8s-operators/lib-common/modules/common"
	"github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	"github.com/openstack-k8s-operators/lib-common/modules/common/env"
	"github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	"github.com/openstack-k8s-operators/lib-common/modules/common/job"
	common_rbac "github.com/openstack-k8s-operators/lib-common/modules/common/rbac"
	testv1beta1 "github.com/openstack-k8s-operators/test-operator/api/v1beta1"
	"github.com/openstack-k8s-operators/test-operator/pkg/horizontest"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
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

//+kubebuilder:rbac:groups=test.openstack.org,resources=horizontests,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=test.openstack.org,resources=horizontests/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=test.openstack.org,resources=horizontests/finalizers,verbs=update;patch
//+kubebuilder:rbac:groups="",resources=pods,verbs=create;delete;get;list;patch;update;watch
//+kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;create;update;watch;patch

// Reconcile - HorizonTest
func (r *HorizonTestReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, _err error) {
	Log := r.GetLogger(ctx)

	// How much time should we wait before calling Reconcile loop when there is a failure
	requeueAfter := time.Second * 60

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
		)
		instance.Status.Conditions.Init(&cl)

		// Register overall status immediately to have an early feedback
		// e.g. in the cli
		return ctrl.Result{}, nil

	}

	if r.CompletedJobExists(ctx, instance, 0) {
		// The job created by the instance was completed. Release the lock
		// so that other instances can spawn a job.
		instance.Status.Conditions.MarkTrue(condition.DeploymentReadyCondition, condition.DeploymentReadyMessage)
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
		common.AppSelector: horizontest.ServiceName,
		"instanceName":     instance.Name,
		"operator":         "test-operator",

		// NOTE(lpiwowar):  This is a workaround since the Horizontest CR does not support
		//                  workflows. However, the label might be required by automation that
		//                  consumes the test-operator (e.g., ci-framework).
		"workflowStep": "0",
	}

	yamlResult, err := EnsureCloudsConfigMapExists(ctx, instance, helper, serviceLabels)

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

	// Create Job
	mountCerts := r.CheckSecretExists(ctx, instance, "combined-ca-bundle")

	mountKeys := false

	mountKubeconfig := false
	if len(instance.Spec.KubeconfigSecretName) != 0 {
		mountKubeconfig = true
	}

	// If the current job is executing the last workflow step -> do not create another job
	if r.JobExists(ctx, instance, 0) {
		return ctrl.Result{}, nil
	}

	// We are about to start job that spawns the pod with tests.
	// This lock ensures that there is always only one pod running.
	lockAcquired, err := r.AcquireLock(ctx, instance, helper, instance.Spec.Parallel)
	if !lockAcquired {
		Log.Info("Cannot acquire lock")
		return ctrl.Result{RequeueAfter: requeueAfter}, err
	}
	Log.Info("Lock acquired")

	// Prepare HorizonTest env vars
	envVars := r.PrepareHorizonTestEnvVars(instance)
	jobName := r.GetJobName(instance, 0)
	logsPVCName := r.GetPVCLogsName(instance, 0)
	containerImage, err := r.GetContainerImage(ctx, instance.Spec.ContainerImage, instance)
	if err != nil {
		return ctrl.Result{}, err
	}

	jobDef := horizontest.Job(
		instance,
		serviceLabels,
		jobName,
		logsPVCName,
		mountCerts,
		mountKeys,
		mountKubeconfig,
		envVars,
		containerImage,
	)
	horizontestJob := job.NewJob(
		jobDef,
		testv1beta1.ConfigHash,
		true,
		time.Duration(5)*time.Second,
		"",
	)

	ctrlResult, err = horizontestJob.DoJob(ctx, helper)
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
func (r *HorizonTestReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&testv1beta1.HorizonTest{}).
		Owns(&batchv1.Job{}).
		Owns(&corev1.Secret{}).
		Owns(&corev1.ConfigMap{}).
		Complete(r)
}

func (r *HorizonTestReconciler) PrepareHorizonTestEnvVars(
	instance *testv1beta1.HorizonTest,
) map[string]env.Setter {
	// Prepare env vars
	envVars := make(map[string]env.Setter)
	envVars["USE_EXTERNAL_FILES"] = env.SetValue("True")
	envVars["HORIZON_LOGS_DIR_NAME"] = env.SetValue("horizon")

	// Mandatory variables
	envVars["ADMIN_USERNAME"] = env.SetValue(instance.Spec.AdminUsername)
	envVars["ADMIN_PASSWORD"] = env.SetValue(instance.Spec.AdminPassword)
	envVars["DASHBOARD_URL"] = env.SetValue(instance.Spec.DashboardUrl)
	envVars["AUTH_URL"] = env.SetValue(instance.Spec.AuthUrl)
	envVars["REPO_URL"] = env.SetValue(instance.Spec.RepoUrl)
	envVars["HORIZON_REPO_BRANCH"] = env.SetValue(instance.Spec.HorizonRepoBranch)

	// Horizon specific configuration
	envVars["IMAGE_FILE"] = env.SetValue("/var/lib/horizontest/cirros-0.6.2-x86_64-disk.img")
	envVars["IMAGE_FILE_NAME"] = env.SetValue("cirros-0.6.2-x86_64-disk")
	envVars["IMAGE_URL"] = env.SetValue("http://download.cirros-cloud.net/0.6.2/cirros-0.6.2-x86_64-disk.img")
	envVars["PROJECT_NAME"] = env.SetValue("horizontest")
	envVars["USER_NAME"] = env.SetValue("horizontest")
	envVars["PASSWORD"] = env.SetValue("horizontest")
	envVars["FLAVOR_NAME"] = env.SetValue("m1.tiny")
	envVars["HORIZON_KEYS_FOLDER"] = env.SetValue("/etc/test_operator")
	envVars["HORIZONTEST_DEBUG_MODE"] = env.SetValue(r.GetDefaultBool(instance.Spec.Debug))

	return envVars
}
