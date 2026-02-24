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

	"github.com/go-logr/logr"
	"github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	"github.com/openstack-k8s-operators/lib-common/modules/common/env"
	"github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	testv1beta1 "github.com/openstack-k8s-operators/test-operator/api/v1beta1"
	"github.com/openstack-k8s-operators/test-operator/internal/horizontest"
	corev1 "k8s.io/api/core/v1"
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
func (r *HorizonTestReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	instance := &testv1beta1.HorizonTest{}

	config := FrameworkConfig[*testv1beta1.HorizonTest]{
		ServiceName:             horizontest.ServiceName,
		NeedsNetworkAttachments: false,
		NeedsConfigMaps:         true,
		NeedsFinalizer:          false,
		SupportsWorkflow:        false,

		GenerateServiceConfigMaps: func(ctx context.Context, helper *helper.Helper, instance *testv1beta1.HorizonTest, _ int) error {
			return r.generateServiceConfigMaps(ctx, helper, instance)
		},

		BuildPod: func(ctx context.Context, instance *testv1beta1.HorizonTest, labels, annotations map[string]string, workflowStepNum int, pvcIndex int) (*corev1.Pod, error) {
			return r.buildHorizonTestPod(ctx, instance, labels, annotations, workflowStepNum, pvcIndex)
		},

		GetInitialConditions: func() []*condition.Condition {
			return []*condition.Condition{
				condition.UnknownCondition(condition.ReadyCondition, condition.InitReason, condition.ReadyInitMessage),
				condition.UnknownCondition(condition.InputReadyCondition, condition.InitReason, condition.InputReadyInitMessage),
				condition.UnknownCondition(condition.DeploymentReadyCondition, condition.InitReason, condition.DeploymentReadyInitMessage),
			}
		},

		ValidateInputs: func(ctx context.Context, instance *testv1beta1.HorizonTest) error {
			if err := r.ValidateOpenstackInputs(ctx, instance, instance.Spec.OpenStackConfigMap, instance.Spec.OpenStackConfigSecret); err != nil {
				return err
			}
			return r.ValidateSecretWithKeys(ctx, instance, instance.Spec.KubeconfigSecretName, []string{})
		},

		GetSpec: func(instance *testv1beta1.HorizonTest) interface{} {
			return &instance.Spec
		},

		GetParallel: func(instance *testv1beta1.HorizonTest) bool {
			return instance.Spec.Parallel
		},

		GetStorageClass: func(instance *testv1beta1.HorizonTest) string {
			return instance.Spec.StorageClass
		},

		SetObservedGeneration: func(instance *testv1beta1.HorizonTest) {
			instance.Status.ObservedGeneration = instance.Generation
		},
	}

	return CommonReconcile(ctx, &r.Reconciler, req, instance, config, r.GetLogger(ctx))
}

func (r *HorizonTestReconciler) generateServiceConfigMaps(
	ctx context.Context,
	h *helper.Helper,
	instance *testv1beta1.HorizonTest,
) error {
	labels := map[string]string{
		operatorNameLabel: "test-operator",
		instanceNameLabel: instance.Name,
	}

	_, err := EnsureCloudsConfigMapExists(
		ctx,
		instance,
		h,
		labels,
		instance.Spec.OpenStackConfigMap,
	)
	return err
}

func (r *HorizonTestReconciler) buildHorizonTestPod(
	ctx context.Context,
	instance *testv1beta1.HorizonTest,
	labels, _ map[string]string,
	workflowStepNum int,
	pvcIndex int,
) (*corev1.Pod, error) {
	mountCerts := r.CheckSecretExists(ctx, instance, "combined-ca-bundle")
	mountKubeconfig := len(instance.Spec.KubeconfigSecretName) != 0

	envVars := r.PrepareHorizonTestEnvVars(instance)
	podName := r.GetPodName(instance, workflowStepNum)
	logsPVCName := r.GetPVCLogsName(instance, pvcIndex)

	containerImage, err := r.GetContainerImage(ctx, instance)
	if err != nil {
		return nil, err
	}

	return horizontest.Pod(
		instance,
		labels,
		podName,
		logsPVCName,
		mountCerts,
		mountKubeconfig,
		envVars,
		containerImage,
	), nil
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
