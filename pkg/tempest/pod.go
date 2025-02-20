package tempest

import (
	"github.com/openstack-k8s-operators/lib-common/modules/common/env"

	testv1beta1 "github.com/openstack-k8s-operators/test-operator/api/v1beta1"
	util "github.com/openstack-k8s-operators/test-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Pod - prepare pod to run Tempest tests
func Pod(
	instance *testv1beta1.Tempest,
	labels map[string]string,
	annotations map[string]string,
	podName string,
	envVarsConfigMapName string,
	customDataConfigMapName string,
	logsPVCName string,
	mountCerts bool,
	mountSSHKey bool,
	containerImage string,
) *corev1.Pod {

	envVars := map[string]env.Setter{}
	runAsUser := int64(42480)
	runAsGroup := int64(42480)
	securityContext := util.GetSecurityContext(runAsUser, []corev1.Capability{}, instance.Spec.Privileged)

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: annotations,
			Name:        podName,
			Namespace:   instance.Namespace,
			Labels:      labels,
		},
		Spec: corev1.PodSpec{
			AutomountServiceAccountToken: &instance.Spec.Privileged,
			RestartPolicy:                corev1.RestartPolicyNever,
			Tolerations:                  instance.Spec.Tolerations,
			NodeSelector:                 instance.Spec.NodeSelector,
			SecurityContext: &corev1.PodSecurityContext{
				RunAsUser:  &runAsUser,
				RunAsGroup: &runAsGroup,
				FSGroup:    &runAsGroup,
			},
			Containers: []corev1.Container{
				{
					Name:            instance.Name + "-tests-runner",
					Image:           containerImage,
					Args:            []string{},
					Env:             env.MergeEnvs([]corev1.EnvVar{}, envVars),
					VolumeMounts:    GetVolumeMounts(mountCerts, mountSSHKey, TempestPropagation, instance),
					SecurityContext: &securityContext,
					Resources:       instance.Spec.Resources,
					EnvFrom: []corev1.EnvFromSource{
						{
							ConfigMapRef: &corev1.ConfigMapEnvSource{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: customDataConfigMapName,
								},
							},
						},
						{
							ConfigMapRef: &corev1.ConfigMapEnvSource{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: envVarsConfigMapName,
								},
							},
						},
					},
				},
			},
			Volumes: GetVolumes(
				instance,
				customDataConfigMapName,
				logsPVCName,
				mountCerts,
				mountSSHKey,
				TempestPropagation,
			),
		},
	}

	if len(instance.Spec.SELinuxLevel) > 0 {
		pod.Spec.SecurityContext.SELinuxOptions = &corev1.SELinuxOptions{
			Level: instance.Spec.SELinuxLevel,
		}
	}

	return pod
}
