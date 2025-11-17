package tempest

import (
	"github.com/openstack-k8s-operators/lib-common/modules/common/env"

	testv1beta1 "github.com/openstack-k8s-operators/test-operator/api/v1beta1"
	util "github.com/openstack-k8s-operators/test-operator/internal/util"
	corev1 "k8s.io/api/core/v1"
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
	envFromSource := []corev1.EnvFromSource{
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
	}

	return util.BuildTestPod(
		annotations,
		PodCapabilities,
		containerImage,
		instance.Name+"-tests-runner",
		envFromSource,
		map[string]env.Setter{},
		labels,
		instance.Namespace,
		instance.Spec.NodeSelector,
		podName,
		instance.Spec.Privileged,
		instance.Spec.Resources,
		PodRunAsGroup,
		PodRunAsUser,
		instance.Spec.SELinuxLevel,
		instance.Spec.Tolerations,
		GetVolumeMounts(mountCerts, mountSSHKey, TempestPropagation, instance),
		GetVolumes(instance, customDataConfigMapName, logsPVCName, mountCerts, mountSSHKey, TempestPropagation),
	)
}
