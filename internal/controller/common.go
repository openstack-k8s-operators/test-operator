package controller

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"crypto/sha256"

	"github.com/go-logr/logr"
	networkv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	"github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	"github.com/openstack-k8s-operators/lib-common/modules/common/configmap"
	"github.com/openstack-k8s-operators/lib-common/modules/common/env"
	"github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	nad "github.com/openstack-k8s-operators/lib-common/modules/common/networkattachment"
	"github.com/openstack-k8s-operators/lib-common/modules/common/pvc"
	"github.com/openstack-k8s-operators/lib-common/modules/common/util"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	k8sresource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	podNameStepInfix           = "-s"
	envVarsConfigMapInfix      = "-env-vars-s"
	customDataConfigMapInfix   = "-custom-data-s"
	workflowStepNameInvalid    = "no-name"
	workflowStepLabel          = "workflowStep"
	instanceNameLabel          = "instanceName"
	operatorNameLabel          = "operator"
	testOperatorLockName       = "test-operator-lock"
	testOperatorLockOwnerField = "owner"
	testOperatorBaseDir        = "/etc/test_operator/"
)

const (
	// ErrConfirmLockOwnership is the error message for lock ownership confirmation failures
	ErrConfirmLockOwnership = "can not confirm ownership of %s lock"
)

const (
	// InfoWaitingOnPod is the info message when waiting for pod completion or lock release
	InfoWaitingOnPod = "Waiting on either pod to finish or release of the lock."
	// InfoTestingCompleted is the info message when all testing is completed
	InfoTestingCompleted = "Testing completed. All pods spawned by the test-operator finished."
	// InfoCreatingFirstPod is the info message when creating the first test pod
	InfoCreatingFirstPod = "Creating first test pod (workflow step %d)."
	// InfoCreatingNextPod is the info message when creating subsequent test pods
	InfoCreatingNextPod = "Creating next test pod (workflow step %d)."
	// InfoCanNotAcquireLock is the info message when lock acquisition fails
	InfoCanNotAcquireLock = "Can not acquire %s lock."
	// InfoCanNotReleaseLock is the info message when lock release fails
	InfoCanNotReleaseLock = "Can not release %s lock."
)

const (
	// RequeueAfterValue tells how much time should we wait before calling Reconcile
	// loop again.
	RequeueAfterValue = time.Second * 60
)

// Static error definitions for test operations
var (
	// ErrReceivedUnexpectedAction indicates that an unexpected action was received.
	ErrReceivedUnexpectedAction = errors.New("unexpected action received")

	// ErrFailedToDeleteLock indicates that the test-operator-lock could not be deleted.
	ErrFailedToDeleteLock = errors.New("failed to delete test-operator-lock")

	// ErrNetworkAttachmentsMismatch indicates that not all pods have interfaces with IPs as configured in NetworkAttachments.
	ErrNetworkAttachmentsMismatch = errors.New("not all pods have interfaces with ips as configured in NetworkAttachments")

	// ErrLockFieldMissing indicates that a required field is missing in the lock config map.
	ErrLockFieldMissing = errors.New("field is missing in the config map")

	// ErrFieldExpectedStruct indicates attempting to access a field on a non-struct type.
	ErrFieldExpectedStruct = errors.New("field cannot be accessed: expected struct")

	// ErrFieldNilPointer indicates attempting to dereference a nil pointer.
	ErrFieldNilPointer = errors.New("field cannot be accessed: nil pointer")

	// ErrFieldNotFound indicates a field name does not exist on the struct.
	ErrFieldNotFound = errors.New("field not found")
)

// Reconciler provides common functionality for all test framework reconcilers
type Reconciler struct {
	Client  client.Client
	Kclient kubernetes.Interface
	Log     logr.Logger
	Scheme  *runtime.Scheme
}

// NextAction holds an action that should be performed by the Reconcile loop.
type NextAction int

const (
	// Wait indicates that we should wait for the state of the OpenShift cluster
	// to change
	Wait = iota

	// CreateFirstPod indicates that the Reconcile loop should create the first pod
	// either specified in the .Spec section or in the .Spec.Workflow section.
	CreateFirstPod

	// CreateNextPod indicates that the Reconcile loop should create a next pod
	// specified in the .Spec.Workflow section (if .Spec.Workflow is defined)
	CreateNextPod

	// EndTesting indicates that all pods have already finished. The Reconcile
	// loop should end the testing and release resources that are required to
	// be release (e.g., global lock)
	EndTesting

	// Failure indicates that an unexpected error was encountered
	Failure
)

// GetPod returns pod that has a specific name (podName) in a given namespace
// (podNamespace).
func (r *Reconciler) GetPod(
	ctx context.Context,
	podName string,
	podNamespace string,
) (*corev1.Pod, error) {
	pod := &corev1.Pod{}
	objectKey := client.ObjectKey{Namespace: podNamespace, Name: podName}
	if err := r.Client.Get(ctx, objectKey, pod); err != nil {
		return pod, err
	}

	return pod, nil
}

// CreatePod creates a pod based on a spec provided via PodSpec.
func (r *Reconciler) CreatePod(
	ctx context.Context,
	h helper.Helper,
	podSpec *corev1.Pod,
) (ctrl.Result, error) {
	_, err := r.GetPod(ctx, podSpec.Name, podSpec.Namespace)
	if err == nil {
		return ctrl.Result{}, nil
	} else if !k8s_errors.IsNotFound(err) {
		return ctrl.Result{}, err
	}

	err = controllerutil.SetControllerReference(h.GetBeforeObject(), podSpec, r.GetScheme())
	if err != nil {
		return ctrl.Result{}, err
	}

	if err := r.Client.Create(ctx, podSpec); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// NextAction indicates what action needs to be performed by the Reconcile loop
// based on the current state of the OpenShift cluster.
func (r *Reconciler) NextAction(
	ctx context.Context,
	instance client.Object,
	workflowLength int,
) (NextAction, int, error) {
	// Get the latest pod. The latest pod is pod with the highest value stored
	// in workflowStep label
	workflowStepIdx := 0
	lastPod, err := r.GetLastPod(ctx, instance)
	if err != nil {
		return Failure, workflowStepIdx, err
	}

	// If there is a pod associated with the current instance.
	if lastPod != nil {
		workflowStepIdx, err := strconv.Atoi(lastPod.Labels[workflowStepLabel])
		if err != nil {
			return Failure, workflowStepIdx, err
		}

		// If the last pod is not in Failed or Succeeded state -> Wait
		lastPodFinished := lastPod.Status.Phase == corev1.PodFailed || lastPod.Status.Phase == corev1.PodSucceeded
		if !lastPodFinished {
			return Wait, workflowStepIdx, nil
		}

		// If the last pod is in Failed or Succeeded state and it is NOT the last
		// pod which was supposed to be created -> CreateNextPod
		if lastPodFinished && !isLastPodIndex(workflowStepIdx, workflowLength) {
			workflowStepIdx++
			return CreateNextPod, workflowStepIdx, nil
		}

		// Otherwise if the pod is in Failed or Succeeded state and it IS the
		// last pod -> EndTesting
		if lastPodFinished && isLastPodIndex(workflowStepIdx, workflowLength) {
			return EndTesting, workflowStepIdx, nil
		}
	}

	// If there is not any pod associated with the instance -> createFirstPod
	if lastPod == nil {
		return CreateFirstPod, workflowStepIdx, nil
	}

	return Failure, workflowStepIdx, nil
}

// isLastPodIndex returns true when podIndex is the index of the last pod that
// should be executed. Otherwise the return value is false.
func isLastPodIndex(podIndex int, workflowLength int) bool {
	switch workflowLength {
	case 0:
		return podIndex == workflowLength
	default:
		return podIndex == (workflowLength - 1)
	}
}

// GetLastPod returns pod associated with an instance which has the highest value
// stored in the workflowStep label
func (r *Reconciler) GetLastPod(
	ctx context.Context,
	instance client.Object,
) (*corev1.Pod, error) {
	labels := map[string]string{instanceNameLabel: instance.GetName()}
	namespaceListOpt := client.InNamespace(instance.GetNamespace())
	labelsListOpt := client.MatchingLabels(labels)
	podList := &corev1.PodList{}
	err := r.Client.List(ctx, podList, namespaceListOpt, labelsListOpt)
	if err != nil {
		return nil, err
	}

	var maxPod *corev1.Pod
	maxPodWorkflowStep := 0

	for _, pod := range podList.Items {
		workflowStep, err := strconv.Atoi(pod.Labels[workflowStepLabel])
		if err != nil {
			return &corev1.Pod{}, err
		}

		if workflowStep >= maxPodWorkflowStep {
			maxPodWorkflowStep = workflowStep
			newMaxPod := pod
			maxPod = &newMaxPod
		}
	}

	return maxPod, nil
}

// GetContainerImage returns the container image to use for the given instance, either from the provided parameter or from configuration
func (r *Reconciler) GetContainerImage(
	ctx context.Context,
	instance interface{},
) (string, error) {
	v := reflect.ValueOf(instance)

	spec, err := SafetyCheck(v, "Spec")
	if err != nil {
		return "", err
	}

	containerImage := GetStringField(spec, "ContainerImage")
	if containerImage != "" {
		return containerImage, nil
	}

	namespace := GetStringField(v, "Namespace")
	kind := GetStringField(v, "Kind")

	cm := &corev1.ConfigMap{}
	objectKey := client.ObjectKey{Namespace: namespace, Name: "test-operator-config"}
	if err := r.Client.Get(ctx, objectKey, cm); err != nil {
		return "", err
	}

	imageKey := strings.ToLower(kind) + "-image"
	if cm.Data != nil {
		if image, exists := cm.Data[imageKey]; exists && image != "" {
			return image, nil
		}
	}

	relatedImage := "RELATED_IMAGE_TEST_" + strings.ToUpper(kind) + "_IMAGE_URL_DEFAULT"
	return util.GetEnvVar(relatedImage, ""), nil
}

// GetPodName returns the name of the pod for the given instance and workflow step
func (r *Reconciler) GetPodName(instance interface{}, stepNum int) string {
	v := reflect.ValueOf(instance)

	name := GetStringField(v, "Name")
	spec, err := SafetyCheck(v, "Spec")
	if err != nil {
		return name
	}

	workflow, err := SafetyCheck(spec, "Workflow")
	if err != nil || workflow.Len() == 0 {
		return name
	}

	// Get workflow step name
	stepName := workflowStepNameInvalid
	if stepNum >= 0 && stepNum < workflow.Len() {
		stepName = GetStringField(workflow.Index(stepNum), "StepName")
		if stepName == "" {
			stepName = workflowStepNameInvalid
		}
	}

	return name + podNameStepInfix + fmt.Sprintf("%02d", stepNum) + "-" + stepName
}

// GetPVCLogsName returns the name of the PVC for logs for the given instance and workflow step
func (r *Reconciler) GetPVCLogsName(instance client.Object, workflowStepNum int) string {
	instanceName := instance.GetName()
	instanceCreationTimestamp := instance.GetCreationTimestamp().Format(time.UnixDate)
	suffixLength := 5
	nameSuffix := GetStringHash(instanceName+instanceCreationTimestamp, suffixLength)
	workflowStep := strconv.Itoa(workflowStepNum)
	return instanceName + "-" + workflowStep + "-" + nameSuffix
}

// CheckSecretExists checks if a secret with the given name exists in the same namespace as the instance
func (r *Reconciler) CheckSecretExists(ctx context.Context, instance client.Object, secretName string) bool {
	secret := &corev1.Secret{}
	err := r.Client.Get(ctx, client.ObjectKey{Namespace: instance.GetNamespace(), Name: secretName}, secret)
	if err != nil && k8s_errors.IsNotFound(err) {
		return false
	}

	return true
}

// GetStringHash returns a hash of the given string with the specified length
func GetStringHash(str string, hashLength int) string {
	hash := sha256.New()
	hash.Write([]byte(str))
	byteSlice := hash.Sum(nil)
	hashString := fmt.Sprintf("%x", byteSlice)

	return hashString[:hashLength]
}

// EnsureLogsPVCExists ensures that a PVC for logs exists for the given instance and workflow step
func (r *Reconciler) EnsureLogsPVCExists(
	ctx context.Context,
	instance client.Object,
	helper *helper.Helper,
	labels map[string]string,
	StorageClassName string,
	workflowStepNum int,
) (ctrl.Result, error) {
	instanceNamespace := instance.GetNamespace()
	pvcName := r.GetPVCLogsName(instance, workflowStepNum)

	pvvc := &corev1.PersistentVolumeClaim{}
	err := r.Client.Get(ctx, client.ObjectKey{Namespace: instanceNamespace, Name: pvcName}, pvvc)
	if err == nil {
		return ctrl.Result{}, nil
	}

	pvcAccessMode := []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}

	testOperatorPvcDef := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pvcName,
			Namespace: instanceNamespace,
			Labels:    labels,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: pvcAccessMode,
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: k8sresource.MustParse("1Gi"),
				},
			},
			StorageClassName: &StorageClassName,
		},
	}

	timeDuration, _ := time.ParseDuration("2m")
	testOperatorPvc := pvc.NewPvc(testOperatorPvcDef, timeDuration)
	ctrlResult, err := testOperatorPvc.CreateOrPatch(ctx, helper)
	if err != nil {
		return ctrlResult, err
	} else if (ctrlResult != ctrl.Result{}) {
		return ctrlResult, nil
	}

	return ctrlResult, nil
}

// GetLogger returns the logger instance
func (r *Reconciler) GetLogger() logr.Logger {
	return r.Log
}

// GetScheme returns the runtime scheme
func (r *Reconciler) GetScheme() *runtime.Scheme {
	return r.Scheme
}

// GetLockInfo retrieves the lock information ConfigMap for the given instance
func (r *Reconciler) GetLockInfo(ctx context.Context, instance client.Object) (*corev1.ConfigMap, error) {
	cm := &corev1.ConfigMap{}
	objectKey := client.ObjectKey{Namespace: instance.GetNamespace(), Name: testOperatorLockName}
	err := r.Client.Get(ctx, objectKey, cm)
	if err != nil {
		return cm, err
	}

	if _, ok := cm.Data[testOperatorLockOwnerField]; !ok {
		return cm, fmt.Errorf("%w: %s field is missing in the %s config map", ErrLockFieldMissing, testOperatorLockOwnerField, testOperatorLockName)
	}

	return cm, err
}

// AcquireLock attempts to acquire a lock for the given instance to prevent concurrent operations
func (r *Reconciler) AcquireLock(
	ctx context.Context,
	instance client.Object,
	h *helper.Helper,
	parallel bool,
) (bool, error) {
	// Do not wait for the lock if the user wants the tests to be
	// executed parallely
	if parallel {
		return true, nil
	}

	instanceGUID := string(instance.GetUID())
	cm, err := r.GetLockInfo(ctx, instance)
	if err != nil && k8s_errors.IsNotFound(err) {
		cm := map[string]string{
			testOperatorLockOwnerField: instanceGUID,
		}

		cms := []util.Template{
			{
				Name:       testOperatorLockName,
				Namespace:  instance.GetNamespace(),
				CustomData: cm,
			},
		}

		err = configmap.EnsureConfigMaps(ctx, h, instance, cms, nil)
		return err == nil, err
	}

	if cm.Data[testOperatorLockOwnerField] == instanceGUID {
		return true, nil
	}

	return false, err
}

// ReleaseLock releases the lock for the given instance
func (r *Reconciler) ReleaseLock(ctx context.Context, instance client.Object) (bool, error) {
	Log := r.GetLogger()

	cm, err := r.GetLockInfo(ctx, instance)
	if err != nil && k8s_errors.IsNotFound(err) {
		return true, nil
	} else if err != nil {
		return false, err
	}

	// Lock can be only released by the instance that created it
	if cm.Data[testOperatorLockOwnerField] != string(instance.GetUID()) {
		return false, nil
	}

	err = r.Client.Delete(ctx, cm)
	if err != nil && k8s_errors.IsNotFound(err) {
		return false, nil
	}

	// Check whether the lock was successfully deleted
	maxRetries := 10
	lockDeletionSleepPeriod := 10
	for i := 0; i < maxRetries; i++ {
		_, err = r.GetLockInfo(ctx, instance)
		if err != nil && k8s_errors.IsNotFound(err) {
			return true, nil
		}

		time.Sleep(time.Second * time.Duration(lockDeletionSleepPeriod))
		Log.Info("Waiting for the test-operator-lock deletion!")
	}

	return false, ErrFailedToDeleteLock
}

// PodExists checks if a pod exists for the given instance and workflow step
func (r *Reconciler) PodExists(ctx context.Context, instance client.Object, workflowStepNum int) bool {
	pod := &corev1.Pod{}
	podName := r.GetPodName(instance, workflowStepNum)
	objectKey := client.ObjectKey{Namespace: instance.GetNamespace(), Name: podName}
	err := r.Client.Get(ctx, objectKey, pod)
	if err != nil && k8s_errors.IsNotFound(err) {
		return false
	}

	return true
}

// GetCommonRbacRules returns the common RBAC rules for test operations, with optional privileged permissions
func GetCommonRbacRules(privileged bool) []rbacv1.PolicyRule {
	rbacPolicyRule := rbacv1.PolicyRule{
		APIGroups:     []string{"security.openshift.io"},
		ResourceNames: []string{"nonroot", "nonroot-v2"},
		Resources:     []string{"securitycontextconstraints"},
		Verbs:         []string{"use"},
	}

	if privileged {
		rbacPolicyRule.ResourceNames = append(
			rbacPolicyRule.ResourceNames,
			[]string{"anyuid", "privileged"}...)
	}

	return []rbacv1.PolicyRule{rbacPolicyRule}
}

// EnsureNetworkAttachments fetches NetworkAttachmentDefinitions and creates annotations
func (r *Reconciler) EnsureNetworkAttachments(
	ctx context.Context,
	log logr.Logger,
	helper *helper.Helper,
	networkAttachments []string,
	namespace string,
	conditions *condition.Conditions,
) (map[string]string, ctrl.Result, error) {
	nadList := []networkv1.NetworkAttachmentDefinition{}
	for _, netAtt := range networkAttachments {
		netAttachDef, err := nad.GetNADWithName(ctx, helper, netAtt, namespace)
		if err != nil {
			if k8s_errors.IsNotFound(err) {
				// Since the net-attach-def CR should have been manually created by the user and referenced in the spec,
				// we treat this as a warning because it means that the service will not be able to start.
				log.Info(fmt.Sprintf("network-attachment-definition %s not found", netAtt))
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

		if netAttachDef != nil {
			nadList = append(nadList, *netAttachDef)
		}
	}

	serviceAnnotations, err := nad.EnsureNetworksAnnotation(nadList)
	if err != nil {
		return nil, ctrl.Result{}, fmt.Errorf("failed create network annotation from %s: %w",
			networkAttachments, err)
	}

	conditions.MarkTrue(condition.NetworkAttachmentsReadyCondition, condition.NetworkAttachmentsReadyMessage)

	return serviceAnnotations, ctrl.Result{}, nil
}

// VerifyNetworkAttachments verifies network status on the pod and updates conditions
func (r *Reconciler) VerifyNetworkAttachments(
	ctx context.Context,
	helper *helper.Helper,
	instance client.Object,
	networkAttachments []string,
	serviceLabels map[string]string,
	nextWorkflowStep int,
	conditions *condition.Conditions,
	networkAttachmentStatus *map[string][]string,
) (ctrl.Result, error) {
	if !r.PodExists(ctx, instance, nextWorkflowStep) {
		return ctrl.Result{}, nil
	}

	networkReady, status, err := nad.VerifyNetworkStatusFromAnnotation(
		ctx,
		helper,
		networkAttachments,
		serviceLabels,
		1,
	)
	if err != nil {
		return ctrl.Result{}, err
	}

	*networkAttachmentStatus = status

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

		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// EnsureCloudsConfigMapExists ensures that frameworks like Tobiko and Horizon have password values
// present in clouds.yaml. This code ensures that we set a default value of
// 12345678 when password value is missing in the clouds.yaml
func EnsureCloudsConfigMapExists(
	ctx context.Context,
	instance client.Object,
	helper *helper.Helper,
	labels map[string]string,
) (ctrl.Result, error) {
	const openstackConfigMapName = "openstack-config"
	const testOperatorCloudsConfigMapName = "test-operator-clouds-config"

	cm, _, _ := configmap.GetConfigMap(
		ctx,
		helper,
		instance,
		testOperatorCloudsConfigMapName,
		time.Second*10,
	)
	if cm.Name == testOperatorCloudsConfigMapName {
		return ctrl.Result{}, nil
	}

	cm, _, _ = configmap.GetConfigMap(
		ctx,
		helper,
		instance,
		openstackConfigMapName,
		time.Second*10,
	)

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
			Name:      testOperatorCloudsConfigMapName,
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

// Int64OrPlaceholder converts int64 to string, returns placeholder if 0
func Int64OrPlaceholder(value int64, placeholder string) string {
	if value > 0 {
		return strconv.FormatInt(value, 10)
	}
	return placeholder
}

// StringOrPlaceholder returns value if non-empty, otherwise placeholder
func StringOrPlaceholder(value, placeholder string) string {
	if value != "" {
		return value
	}
	return placeholder
}

// SetBoolEnvVars sets boolean values as string environment variables
func SetBoolEnvVars(envVars map[string]env.Setter, boolVars map[string]bool) {
	for key, value := range boolVars {
		envVars[key] = env.SetValue(strconv.FormatBool(value))
	}
}

// SetStringEnvVars sets string environment variables
func SetStringEnvVars(envVars map[string]env.Setter, stringVars map[string]string) {
	for key, value := range stringVars {
		envVars[key] = env.SetValue(value)
	}
}

// SetFileEnvVar sets a file in customData and creates an env var
func SetFileEnvVar(
	customData map[string]string,
	envVars map[string]string,
	content string,
	filename string,
	envVarName string,
) {
	if len(content) == 0 {
		return
	}
	customData[filename] = content
	envVars[envVarName] = testOperatorBaseDir + filename
}

// SetDictEnvVar sets dictionary env vars
func SetDictEnvVar(envVars map[string]string, fields map[string]string) {
	for key, value := range fields {
		envVars[key] += value + ","
	}
}

// GetStringField returns reflect string field safely
func GetStringField(v reflect.Value, fieldName string) string {
	field, err := SafetyCheck(v, fieldName)
	if err != nil || field.Kind() != reflect.String {
		return ""
	}

	return field.String()
}

// SafetyCheck returns reflect value after checking its validity
func SafetyCheck(v reflect.Value, fieldName string) (reflect.Value, error) {
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return reflect.Value{}, fmt.Errorf("%s: %w", fieldName, ErrFieldNilPointer)
		}
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return reflect.Value{}, fmt.Errorf("%s: %w, got %s", fieldName, ErrFieldExpectedStruct, v.Kind())
	}

	field := v.FieldByName(fieldName)
	if !field.IsValid() {
		return reflect.Value{}, fmt.Errorf("%s: %w", fieldName, ErrFieldNotFound)
	}

	return field, nil
}

// IsEmpty checks if the provided value is empty based on its type
func IsEmpty(value interface{}) bool {
	if v, ok := value.(reflect.Value); ok {
		switch v.Kind() {
		case reflect.String, reflect.Map:
			return v.Len() == 0
		case reflect.Ptr, reflect.Interface, reflect.Slice:
			return v.IsNil()
		}
	}
	return false
}

// MergeSections iterates through all CR parameters and overrides them
// with non-empty values from the workflow section of the current step.
func MergeSections(main interface{}, workflow interface{}) {
	mReflect := reflect.ValueOf(main).Elem()
	wReflect := reflect.ValueOf(workflow)

	for i := 0; i < mReflect.NumField(); i++ {
		name := mReflect.Type().Field(i).Name
		mValue := mReflect.Field(i)
		wValue := wReflect.FieldByName(name)

		if mValue.Kind() == reflect.Struct && wValue.Kind() != reflect.Ptr {
			switch name {
			case "CommonOptions":
				wValue := wReflect.FieldByName("WorkflowCommonOptions")
				MergeSections(mValue.Addr().Interface(), wValue.Interface())
			case "TempestRun", "TempestconfRun":
				MergeSections(mValue.Addr().Interface(), wValue.Interface())
			}
			continue
		}

		if wValue.IsValid() && !IsEmpty(wValue) {
			if wValue.Kind() == reflect.Ptr && mValue.Kind() != reflect.Ptr {
				wValue = wValue.Elem()
			}
			mValue.Set(wValue)
		}
	}
}
