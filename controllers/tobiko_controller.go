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

	"github.com/go-logr/logr"
	"github.com/openstack-k8s-operators/lib-common/modules/common/configmap"
	"github.com/openstack-k8s-operators/lib-common/modules/common/env"
	"github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	"github.com/openstack-k8s-operators/lib-common/modules/common/util"
	testv1beta1 "github.com/openstack-k8s-operators/test-operator/api/v1beta1"
	"gopkg.in/yaml.v3"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

//+kubebuilder:rbac:groups=test.openstack.org,resources=tobikoes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=test.openstack.org,resources=tobikoes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=test.openstack.org,resources=tobikoes/finalizers,verbs=update;patch
//+kubebuilder:rbac:groups="",resources=pods,verbs=create;delete;get;list;patch;update;watch
//+kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;create;update;watch;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *TobikoReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, _err error) {

	// TODO(lpiwowar)
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
	result := make(map[string]interface{})

	err := yaml.Unmarshal([]byte(cm.Data["clouds.yaml"]), &result)
	if err != nil {
		return ctrl.Result{}, err
	}

	clouds := result["clouds"].(map[string]interface{})
	defaultValue := clouds["default"].(map[string]interface{})
	auth := defaultValue["auth"].(map[string]interface{})

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
	err = configmap.EnsureConfigMaps(ctx, helper, instance, cms, nil)
	if err != nil {
		return ctrl.Result{}, err
	}

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

	testenv := r.OverwriteValueWithWorkflow(instance.Spec, "Testenv", "string", step).(string)
	envVars["TOBIKO_TESTENV"] = env.SetValue(testenv)

	version := r.OverwriteValueWithWorkflow(instance.Spec, "Version", "string", step).(string)
	envVars["TOBIKO_VERSION"] = env.SetValue(version)

	pytestAddopts := r.OverwriteValueWithWorkflow(instance.Spec, "PytestAddopts", "string", step).(string)
	envVars["TOBIKO_PYTEST_ADDOPTS"] = env.SetValue(pytestAddopts)

	preventCreate := r.OverwriteValueWithWorkflow(instance.Spec, "PreventCreate", "pbool", step).(bool)
	if preventCreate {
		envVars["TOBIKO_PREVENT_CREATE"] = env.SetValue("True")
	}

	numProcesses := r.OverwriteValueWithWorkflow(instance.Spec, "NumProcesses", "puint8", step).(uint8)
	if numProcesses > 0 {
		envVars["TOX_NUM_PROCESSES"] = env.SetValue(strconv.Itoa(int(numProcesses)))
	}

	envVars["TOBIKO_KEYS_FOLDER"] = env.SetValue("/etc/test_operator")
	envVars["TOBIKO_DEBUG_MODE"] = env.SetValue(r.GetDefaultBool(instance.Spec.Debug))
	// Prepare env vars - end

	// Prepare custom data
	customData := make(map[string]string)
	tobikoConf := r.OverwriteValueWithWorkflow(instance.Spec, "Config", "string", step).(string)
	customData["tobiko.conf"] = tobikoConf

	privateKeyData := make(map[string]string)
	privateKey := r.OverwriteValueWithWorkflow(instance.Spec, "PrivateKey", "string", step).(string)
	privateKeyData["id_ecdsa"] = privateKey

	publicKeyData := make(map[string]string)
	publicKey := r.OverwriteValueWithWorkflow(instance.Spec, "PublicKey", "string", step).(string)
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

	err := configmap.EnsureConfigMaps(ctx, helper, instance, cms, nil)
	if err != nil {
		return map[string]env.Setter{}
	}

	return envVars
}
