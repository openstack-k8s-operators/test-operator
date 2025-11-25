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
	"fmt"
	"strconv"
	"time"

	"github.com/go-logr/logr"
	networkv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	"github.com/openstack-k8s-operators/lib-common/modules/common"
	"github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	"github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	nad "github.com/openstack-k8s-operators/lib-common/modules/common/networkattachment"
	corev1 "k8s.io/api/core/v1"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// FrameworkInstance defines the interface that all test framework CRs must implement
type FrameworkInstance interface {
	client.Object
	GetConditions() *condition.Conditions
}

// FrameworkConfig defines framework-specific configuration and behavior
type FrameworkConfig[T FrameworkInstance] struct {
	// ServiceName for labeling (e.g., "tempest", "tobiko")
	ServiceName string

	// NeedsNetworkAttachments indicates if NADs should be handled
	NeedsNetworkAttachments bool

	// NeedsConfigMaps indicates if ServiceConfigReadyCondition is needed
	NeedsConfigMaps bool

	// GenerateServiceConfigMaps creates framework-specific config maps
	GenerateServiceConfigMaps func(ctx context.Context, r *Reconciler, helper *helper.Helper, instance T, workflowStep int) error

	// BuildPod creates the framework-specific pod definition
	BuildPod func(ctx context.Context, r *Reconciler, instance T, labels, annotations map[string]string, workflowStep int) (*corev1.Pod, error)

	// GetInitialConditions returns the condition list for a new instance
	GetInitialConditions func() []*condition.Condition

	// Field accessors
	GetWorkflowLength          func(instance T) int
	GetParallel                func(instance T) bool
	GetStorageClass            func(instance T) string
	GetNetworkAttachments      func(instance T) []string
	GetNetworkAttachmentStatus func(instance T) map[string][]string
	SetNetworkAttachmentStatus func(instance T, status map[string][]string)

	GetSpec         func(instance T) interface{}           // Optional
	GetWorkflowStep func(instance T, step int) interface{} // Optional
}

// CommonReconcile executes the standard reconciliation workflow using generics
func CommonReconcile[T FrameworkInstance](
	ctx context.Context,
	r *Reconciler,
	req ctrl.Request,
	instance T,
	config FrameworkConfig[T],
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
	helper, err := helper.NewHelper(instance, r.Client, r.Kclient, r.Scheme, r.Log)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Get conditions from instance
	conditions := instance.GetConditions()
	if conditions == nil {
		return ctrl.Result{}, nil // TODO fmt.Errorf("instance does not support conditions")
	}

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
		// update the overall status condition if service is ready
		if conditions.AllSubConditionIsTrue() {
			conditions.MarkTrue(condition.ReadyCondition, condition.ReadyMessage)
		}
		condition.RestoreLastTransitionTimes(conditions, savedConditions)
		if conditions.IsUnknown(condition.ReadyCondition) {
			conditions.Set(conditions.Mirror(condition.ReadyCondition))
		}
		err := helper.PatchInstance(ctx, instance)
		if err != nil {
			_err = err
		}
	}()

	if isNewInstance {
		cl := condition.CreateList(config.GetInitialConditions()...)
		conditions.Init(&cl)

		// Register overall status immediately to have an early feedback
		// e.g. in the cli
		return ctrl.Result{}, nil
	}

	// Initialize network attachments status if needed
	if config.NeedsNetworkAttachments {
		if config.GetNetworkAttachmentStatus(instance) == nil {
			config.SetNetworkAttachmentStatus(instance, map[string][]string{})
		}
	}

	// Handle service delete
	if !instance.GetDeletionTimestamp().IsZero() {
		Log.Info("Reconciling Service delete")
		controllerutil.RemoveFinalizer(instance, helper.GetFinalizer())
		Log.Info("Reconciled Service delete successfully")
		return ctrl.Result{}, nil
	}

	workflowLength := config.GetWorkflowLength(instance)
	nextAction, workflowStep, err := r.NextAction(ctx, instance, workflowLength)

	// Merge workflow step if applicable
	if workflowLength != 0 && workflowStep < workflowLength {
		spec := config.GetSpec(instance)
		workflowStepData := config.GetWorkflowStep(instance, workflowStep)
		MergeSections(spec, workflowStepData)
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

		conditions.MarkTrue(condition.DeploymentReadyCondition, condition.DeploymentReadyMessage)
		Log.Info(InfoTestingCompleted)
		return ctrl.Result{}, nil

	case CreateFirstPod:
		lockAcquired, err := r.AcquireLock(ctx, instance, helper, config.GetParallel(instance))
		if !lockAcquired {
			Log.Info(fmt.Sprintf(InfoCanNotAcquireLock, testOperatorLockName))
			return ctrl.Result{RequeueAfter: RequeueAfterValue}, err
		}

		Log.Info(fmt.Sprintf(InfoCreatingFirstPod, workflowStep))

	case CreateNextPod:
		// Confirm that we still hold the lock. This is useful to check if for
		// example somebody / something deleted the lock and it got claimed by
		// another instance. This is considered to be an error state.
		lockAcquired, err := r.AcquireLock(ctx, instance, helper, config.GetParallel(instance))
		if !lockAcquired {
			Log.Error(err, ErrConfirmLockOwnership, testOperatorLockName)
			return ctrl.Result{RequeueAfter: RequeueAfterValue}, err
		}

		Log.Info(fmt.Sprintf(InfoCreatingNextPod, workflowStep))

	default:
		return ctrl.Result{}, ErrReceivedUnexpectedAction
	}

	serviceLabels := map[string]string{
		common.AppSelector: config.ServiceName,
		workflowStepLabel:  strconv.Itoa(workflowStep),
		instanceNameLabel:  instance.GetName(),
		operatorNameLabel:  "test-operator",
	}

	workflowStepNum := 0
	// Create multiple PVCs for parallel execution
	if config.GetParallel(instance) && workflowStep < config.GetWorkflowLength(instance) {
		workflowStepNum = workflowStep
	}

	// Create PersistentVolumeClaim
	ctrlResult, err := r.EnsureLogsPVCExists(
		ctx,
		instance,
		helper,
		serviceLabels,
		config.GetStorageClass(instance),
		workflowStepNum,
	)
	if err != nil {
		return ctrlResult, err
	} else if (ctrlResult != ctrl.Result{}) {
		return ctrlResult, nil
	}

	// Generate ConfigMaps if needed
	if config.NeedsConfigMaps {
		if err = config.GenerateServiceConfigMaps(ctx, r, helper, instance, workflowStep); err != nil {
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
	// Generate ConfigMaps - end

	// Handle network attachments if needed
	var serviceAnnotations map[string]string
	if config.NeedsNetworkAttachments {
		annotations, ctrlResult, err := handleNetworkAttachments(
			ctx, r, instance, helper, serviceLabels, config, workflowStep, conditions,
		)
		if err != nil || (ctrlResult != ctrl.Result{}) {
			return ctrlResult, err
		}
		serviceAnnotations = annotations
	}

	// Build pod
	podDef, err := config.BuildPod(ctx, r, instance, serviceLabels, serviceAnnotations, workflowStep)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Create a new pod
	ctrlResult, err = r.CreatePod(ctx, *helper, podDef)
	if err != nil {
		// Release lock on failure
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
	// Create a new pod - end

	return ctrl.Result{}, nil
}

func handleNetworkAttachments[T FrameworkInstance](
	ctx context.Context,
	r *Reconciler,
	instance T,
	helper *helper.Helper,
	labels map[string]string,
	config FrameworkConfig[T],
	workflowStep int,
	conditions *condition.Conditions,
) (map[string]string, ctrl.Result, error) {
	nadList := []networkv1.NetworkAttachmentDefinition{}
	networkAttachments := config.GetNetworkAttachments(instance)

	for _, netAtt := range networkAttachments {
		nadObj, err := nad.GetNADWithName(ctx, helper, netAtt, instance.GetNamespace())
		if err != nil {
			if k8s_errors.IsNotFound(err) {
				// Since the net-attach-def CR should have been manually created by the user and referenced in the spec,
				// we treat this as a warning because it means that the service will not be able to start.
				r.Log.Info(fmt.Sprintf("network-attachment-definition %s not found", netAtt))
				conditions.Set(condition.FalseCondition(
					condition.NetworkAttachmentsReadyCondition,
					condition.ErrorReason,
					condition.SeverityWarning,
					condition.NetworkAttachmentsReadyWaitingMessage,
					netAtt))
				return nil, ctrl.Result{RequeueAfter: time.Second * 10}, nil
			}
			conditions.Set(condition.FalseCondition(
				condition.NetworkAttachmentsReadyCondition,
				condition.ErrorReason,
				condition.SeverityWarning,
				condition.NetworkAttachmentsReadyErrorMessage,
				err.Error()))
			return nil, ctrl.Result{}, err
		}

		if nadObj != nil {
			nadList = append(nadList, *nadObj)
		}
	}

	serviceAnnotations, err := nad.EnsureNetworksAnnotation(nadList)
	if err != nil {
		return nil, ctrl.Result{}, fmt.Errorf("failed create network annotation from %s: %w",
			networkAttachments, err)
	}

	// Verify network status if pod exists
	if r.PodExists(ctx, instance, workflowStep) {
		networkReady, networkAttachmentStatus, err := nad.VerifyNetworkStatusFromAnnotation(
			ctx,
			helper,
			networkAttachments,
			labels,
			1,
		)
		if err != nil {
			return nil, ctrl.Result{}, err
		}

		config.SetNetworkAttachmentStatus(instance, networkAttachmentStatus)

		if networkReady {
			conditions.MarkTrue(
				condition.NetworkAttachmentsReadyCondition,
				condition.NetworkAttachmentsReadyMessage)
		} else {
			err := fmt.Errorf("%w: %s", ErrNetworkAttachmentsMismatch, networkAttachments)
			conditions.Set(condition.FalseCondition(
				condition.NetworkAttachmentsReadyCondition,
				condition.ErrorReason,
				condition.SeverityWarning,
				condition.NetworkAttachmentsReadyErrorMessage,
				err.Error()))
			return nil, ctrl.Result{}, err
		}
	}

	return serviceAnnotations, ctrl.Result{}, nil
}
