package controllers

import (
	"context"
	// "fmt"
	"strconv"
	"time"

	"github.com/go-logr/logr"
	"github.com/openstack-k8s-operators/lib-common/modules/common/configmap"
	"github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	"github.com/openstack-k8s-operators/lib-common/modules/common/pvc"
	"github.com/openstack-k8s-operators/lib-common/modules/common/util"
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

type Reconciler struct {
	Client  client.Client
	Kclient kubernetes.Interface
	Log     logr.Logger
	Scheme  *runtime.Scheme
}

func (r *Reconciler) CheckSecretExists(ctx context.Context, instance client.Object, secretName string) bool {
	secret := &corev1.Secret{}
	err := r.Client.Get(ctx, client.ObjectKey{Namespace: instance.GetNamespace(), Name: secretName}, secret)
	if err != nil && k8s_errors.IsNotFound(err) {
		return false
	} else {
		return true
	}
}

func (r *Reconciler) EnsureLogsPVCExists(ctx context.Context, instance client.Object, helper *helper.Helper, NamePVC string) (ctrl.Result, error) {
	pvvc := &corev1.PersistentVolumeClaim{}
	err := r.Client.Get(ctx, client.ObjectKey{Namespace: instance.GetNamespace(), Name: NamePVC}, pvvc)
	if err == nil {
		return ctrl.Result{}, nil
	}

	testOperatorPvcDef := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      NamePVC,
			Namespace: instance.GetNamespace(),
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
				corev1.ReadWriteMany,
			},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: k8sresource.MustParse("1Gi"),
				},
			},
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
	} else {
		return "false"
	}
}

func (r *Reconciler) GetDefaultInt(variable int64) string {
	if variable != -1 {
		return strconv.FormatInt(variable, 10)
	} else {
		return ""
	}
}

func (r *Reconciler) AcquireLock(ctx context.Context, instance client.Object, h *helper.Helper, parallel bool) bool {
	// Do not wait for the lock if the user wants the tests to be
	// executed parallely
	if parallel {
		return true
	}

	for {
		cm := &corev1.ConfigMap{}
		err := r.Client.Get(ctx, client.ObjectKey{Namespace: instance.GetNamespace(), Name: "test-operator-lock"}, cm)
		if err != nil {
			if k8s_errors.IsNotFound(err) {
				cms := []util.Template{
					{
						Name:      "test-operator-lock",
						Namespace: instance.GetNamespace(),
					},
				}
				configmap.EnsureConfigMaps(ctx, h, instance, cms, nil)
				return true
			} else {
				return false
			}
		}

		return false
	}
}

func (r *Reconciler) ReleaseLock(ctx context.Context, instance client.Object) bool {
	cm := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: instance.GetNamespace(),
			Name:      "test-operator-lock",
		},
	}

	r.Client.Delete(ctx, &cm)
	return true
}

func (r *Reconciler) CompletedJobExists(ctx context.Context, instance client.Object) bool {
	job := &batchv1.Job{}
	err := r.Client.Get(ctx, client.ObjectKey{Namespace: instance.GetNamespace(), Name: instance.GetName()}, job)
	if err != nil {
		return false
	}

	if job.Status.Succeeded > 0 || job.Status.Failed > 0 {
		return true
	}

	return false
}

func (r *Reconciler) JobExists(ctx context.Context, instance client.Object) bool {
	job := &batchv1.Job{}
	err := r.Client.Get(ctx, client.ObjectKey{Namespace: instance.GetNamespace(), Name: instance.GetName()}, job)
	if err != nil {
		if k8s_errors.IsNotFound(err) {
			return false
		} else {
			return false
		}
	}

	return true
}
