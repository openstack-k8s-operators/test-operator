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
	"strconv"
	"time"

	"github.com/openstack-k8s-operators/lib-common/modules/common"
	"github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	"github.com/openstack-k8s-operators/lib-common/modules/common/configmap"
	"github.com/openstack-k8s-operators/lib-common/modules/common/env"
	"github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	"github.com/openstack-k8s-operators/lib-common/modules/common/job"
	common_rbac "github.com/openstack-k8s-operators/lib-common/modules/common/rbac"
	"github.com/openstack-k8s-operators/lib-common/modules/common/util"
	testv1beta1 "github.com/openstack-k8s-operators/test-operator/api/v1beta1"
	"github.com/openstack-k8s-operators/test-operator/pkg/tobiko"
	"gopkg.in/yaml.v2"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// TobikoReconciler reconciles a Tobiko object
type TobikoReconciler struct {
	Reconciler
}

//+kubebuilder:rbac:groups=test.openstack.org,resources=tobikoes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=test.openstack.org,resources=tobikoes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=test.openstack.org,resources=tobikoes/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=pods,verbs=create;delete;get;list;patch;update;watch
//+kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;create;update;watch;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *TobikoReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	// How much time should we wait before calling Reconcile loop when there is a failure
	requeueAfter := time.Second * 60

	logging := log.FromContext(ctx)
	instance := &testv1beta1.Tobiko{}
	err := r.Client.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if k8s_errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Check whether the user wants to execute workflow
	workflowActive := false
	if len(instance.Spec.Workflow) > 0 {
		workflowActive = true
	}

	helper, err := helper.NewHelper(
		instance,
		r.Client,
		r.Kclient,
		r.Scheme,
		r.Log,
	)

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
		logging.Info("Job completed")
		r.ReleaseLock(ctx, instance)
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

	instance.Status.Conditions.MarkTrue(condition.InputReadyCondition, condition.InputReadyMessage)

	serviceLabels := map[string]string{
		common.AppSelector: tobiko.ServiceName,
		"workflowStep":     strconv.Itoa(externalWorkflowCounter),
		"instanceName":     instance.Name,
		"operator":         "test-operator",
	}

	result, err := r.EnsureTobikoCloudsYAML(ctx, instance, helper, serviceLabels)

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
	if (len(instance.Spec.PublicKey) == 0) || (len(instance.Spec.PrivateKey) == 0) {
		logging.Info("Both values privateKey and publicKey need to be specified. Keys not mounted.")
	} else {
		mountKeys = true
	}

	mountKubeconfig := false
	if len(instance.Spec.KubeconfigSecretName) != 0 {
		mountKubeconfig = true
	}

	// If the current job is executing the last workflow step -> do not create another job
	if workflowActive && externalWorkflowCounter >= len(instance.Spec.Workflow) {
		return ctrl.Result{}, nil
	} else if !workflowActive && r.JobExists(ctx, instance, currentWorkflowStep) {
		return ctrl.Result{}, nil
	}

	// We are about to start job that spawns the pod with tests.
	// This lock ensures that there is always only one pod running.
	if !r.AcquireLock(ctx, instance, helper, instance.Spec.Parallel) {
		logging.Info("Can not acquire lock")
		return ctrl.Result{RequeueAfter: requeueAfter}, nil
	}
	logging.Info("Lock acquired")

	if workflowActive {
		r.WorkflowStepCounterIncrease(ctx, instance, helper)
	}

	// Prepare Tobiko env vars
	envVars := r.PrepareTobikoEnvVars(ctx, serviceLabels, instance, helper, externalWorkflowCounter)
	jobName := r.GetJobName(instance, externalWorkflowCounter)
	logsPVCName := r.GetPVCLogsName(instance)
	logging.Info(instance.Spec.ContainerImage)
	containerImage := GetContainerImage(ctx, helper, instance)
	logging.Info("HIHIHI")
	logging.Info(containerImage)
	jobDef := tobiko.Job(
		instance,
		serviceLabels,
		jobName,
		logsPVCName,
		containerImage,
		mountCerts,
		mountKeys,
		mountKubeconfig,
		envVars,
	)
	tobikoJob := job.NewJob(
		jobDef,
		testv1beta1.ConfigHash,
		true,
		time.Duration(5)*time.Second,
		"",
	)

	ctrlResult, err = tobikoJob.DoJob(ctx, helper)
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

// SetupWithManager sets up the controller with the Manager.
func (r *TobikoReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&testv1beta1.Tobiko{}).
		Owns(&batchv1.Job{}).
		Owns(&corev1.Secret{}).
		Owns(&corev1.ConfigMap{}).
		Complete(r)
}

// Tobiko requires password value to be present in clouds.yaml
// This code ensures that we set a default value of 12345678 when
// password value is missing in the clouds.yaml
func (r *TobikoReconciler) EnsureTobikoCloudsYAML(ctx context.Context, instance client.Object, helper *helper.Helper, labels map[string]string) (ctrl.Result, error) {
	cm, _, _ := configmap.GetConfigMap(ctx, helper, instance, "openstack-config", time.Second*10)
	result := make(map[interface{}]interface{})

	err := yaml.Unmarshal([]byte(cm.Data["clouds.yaml"]), &result)
	if err != nil {
		return ctrl.Result{}, err
	}

	clouds := result["clouds"].(map[interface{}]interface{})
	default_value := clouds["default"].(map[interface{}]interface{})
	auth := default_value["auth"].(map[interface{}]interface{})

	if _, ok := auth["password"].(string); !ok {
		auth["password"] = "12345678"
	}

	yamlString, err := yaml.Marshal(result)
	if err != nil {
		return ctrl.Result{}, err
	}

	cms := []util.Template{
		{
			Name:      "tobiko-clouds-config",
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

// This function prepares env variables for a single workflow step.
func (r *TobikoReconciler) PrepareTobikoEnvVars(
	ctx context.Context,
	labels map[string]string,
	instance *testv1beta1.Tobiko,
	helper *helper.Helper,
	step int,
) map[string]env.Setter {

	// NOTE(lpiwowar): Move all the merge code to the webhook once it is completed.
	//                 It will clean up the workflow code and remove the duplicit code
	//                 (Tempest vs Tobiko)
	if step < len(instance.Spec.Workflow) {
		if instance.Spec.Workflow[step].NodeSelector != nil {
			instance.Spec.NodeSelector = *instance.Spec.Workflow[step].NodeSelector
		}

		if instance.Spec.Workflow[step].Tolerations != nil {
			instance.Spec.Tolerations = *instance.Spec.Workflow[step].Tolerations
		}
	}

	// Prepare env vars
	envVars := make(map[string]env.Setter)
	envVars["USE_EXTERNAL_FILES"] = env.SetValue("True")
	envVars["TOBIKO_LOGS_DIR_NAME"] = env.SetValue(r.GetJobName(instance, step))

	testenv := r.OverwriteValueWithWorkflow(ctx, instance.Spec, "Testenv", "string", step).(string)
	envVars["TOBIKO_TESTENV"] = env.SetValue(testenv)

	version := r.OverwriteValueWithWorkflow(ctx, instance.Spec, "Version", "string", step).(string)
	envVars["TOBIKO_VERSION"] = env.SetValue(version)

	pytestAddopts := r.OverwriteValueWithWorkflow(ctx, instance.Spec, "PytestAddopts", "string", step).(string)
	envVars["TOBIKO_PYTEST_ADDOPTS"] = env.SetValue(pytestAddopts)

	preventCreate := r.OverwriteValueWithWorkflow(ctx, instance.Spec, "PreventCreate", "pbool", step).(bool)
	if preventCreate {
		envVars["TOBIKO_PREVENT_CREATE"] = env.SetValue("True")
	}

	numProcesses := r.OverwriteValueWithWorkflow(ctx, instance.Spec, "NumProcesses", "puint8", step).(uint8)
	if numProcesses > 0 {
		envVars["TOX_NUM_PROCESSES"] = env.SetValue(strconv.Itoa(int(numProcesses)))
	}

	envVars["TOBIKO_KEYS_FOLDER"] = env.SetValue("/etc/test_operator")
	// Prepare env vars - end

	// Prepare custom data
	customData := make(map[string]string)
	tobikoConf := r.OverwriteValueWithWorkflow(ctx, instance.Spec, "Config", "string", step).(string)
	customData["tobiko.conf"] = tobikoConf

	privateKeyData := make(map[string]string)
	privateKey := r.OverwriteValueWithWorkflow(ctx, instance.Spec, "PrivateKey", "string", step).(string)
	privateKeyData["id_ecdsa"] = privateKey

	publicKeyData := make(map[string]string)
	publicKey := r.OverwriteValueWithWorkflow(ctx, instance.Spec, "PublicKey", "string", step).(string)
	publicKeyData["id_ecdsa.pub"] = publicKey

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

	configmap.EnsureConfigMaps(ctx, helper, instance, cms, nil)

	return envVars
}
