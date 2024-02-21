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
	instance := &testv1beta1.Tobiko{}
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

	serviceLabels := map[string]string{
		common.AppSelector: tobiko.ServiceName,
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

	// Prepare env vars
	envVars := make(map[string]env.Setter)
	envVars["TOBIKO_TESTENV"] = env.SetValue(instance.Spec.Testenv)
	envVars["USE_EXTERNAL_FILES"] = env.SetValue("True")
	envVars["TOBIKO_VERSION"] = env.SetValue(instance.Spec.Version)
	envVars["TOBIKO_PYTEST_ADDOPTS"] = env.SetValue(instance.Spec.PytestAddopts)
	if instance.Spec.PreventCreate {
		envVars["TOBIKO_PREVENT_CREATE"] = env.SetValue("True")
	}
	if instance.Spec.NumProcesses > 0 {
		envVars["TOX_NUM_PROCESSES"] = env.SetValue(strconv.Itoa(int(instance.Spec.NumProcesses)))
	}
	// Prepare env vars - end

	// Prepare custom data
	customData := make(map[string]string)
	if len(instance.Spec.Config) != 0 {
		customData["tobiko.conf"] = instance.Spec.Config
	}

	privateKeyData := make(map[string]string)
	if len(instance.Spec.PrivateKey) != 0 {
		privateKeyData["id_ecdsa"] = instance.Spec.PrivateKey
	}

	publicKeyData := make(map[string]string)
	if len(instance.Spec.PublicKey) != 0 {
		publicKeyData["id_ecdsa.pub"] = instance.Spec.PublicKey
	}

	cms := []util.Template{
		{
			Name:         instance.Name + "tobiko-config",
			Namespace:    instance.Namespace,
			InstanceType: instance.Kind,
			CustomData:   customData,
		},
		{
			Name:         instance.Name + "tobiko-private-key",
			Namespace:    instance.Namespace,
			InstanceType: instance.Kind,
			CustomData:   privateKeyData,
		},
		{
			Name:         instance.Name + "tobiko-public-key",
			Namespace:    instance.Namespace,
			InstanceType: instance.Kind,
			CustomData:   publicKeyData,
		},
	}

	configmap.EnsureConfigMaps(ctx, helper, instance, cms, nil)
	// Prepare custom data - end

	result, err := r.EnsureTobikoCloudsYAML(ctx, instance, helper)
	if err != nil {
		return result, err
	}

	// Create PersistentVolumeClaim
	ctrlResult, err := r.EnsureLogsPVCExists(ctx, instance, helper, instance.Name, instance.Spec.StorageClass)
	if err != nil {
		return ctrlResult, err
	} else if (ctrlResult != ctrl.Result{}) {
		return ctrlResult, nil
	}
	// Create PersistentVolumeClaim - end

	// Create Job
	mountCerts := r.CheckSecretExists(ctx, instance, "combined-ca-bundle")

	mountKeys := false
	if (len(publicKeyData) == 0) || (len(privateKeyData) == 0) {
		logging.Info("Both values privateKey and publicKey need to be specified. Keys not mounted.")
	} else {
		mountKeys = true
	}

	mountKubeconfig := false
	if len(instance.Spec.KubeconfigSecretName) != 0 {
		mountKubeconfig = true
	}

	jobDef := tobiko.Job(instance, serviceLabels, mountCerts, mountKeys, mountKubeconfig, envVars)
	tobikoJob := job.NewJob(
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
	if !r.AcquireLock(ctx, instance, helper, instance.Spec.Parallel) {
		logging.Info("Can not acquire lock")
		requeueAfter := time.Second * 60
		return ctrl.Result{RequeueAfter: requeueAfter}, nil
	} else {
		logging.Info("Lock acquired")
	}

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
func (r *TobikoReconciler) EnsureTobikoCloudsYAML(ctx context.Context, instance client.Object, helper *helper.Helper) (ctrl.Result, error) {
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
			CustomData: map[string]string{
				"clouds.yaml": string(yamlString),
			},
		},
	}
	configmap.EnsureConfigMaps(ctx, helper, instance, cms, nil)

	return ctrl.Result{}, nil
}
