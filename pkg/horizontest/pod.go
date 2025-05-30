package horizontest

import (
	"github.com/openstack-k8s-operators/lib-common/modules/common/env"

	testv1beta1 "github.com/openstack-k8s-operators/test-operator/api/v1beta1"
	util "github.com/openstack-k8s-operators/test-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Pod - prepare pod to run Horizon tests
func Pod(
	instance *testv1beta1.HorizonTest,
	labels map[string]string,
	podName string,
	logsPVCName string,
	mountCerts bool,
	mountKeys bool,
	mountKubeconfig bool,
	envVars map[string]env.Setter,
	containerImage string,
) *corev1.Pod {

	runAsUser := int64(42455)
	runAsGroup := int64(42455)

	capabilities := []corev1.Capability{"NET_ADMIN", "NET_RAW"}
	securityContext := util.GetSecurityContext(runAsUser, capabilities, instance.Spec.Privileged)

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: instance.Namespace,
			Labels:    labels,
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
					Name:            instance.Name,
					Image:           containerImage,
					Args:            []string{},
					Env:             env.MergeEnvs([]corev1.EnvVar{}, envVars),
					VolumeMounts:    GetVolumeMounts(mountCerts, mountKeys, mountKubeconfig, HorizonTestPropagation, instance),
					SecurityContext: &securityContext,
					Resources:       instance.Spec.Resources,
				},
			},
			Volumes: GetVolumes(
				instance,
				logsPVCName,
				mountCerts,
				mountKubeconfig,
				HorizonTestPropagation,
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
