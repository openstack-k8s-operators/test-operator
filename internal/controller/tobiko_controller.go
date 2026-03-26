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
	"github.com/openstack-k8s-operators/lib-common/modules/common/env"
	"github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	"github.com/openstack-k8s-operators/lib-common/modules/common/util"
	testv1beta1 "github.com/openstack-k8s-operators/test-operator/api/v1beta1"
	"github.com/openstack-k8s-operators/test-operator/internal/tobiko"
	corev1 "k8s.io/api/core/v1"
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
func (r *TobikoReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	instance := &testv1beta1.Tobiko{}

	config := TestResourceConfig[*testv1beta1.Tobiko]{
		ServiceName:             tobiko.ServiceName,
		NeedsNetworkAttachments: true,
		NeedsConfigMaps:         true,
		NeedsFinalizer:          false,
		SupportsWorkflow:        true,

		GenerateServiceConfigMaps: func(ctx context.Context, helper *helper.Helper, labels map[string]string, instance *testv1beta1.Tobiko, workflowStepIndex int) error {
			return r.generateServiceConfigMaps(ctx, helper, labels, instance, workflowStepIndex)
		},

		BuildPod: func(ctx context.Context, instance *testv1beta1.Tobiko, labels, annotations map[string]string, workflowStepIndex int, pvcIndex int) (*corev1.Pod, error) {
			return r.buildTobikoPod(ctx, instance, labels, annotations, workflowStepIndex, pvcIndex)
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

		ValidateInputs: func(ctx context.Context, instance *testv1beta1.Tobiko) error {
			if err := r.ValidateOpenstackInputs(ctx, instance, instance.Spec.OpenStackConfigMap, instance.Spec.OpenStackConfigSecret); err != nil {
				return err
			}
			return r.ValidateSecretWithKeys(ctx, instance, instance.Spec.KubeconfigSecretName, []string{})
		},

		GetSpec: func(instance *testv1beta1.Tobiko) interface{} {
			return &instance.Spec
		},

		GetWorkflowStep: func(instance *testv1beta1.Tobiko, step int) interface{} {
			return instance.Spec.Workflow[step]
		},

		GetWorkflowLength: func(instance *testv1beta1.Tobiko) int {
			return len(instance.Spec.Workflow)
		},

		GetParallel: func(instance *testv1beta1.Tobiko) bool {
			return instance.Spec.Parallel
		},

		GetNetworkAttachments: func(instance *testv1beta1.Tobiko) []string {
			return instance.Spec.NetworkAttachments
		},

		GetNetworkAttachmentStatus: func(instance *testv1beta1.Tobiko) *map[string][]string {
			return &instance.Status.NetworkAttachments
		},
	}

	return CommonReconcile(ctx, &r.Reconciler, req, instance, config, r.GetLogger(ctx))
}

func (r *TobikoReconciler) buildTobikoPod(
	ctx context.Context,
	instance *testv1beta1.Tobiko,
	labels, annotations map[string]string,
	workflowStepIndex int,
	pvcIndex int,
) (*corev1.Pod, error) {
	Log := r.GetLogger(ctx)

	mountCerts := r.CheckSecretExists(ctx, instance, "combined-ca-bundle")
	mountKubeconfig := len(instance.Spec.KubeconfigSecretName) != 0
	mountKeys := len(instance.Spec.PublicKey) > 0 && len(instance.Spec.PrivateKey) > 0
	if !mountKeys {
		Log.Info("Both values 'privateKey' and 'publicKey' need to be specified. Keys not mounted.")
	}

	// Prepare Tobiko env vars
	envVars := r.PrepareTobikoEnvVars(instance, workflowStepIndex)
	podName := r.GetPodName(instance, workflowStepIndex)
	logsPVCName := r.GetPVCLogsName(instance, pvcIndex)

	containerImage, err := r.GetContainerImage(ctx, instance)
	if err != nil {
		return nil, err
	}

	return tobiko.Pod(
		instance,
		labels,
		annotations,
		podName,
		logsPVCName,
		mountCerts,
		mountKeys,
		mountKubeconfig,
		workflowStepIndex,
		envVars,
		containerImage,
	), nil
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

func (r *TobikoReconciler) generateServiceConfigMaps(
	ctx context.Context,
	h *helper.Helper,
	labels map[string]string,
	instance *testv1beta1.Tobiko,
	workflowStepIndex int,
) error {
	err := EnsureCloudsConfigMapExists(
		ctx,
		instance,
		h,
		labels,
		instance.Spec.OpenStackConfigMap,
	)
	if err != nil {
		return err
	}

	templateSpecs := []struct {
		infix string
		key   string
		value string
	}{
		{tobiko.ConfigMapInfixConfig, tobiko.ConfigFileName, instance.Spec.Config},
		{tobiko.ConfigMapInfixPrivateKey, tobiko.PrivateKeyFileName, instance.Spec.PrivateKey},
		{tobiko.ConfigMapInfixPublicKey, tobiko.PublicKeyFileName, instance.Spec.PublicKey},
	}

	cms := make([]util.Template, 0, len(templateSpecs))
	for _, spec := range templateSpecs {
		cms = append(cms, util.Template{
			Name:         tobiko.GetConfigMapName(instance, spec.infix, workflowStepIndex),
			Namespace:    instance.Namespace,
			InstanceType: instance.Kind,
			Labels:       labels,
			CustomData:   map[string]string{spec.key: spec.value},
		})
	}

	return configmap.EnsureConfigMaps(ctx, h, instance, cms, nil)
}

// PrepareTobikoEnvVars prepares environment variables for a single workflow step
func (r *TobikoReconciler) PrepareTobikoEnvVars(
	instance *testv1beta1.Tobiko,
	workflowStepIndex int,
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
		"TOBIKO_LOGS_DIR_NAME":  r.GetPodName(instance, workflowStepIndex),
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

	return envVars
}
