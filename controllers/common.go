package controllers

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"time"

	"crypto/sha256"

	"github.com/go-logr/logr"
	"github.com/openstack-k8s-operators/lib-common/modules/common/configmap"
	"github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	"github.com/openstack-k8s-operators/lib-common/modules/common/pvc"
	"github.com/openstack-k8s-operators/lib-common/modules/common/util"
	v1beta1 "github.com/openstack-k8s-operators/test-operator/api/v1beta1"
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
	workflowNameSuffix       = "-workflow-counter"
	podNameStepInfix         = "-workflow-step-"
	envVarsConfigMapinfix    = "-env-vars-step-"
	customDataConfigMapinfix = "-custom-data-step-"
	workflowStepNumInvalid   = -1
	workflowStepNameInvalid  = "no-step-name"
	workflowStepLabel        = "workflowStep"
	instanceNameLabel        = "instanceName"
	operatorNameLabel        = "operator"

	testOperatorLockName       = "test-operator-lock"
	testOperatorLockOnwerField = "owner"
)

const (
	ErrNetworkAttachments       = "not all pods have interfaces with ips as configured in NetworkAttachments: %s"
	ErrReceivedUnexpectedAction = "unexpected action received"
	ErrConfirmLockOwnership     = "can not confirm ownership of %s lock"
)

const (
	InfoWaitingOnPod      = "Waiting on either pod to finish or release of the lock."
	InfoTestingCompleted  = "Testing completed. All pods spawned by the test-operator finished."
	InfoCreatingFirstPod  = "Creating first test pod (workflow step %d)."
	InfoCreatingNextPod   = "Creating next test pod (workflow step %d)."
	InfoCanNotAcquireLock = "Can not acquire %s lock."
	InfoCanNotReleaseLock = "Can not release %s lock."
)

const (
	// RequeueAfterValue tells how much time should we wait before calling Reconcile
	// loop again.
	RequeueAfterValue = time.Second * 60
)

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

		// If the last pod is not in Failed or Succeded state -> Wait
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

		// Otherwise if the pod is in Failed or Succeded stated and it IS the
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

func GetEnvVarsConfigMapName(instance interface{}, workflowStepNum int) string {
	if _, ok := instance.(*v1beta1.Tobiko); ok {
		return "not-implemented"
	} else if typedInstance, ok := instance.(*v1beta1.Tempest); ok {
		return typedInstance.Name + envVarsConfigMapinfix + strconv.Itoa(workflowStepNum)
	}

	return "not-implemented"
}

func GetCustomDataConfigMapName(instance interface{}, workflowStepNum int) string {
	if _, ok := instance.(*v1beta1.Tobiko); ok {
		return "not-implemented"
	} else if typedInstance, ok := instance.(*v1beta1.Tempest); ok {
		return typedInstance.Name + customDataConfigMapinfix + strconv.Itoa(workflowStepNum)
	}

	return "not-implemented"
}

func (r *Reconciler) GetContainerImage(
	ctx context.Context,
	containerImage string,
	instance interface{},
) (string, error) {
	cm := &corev1.ConfigMap{}
	testOperatorConfigMapName := "test-operator-config"
	if typedInstance, ok := instance.(*v1beta1.Tempest); ok {
		if len(containerImage) > 0 {
			return containerImage, nil
		}

		objectKey := client.ObjectKey{Namespace: typedInstance.Namespace, Name: testOperatorConfigMapName}
		err := r.Client.Get(ctx, objectKey, cm)
		if err != nil {
			return "", err
		}

		if cm.Data == nil {
			return util.GetEnvVar("RELATED_IMAGE_TEST_TEMPEST_IMAGE_URL_DEFAULT", ""), nil

		}

		if cmImage, exists := cm.Data["tempest-image"]; exists {
			return cmImage, nil
		}

		return util.GetEnvVar("RELATED_IMAGE_TEST_TEMPEST_IMAGE_URL_DEFAULT", ""), nil
	} else if typedInstance, ok := instance.(*v1beta1.Tobiko); ok {
		if len(containerImage) > 0 {
			return containerImage, nil
		}

		objectKey := client.ObjectKey{Namespace: typedInstance.Namespace, Name: testOperatorConfigMapName}
		err := r.Client.Get(ctx, objectKey, cm)
		if err != nil {
			return "", err
		}

		if cm.Data == nil {
			return util.GetEnvVar("RELATED_IMAGE_TEST_TOBIKO_IMAGE_URL_DEFAULT", ""), nil

		}

		if cmImage, exists := cm.Data["tobiko-image"]; exists {
			return cmImage, nil
		}

		return util.GetEnvVar("RELATED_IMAGE_TEST_TOBIKO_IMAGE_URL_DEFAULT", ""), nil
	} else if typedInstance, ok := instance.(*v1beta1.HorizonTest); ok {
		if len(containerImage) > 0 {
			return containerImage, nil
		}

		objectKey := client.ObjectKey{Namespace: typedInstance.Namespace, Name: testOperatorConfigMapName}
		err := r.Client.Get(ctx, objectKey, cm)
		if err != nil {
			return "", err
		}

		if cm.Data == nil {
			return util.GetEnvVar("RELATED_IMAGE_TEST_HORIZONTEST_IMAGE_URL_DEFAULT", ""), nil

		}

		if cmImage, exists := cm.Data["horizontest-image"]; exists {
			return cmImage, nil
		}

		return util.GetEnvVar("RELATED_IMAGE_TEST_HORIZONTEST_IMAGE_URL_DEFAULT", ""), nil
	} else if typedInstance, ok := instance.(*v1beta1.AnsibleTest); ok {
		if len(containerImage) > 0 {
			return containerImage, nil
		}

		objectKey := client.ObjectKey{Namespace: typedInstance.Namespace, Name: testOperatorConfigMapName}
		err := r.Client.Get(ctx, objectKey, cm)
		if err != nil {
			return "", err
		}

		if cm.Data == nil {
			return util.GetEnvVar("RELATED_IMAGE_TEST_ANSIBLETEST_IMAGE_URL_DEFAULT", ""), nil
		}

		if cmImage, exists := cm.Data["ansibletest-image"]; exists {
			return cmImage, nil
		}

		return util.GetEnvVar("RELATED_IMAGE_TEST_ANSIBLETEST_IMAGE_URL_DEFAULT", ""), nil
	}

	return "", nil
}

func (r *Reconciler) GetPodName(instance interface{}, workflowStepNum int) string {
	if typedInstance, ok := instance.(*v1beta1.Tobiko); ok {
		if len(typedInstance.Spec.Workflow) == 0 || workflowStepNum == workflowStepNumInvalid {
			return typedInstance.Name
		}

		workflowStepName := workflowStepNameInvalid
		if workflowStepNum < len(typedInstance.Spec.Workflow) {
			workflowStepName = typedInstance.Spec.Workflow[workflowStepNum].StepName
		}

		return typedInstance.Name + podNameStepInfix + fmt.Sprintf("%02d", workflowStepNum) + "-" + workflowStepName
	} else if typedInstance, ok := instance.(*v1beta1.Tempest); ok {
		if len(typedInstance.Spec.Workflow) == 0 || workflowStepNum == workflowStepNumInvalid {
			return typedInstance.Name
		}

		workflowStepName := workflowStepNameInvalid
		if workflowStepNum < len(typedInstance.Spec.Workflow) {
			workflowStepName = typedInstance.Spec.Workflow[workflowStepNum].StepName
		}

		return typedInstance.Name + podNameStepInfix + fmt.Sprintf("%02d", workflowStepNum) + "-" + workflowStepName
	} else if typedInstance, ok := instance.(*v1beta1.HorizonTest); ok {
		return typedInstance.Name
	} else if typedInstance, ok := instance.(*v1beta1.AnsibleTest); ok {
		if len(typedInstance.Spec.Workflow) == 0 || workflowStepNum == workflowStepNumInvalid {
			return typedInstance.Name
		}

		workflowStepName := workflowStepNameInvalid
		if workflowStepNum < len(typedInstance.Spec.Workflow) {
			workflowStepName = typedInstance.Spec.Workflow[workflowStepNum].StepName
		}

		return typedInstance.Name + podNameStepInfix + fmt.Sprintf("%02d", workflowStepNum) + "-" + workflowStepName
	}

	return workflowStepNameInvalid
}

func (r *Reconciler) GetWorkflowConfigMapName(instance client.Object) string {
	return instance.GetName() + workflowNameSuffix
}

func (r *Reconciler) GetPVCLogsName(instance client.Object, workflowStepNum int) string {
	instanceName := instance.GetName()
	instanceCreationTimestamp := instance.GetCreationTimestamp().Format(time.UnixDate)
	suffixLength := 5
	nameSuffix := GetStringHash(instanceName+instanceCreationTimestamp, suffixLength)
	workflowStep := strconv.Itoa(workflowStepNum)
	return instanceName + "-" + workflowStep + "-" + nameSuffix
}

func (r *Reconciler) CheckSecretExists(ctx context.Context, instance client.Object, secretName string) bool {
	secret := &corev1.Secret{}
	err := r.Client.Get(ctx, client.ObjectKey{Namespace: instance.GetNamespace(), Name: secretName}, secret)
	if err != nil && k8s_errors.IsNotFound(err) {
		return false
	}

	return true
}

func GetStringHash(str string, hashLength int) string {
	hash := sha256.New()
	hash.Write([]byte(str))
	byteSlice := hash.Sum(nil)
	hashString := fmt.Sprintf("%x", byteSlice)

	return hashString[:hashLength]
}

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

func (r *Reconciler) GetClient() client.Client {
	return r.Client
}

func (r *Reconciler) GetLogger() logr.Logger {
	return r.Log
}

func (r *Reconciler) GetScheme() *runtime.Scheme {
	return r.Scheme
}

func (r *Reconciler) GetDefaultBool(variable bool) string {
	if variable {
		return "true"
	}

	return "false"
}

func (r *Reconciler) GetDefaultInt(variable int64, defaultValue ...string) string {
	if variable != 0 {
		return strconv.FormatInt(variable, 10)
	} else if len(defaultValue) > 0 {
		return defaultValue[0]
	}

	return ""
}

func (r *Reconciler) GetLockInfo(ctx context.Context, instance client.Object) (*corev1.ConfigMap, error) {
	cm := &corev1.ConfigMap{}
	objectKey := client.ObjectKey{Namespace: instance.GetNamespace(), Name: testOperatorLockName}
	err := r.Client.Get(ctx, objectKey, cm)
	if err != nil {
		return cm, err
	}

	if _, ok := cm.Data[testOperatorLockOnwerField]; !ok {
		errMsg := fmt.Sprintf(
			"%s field is missing in the %s config map",
			testOperatorLockOnwerField, testOperatorLockName,
		)

		return cm, errors.New(errMsg)
	}

	return cm, err
}

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
			testOperatorLockOnwerField: instanceGUID,
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

	if cm.Data[testOperatorLockOnwerField] == instanceGUID {
		return true, nil
	}

	return false, err
}

func (r *Reconciler) ReleaseLock(ctx context.Context, instance client.Object) (bool, error) {
	Log := r.GetLogger()

	cm, err := r.GetLockInfo(ctx, instance)
	if err != nil && k8s_errors.IsNotFound(err) {
		return true, nil
	} else if err != nil {
		return false, err
	}

	// Lock can be only released by the instance that created it
	if cm.Data[testOperatorLockOnwerField] != string(instance.GetUID()) {
		return false, nil
	}

	err = r.Client.Delete(ctx, cm)
	if err != nil && k8s_errors.IsNotFound(err) {
		return false, nil
	}

	// Check whether the lock was successfully deleted deleted
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

	return false, errors.New("failed to delete test-operator-lock")
}

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

func (r *Reconciler) setConfigOverwrite(customData map[string]string, configOverwrite map[string]string) {
	for key, data := range configOverwrite {
		customData[key] = data
	}
}

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

// Some frameworks like (e.g., Tobiko and Horizon) require password value to be
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
