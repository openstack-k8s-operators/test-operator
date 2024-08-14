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
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	k8sresource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	workflowNameSuffix       = "-workflow-counter"
	jobNameStepInfix         = "-workflow-step-"
	logDirNameInfix          = "-workflow-step-"
	envVarsConfigMapinfix    = "-env-vars-step-"
	customDataConfigMapinfix = "-custom-data-step-"
	workflowStepNumInvalid   = -1

	testOperatorLockName       = "test-operator-lock"
	testOperatorLockOnwerField = "owner"
)

type Reconciler struct {
	Client  client.Client
	Kclient kubernetes.Interface
	Log     logr.Logger
	Scheme  *runtime.Scheme
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

func (r *Reconciler) GetJobName(instance interface{}, workflowStepNum int) string {
	if typedInstance, ok := instance.(*v1beta1.Tobiko); ok {
		if len(typedInstance.Spec.Workflow) == 0 || workflowStepNum == workflowStepNumInvalid {
			return typedInstance.Name
		}

		workflowStepName := typedInstance.Spec.Workflow[workflowStepNum].StepName
		return typedInstance.Name + "-" + workflowStepName + jobNameStepInfix + strconv.Itoa(workflowStepNum)
	} else if typedInstance, ok := instance.(*v1beta1.Tempest); ok {
		if len(typedInstance.Spec.Workflow) == 0 || workflowStepNum == workflowStepNumInvalid {
			return typedInstance.Name
		}

		workflowStepName := typedInstance.Spec.Workflow[workflowStepNum].StepName
		return typedInstance.Name + "-" + workflowStepName + jobNameStepInfix + strconv.Itoa(workflowStepNum)
	} else if typedInstance, ok := instance.(*v1beta1.HorizonTest); ok {
		return typedInstance.Name
	} else if typedInstance, ok := instance.(*v1beta1.AnsibleTest); ok {
		if len(typedInstance.Spec.Workflow) == 0 || workflowStepNum == workflowStepNumInvalid {
			return typedInstance.Name
		}

		workflowStepName := typedInstance.Spec.Workflow[workflowStepNum].StepName
		return typedInstance.Name + "-" + workflowStepName + jobNameStepInfix + strconv.Itoa(workflowStepNum)
	}

	return ""
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
			Resources: corev1.ResourceRequirements{
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

	_, err := r.GetLockInfo(ctx, instance)
	if err != nil && k8s_errors.IsNotFound(err) {
		cm := map[string]string{
			testOperatorLockOnwerField: string(instance.GetUID()),
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

	return false, err
}

func (r *Reconciler) ReleaseLock(ctx context.Context, instance client.Object) (bool, error) {
	Log := r.GetLogger()

	cm, err := r.GetLockInfo(ctx, instance)
	if err != nil && k8s_errors.IsNotFound(err) {
		return false, nil
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

func (r *Reconciler) WorkflowStepCounterCreate(ctx context.Context, instance client.Object, h *helper.Helper) bool {
	cm := &corev1.ConfigMap{}
	err := r.Client.Get(ctx, client.ObjectKey{Namespace: instance.GetNamespace(), Name: r.GetWorkflowConfigMapName(instance)}, cm)
	if err == nil {
		return true
	}

	counterData := make(map[string]string)
	counterData["counter"] = "0"

	cms := []util.Template{
		{
			Name:       r.GetWorkflowConfigMapName(instance),
			Namespace:  instance.GetNamespace(),
			CustomData: counterData,
		},
	}

	err = configmap.EnsureConfigMaps(ctx, h, instance, cms, nil)
	return err == nil
}

func (r *Reconciler) WorkflowStepCounterIncrease(ctx context.Context, instance client.Object, h *helper.Helper) bool {
	cm := &corev1.ConfigMap{}
	err := r.Client.Get(ctx, client.ObjectKey{Namespace: instance.GetNamespace(), Name: r.GetWorkflowConfigMapName(instance)}, cm)
	if err != nil {
		return false
	}

	counterValue, _ := strconv.Atoi(cm.Data["counter"])
	newCounterValue := strconv.Itoa(counterValue + 1)
	cm.Data["counter"] = newCounterValue

	cms := []util.Template{
		{
			Name:       r.GetWorkflowConfigMapName(instance),
			Namespace:  instance.GetNamespace(),
			CustomData: cm.Data,
		},
	}

	err = configmap.EnsureConfigMaps(ctx, h, instance, cms, nil)
	return err == nil
}

func (r *Reconciler) WorkflowStepCounterRead(ctx context.Context, instance client.Object) int {
	cm := &corev1.ConfigMap{}
	err := r.Client.Get(ctx, client.ObjectKey{Namespace: instance.GetNamespace(), Name: r.GetWorkflowConfigMapName(instance)}, cm)
	if err != nil {
		return workflowStepNumInvalid
	}

	counter, _ := strconv.Atoi(cm.Data["counter"])
	return counter
}

func (r *Reconciler) CompletedJobExists(ctx context.Context, instance client.Object, workflowStepNum int) bool {
	job := &batchv1.Job{}
	err := r.Client.Get(ctx, client.ObjectKey{Namespace: instance.GetNamespace(), Name: r.GetJobName(instance, workflowStepNum)}, job)

	if err != nil {
		return false
	}

	if job.Status.Succeeded > 0 || job.Status.Failed > 0 {
		return true
	}

	return false
}

func (r *Reconciler) JobExists(ctx context.Context, instance client.Object, workflowStepNum int) bool {
	job := &batchv1.Job{}
	jobName := r.GetJobName(instance, workflowStepNum)
	objectKey := client.ObjectKey{Namespace: instance.GetNamespace(), Name: jobName}
	err := r.Client.Get(ctx, objectKey, job)
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

func (r *Reconciler) OverwriteValueWithWorkflow(
	instance v1beta1.TobikoSpec,
	sectionName string,
	workflowValueType string,
	workflowStepNum int,
) interface{} {
	if len(instance.Workflow)-1 < workflowStepNum {
		reflected := reflect.ValueOf(instance)
		fieldValue := reflected.FieldByName(sectionName)
		return fieldValue.Interface()
	}

	reflected := reflect.ValueOf(instance)
	tobikoSpecValue := reflected.FieldByName(sectionName).Interface()

	reflected = reflect.ValueOf(instance.Workflow[workflowStepNum])
	tobikoWorkflowValue := reflected.FieldByName(sectionName).Interface()

	if workflowValueType == "pbool" {
		if val, ok := tobikoWorkflowValue.(*bool); ok && val != nil {
			return *(tobikoWorkflowValue.(*bool))
		}
		return tobikoSpecValue.(bool)
	} else if workflowValueType == "puint8" {
		if val, ok := tobikoWorkflowValue.(*uint8); ok && val != nil {
			return *(tobikoWorkflowValue.(*uint8))
		}
		return tobikoSpecValue
	} else if workflowValueType == "string" {
		if val, ok := tobikoWorkflowValue.(string); ok && val != "" {
			return tobikoWorkflowValue
		}
		return tobikoSpecValue
	}

	return nil
}
