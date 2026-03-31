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
	"github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	corev1 "k8s.io/api/core/v1"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// TestResource defines the interface that all test-operator Custom Resources must implement
type TestResource interface {
	client.Object
	GetConditions() *condition.Conditions
	GetStorageClass() string
	SetObservedGeneration()
}

// TestResourceConfig defines resource-specific configuration and behavior
type TestResourceConfig[T TestResource] struct {
	// ServiceName for labeling (e.g., "tempest", "tobiko")
	ServiceName string

	// NeedsNetworkAttachments indicates if NADs should be handled
	NeedsNetworkAttachments bool

	// NeedsConfigMaps indicates if ServiceConfigReadyCondition is needed
	NeedsConfigMaps bool

	// NeedsFinalizer indicates if the controller needs finalizer handling
	NeedsFinalizer bool

	// SupportsWorkflow indicates if the controller supports workflow feature
	SupportsWorkflow bool

	// GenerateServiceConfigMaps creates resource-specific config maps
	GenerateServiceConfigMaps func(ctx context.Context, helper *helper.Helper, labels map[string]string, instance T, workflowStepIndex int) error

	// BuildPod creates the resource-specific pod definition
	BuildPod func(ctx context.Context, instance T, labels, annotations map[string]string, workflowStepIndex int, pvcIndex int) (*corev1.Pod, error)

	// GetInitialConditions returns the condition list for a new instance
	GetInitialConditions func() []*condition.Condition

	// ValidateInputs validates resource-specific inputs
	ValidateInputs func(ctx context.Context, instance T) error

	// Optional filed accessors
	GetParallel                func(instance T) bool
	GetNetworkAttachments      func(instance T) []string
	GetNetworkAttachmentStatus func(instance T) *map[string][]string

	// Optional filed accessors - workflow support
	GetSpec           func(instance T) interface{}
	GetWorkflowStep   func(instance T, step int) interface{}
	GetWorkflowLength func(instance T) int
}

// CommonReconcile executes the standard reconciliation workflow using generics
func CommonReconcile[T TestResource](
	ctx context.Context,
	r *Reconciler,
	req ctrl.Request,
	instance T,
	config TestResourceConfig[T],
	Log logr.Logger,
) (result ctrl.Result, _err error) {
	err := r.Client.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if k8s_errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Create a helper
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

	// Get conditions from instance
	conditions := instance.GetConditions()

	// Initialize status
	isNewInstance := len(*conditions) == 0
	if isNewInstance {
		*conditions = condition.Conditions{}
	}

	// Save a copy of the conditions so that we can restore the LastTransitionTime
	// when a condition's state doesn't change.
	savedConditions := conditions.DeepCopy()

	// Always patch the instance status when exiting this function so we
	// can persist any changes.
	defer func() {
		// Don't update the status, if reconciler Panics
		if r := recover(); r != nil {
			Log.Info(fmt.Sprintf("panic during reconcile %v\n", r))
			panic(r)
		}
		condition.RestoreLastTransitionTimes(conditions, savedConditions)
		if conditions.IsUnknown(condition.ReadyCondition) {
			conditions.Set(conditions.Mirror(condition.ReadyCondition))
		}
		err := helper.PatchInstance(ctx, instance)
		if err != nil {
			_err = err
			return
		}
	}()

	if isNewInstance {
		// Initialize conditions used later as Status=Unknown
		cl := condition.CreateList(config.GetInitialConditions()...)
		conditions.Init(&cl)

		// Register overall status immediately to have an early feedback
		// e.g. in the cli
		return ctrl.Result{}, nil
	}
	instance.SetObservedGeneration()

	// If we're not deleting this and the service object doesn't have our
	// finalizer, add it.
	if config.NeedsFinalizer && instance.GetDeletionTimestamp().IsZero() &&
		controllerutil.AddFinalizer(instance, helper.GetFinalizer()) {
		return ctrl.Result{}, nil
	}

	// Handle service delete
	if config.NeedsFinalizer && !instance.GetDeletionTimestamp().IsZero() {
		Log.Info("Reconciling Service delete")
		controllerutil.RemoveFinalizer(instance, helper.GetFinalizer())
		Log.Info("Reconciled Service delete successfully")
		return ctrl.Result{}, nil
	}

	if config.NeedsNetworkAttachments {
		networkStatus := config.GetNetworkAttachmentStatus(instance)
		if networkStatus != nil && *networkStatus == nil {
			*networkStatus = map[string][]string{}
		}
	}

	workflowLength := 0
	if config.SupportsWorkflow {
		workflowLength = config.GetWorkflowLength(instance)
	}

	nextAction, workflowStepIndex, err := r.NextAction(ctx, instance, workflowLength)
	if nextAction == Failure {
		return ctrl.Result{}, err
	}

	// Apply workflow step overrides to the base spec
	if config.SupportsWorkflow && workflowStepIndex < workflowLength {
		spec := config.GetSpec(instance)
		workflowStepData := config.GetWorkflowStep(instance, workflowStepIndex)
		MergeSections(spec, workflowStepData)
	}

	parallel := false
	if config.GetParallel != nil {
		parallel = config.GetParallel(instance)
	}

	pvcIndex := 0
	// Create multiple PVCs for parallel execution
	if parallel && config.SupportsWorkflow && workflowStepIndex < workflowLength {
		pvcIndex = workflowStepIndex
	}

	switch nextAction {
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

		conditions.MarkTrue(condition.DeploymentReadyCondition, condition.DeploymentReadyMessage)

		if conditions.AllSubConditionIsTrue() {
			conditions.MarkTrue(condition.ReadyCondition, condition.ReadyMessage)
		}

		Log.Info(InfoTestingCompleted)
		return ctrl.Result{}, nil

	case CreateFirstPod:
		lockAcquired, err := r.AcquireLock(ctx, instance, helper, parallel)
		if !lockAcquired {
			Log.Info(fmt.Sprintf(InfoCanNotAcquireLock, testOperatorLockName))
			return ctrl.Result{RequeueAfter: RequeueAfterValue}, err
		}

		Log.Info(fmt.Sprintf(InfoCreatingFirstPod, workflowStepIndex))

	case CreateNextPod:
		// Confirm that we still hold the lock. This is useful to check if for
		// example somebody / something deleted the lock and it got claimed by
		// another instance. This is considered to be an error state.
		lockAcquired, err := r.AcquireLock(ctx, instance, helper, parallel)
		if !lockAcquired {
			Log.Error(err, fmt.Sprintf(ErrConfirmLockOwnership, testOperatorLockName))
			return ctrl.Result{RequeueAfter: RequeueAfterValue}, err
		}

		Log.Info(fmt.Sprintf(InfoCreatingNextPod, workflowStepIndex))

	default:
		return ctrl.Result{}, ErrReceivedUnexpectedAction
	}

	serviceLabels := map[string]string{
		common.AppSelector: config.ServiceName,
		workflowStepLabel:  strconv.Itoa(workflowStepIndex),
		instanceNameLabel:  instance.GetName(),
		operatorNameLabel:  "test-operator",
	}

	// Validate inputs
	if config.ValidateInputs != nil {
		if err := config.ValidateInputs(ctx, instance); err != nil {
			conditions.Set(condition.FalseCondition(
				condition.InputReadyCondition,
				condition.ErrorReason,
				condition.SeverityError,
				condition.InputReadyErrorMessage,
				err.Error()))
			return ctrl.Result{RequeueAfter: RequeueAfterValue}, err
		}
		conditions.MarkTrue(condition.InputReadyCondition, condition.InputReadyMessage)
	}

	// Create PersistentVolumeClaim
	ctrlResult, err := r.EnsureLogsPVCExists(
		ctx,
		instance,
		helper,
		serviceLabels,
		instance.GetStorageClass(),
		pvcIndex,
	)
	if err != nil {
		return ctrlResult, err
	} else if (ctrlResult != ctrl.Result{}) {
		return ctrlResult, nil
	}

	// Generate ConfigMaps containing test configuration
	if config.NeedsConfigMaps {
		err = config.GenerateServiceConfigMaps(ctx, helper, serviceLabels, instance, workflowStepIndex)
		if err != nil {
			conditions.Set(condition.FalseCondition(
				condition.ServiceConfigReadyCondition,
				condition.ErrorReason,
				condition.SeverityWarning,
				condition.ServiceConfigReadyErrorMessage,
				err.Error()))
			return ctrl.Result{}, err
		}
		conditions.MarkTrue(condition.ServiceConfigReadyCondition, condition.ServiceConfigReadyMessage)
	}

	var serviceAnnotations map[string]string
	if config.NeedsNetworkAttachments {
		annotations, ctrlResult, err := r.EnsureNetworkAttachments(
			ctx,
			Log,
			helper,
			config.GetNetworkAttachments(instance),
			instance.GetNamespace(),
			conditions,
		)
		if err != nil || (ctrlResult != ctrl.Result{}) {
			return ctrlResult, err
		}
		serviceAnnotations = annotations
	}

	// Build pod
	podDef, err := config.BuildPod(
		ctx,
		instance,
		serviceLabels,
		serviceAnnotations,
		workflowStepIndex,
		pvcIndex,
	)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Create a new pod
	ctrlResult, err = r.CreatePod(ctx, *helper, podDef)
	if err != nil {
		// Release the lock and allow other controllers to spawn
		// a pod.
		if lockReleased, lockErr := r.ReleaseLock(ctx, instance); !lockReleased {
			return ctrl.Result{RequeueAfter: RequeueAfterValue}, lockErr
		}
		conditions.Set(condition.FalseCondition(
			condition.DeploymentReadyCondition,
			condition.ErrorReason,
			condition.SeverityWarning,
			condition.DeploymentReadyErrorMessage,
			err.Error()))
		return ctrlResult, err
	} else if (ctrlResult != ctrl.Result{}) {
		conditions.Set(condition.FalseCondition(
			condition.DeploymentReadyCondition,
			condition.RequestedReason,
			condition.SeverityInfo,
			condition.DeploymentReadyRunningMessage))
		return ctrlResult, nil
	}

	if config.NeedsNetworkAttachments {
		ctrlResult, err = r.VerifyNetworkAttachments(
			ctx,
			helper,
			instance,
			config.GetNetworkAttachments(instance),
			serviceLabels,
			workflowStepIndex,
			conditions,
			config.GetNetworkAttachmentStatus(instance),
		)
		if err != nil || (ctrlResult != ctrl.Result{}) {
			return ctrlResult, err
		}
	}

	// Mark ready if all conditions are true
	if conditions.AllSubConditionIsTrue() {
		conditions.MarkTrue(condition.ReadyCondition, condition.ReadyMessage)
	}

	Log.Info("Reconciled Service successfully")
	return ctrl.Result{}, nil
}
