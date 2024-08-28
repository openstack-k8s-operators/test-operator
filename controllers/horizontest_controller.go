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
	"github.com/openstack-k8s-operators/lib-common/modules/common/configmap"
	"github.com/openstack-k8s-operators/lib-common/modules/common/env"
	"github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	"github.com/openstack-k8s-operators/lib-common/modules/common/job"
	common_rbac "github.com/openstack-k8s-operators/lib-common/modules/common/rbac"
	"github.com/openstack-k8s-operators/lib-common/modules/common/util"
	testv1beta1 "github.com/openstack-k8s-operators/test-operator/api/v1beta1"
	"github.com/openstack-k8s-operators/test-operator/pkg/horizontest"
	"gopkg.in/yaml.v3"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// HorizonTestReconciler reconciles a HorizonTest object
type HorizonTestReconciler struct {
	Reconciler
}

// GetLogger returns a logger object with a prefix of "controller.name" and additional controller context fields
func (r *HorizonTestReconciler) GetLogger(ctx context.Context) logr.Logger {
	return log.FromContext(ctx).WithName("Controllers").WithName("Tobiko")
}

//+kubebuilder:rbac:groups=test.openstack.org,resources=horizontests,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=test.openstack.org,resources=horizontests/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=test.openstack.org,resources=horizontests/finalizers,verbs=update;patch
//+kubebuilder:rbac:groups="",resources=pods,verbs=create;delete;get;list;patch;update;watch
//+kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;create;update;watch;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the HorizonTest object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *HorizonTestReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
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

	serviceLabels := map[string]string{
		common.AppSelector: horizontest.ServiceName,
		"instanceName":     instance.Name,
		"operator":         "test-operator",

		// NOTE(lpiwowar):  This is a workaround since the Horizontest CR does not support
		//                  workflows. However, the label might be required by automation that
		//                  consumes the test-operator (e.g., ci-framework).
		"workflowStep":     "0",
	}

	result, err := r.EnsureHorizonTestCloudsYAML(ctx, instance, helper, serviceLabels)

	if err != nil {
		return result, err
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
	envVars := r.PrepareHorizonTestEnvVars(ctx, serviceLabels, instance, helper)
	jobName := r.GetJobName(instance, 0)
	logsPVCName := r.GetPVCLogsName(instance)
	jobDef := horizontest.Job(
		instance,
		serviceLabels,
		jobName,
		logsPVCName,
		mountCerts,
		mountKeys,
		mountKubeconfig,
		envVars,
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

// Horizon requires password value to be present in clouds.yaml
// This code ensures that we set a default value of 12345678 when
// password value is missing in the clouds.yaml
func (r *HorizonTestReconciler) EnsureHorizonTestCloudsYAML(ctx context.Context, instance client.Object, helper *helper.Helper, labels map[string]string) (ctrl.Result, error) {
	cm, _, _ := configmap.GetConfigMap(ctx, helper, instance, "openstack-config", time.Second*10)
	result := make(map[string]interface{})

	err := yaml.Unmarshal([]byte(cm.Data["clouds.yaml"]), &result)
	if err != nil {
		return ctrl.Result{}, err
	}

	clouds := result["clouds"].(map[string]interface{})
	default_value := clouds["default"].(map[string]interface{})
	auth := default_value["auth"].(map[string]interface{})

	if _, ok := auth["password"].(string); !ok {
		auth["password"] = "12345678"
	}

	yamlString, err := yaml.Marshal(result)
	if err != nil {
		return ctrl.Result{}, err
	}

	cms := []util.Template{
		{
			Name:      "horizontest-clouds-config",
			Namespace: instance.GetNamespace(),
			Labels:    labels,
			CustomData: map[string]string{
				"clouds.yaml": string(yamlString),
			},
		},
	}
	configmap.EnsureConfigMaps(ctx, helper, instance, cms, nil)

	return ctrl.Result{}, nil
}

func (r *HorizonTestReconciler) PrepareHorizonTestEnvVars(
	ctx context.Context,
	labels map[string]string,
	instance *testv1beta1.HorizonTest,
	helper *helper.Helper,
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

	return envVars
}
