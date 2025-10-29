package ansibletest

import (
	"github.com/openstack-k8s-operators/lib-common/modules/common/env"

	testv1beta1 "github.com/openstack-k8s-operators/test-operator/api/v1beta1"
	util "github.com/openstack-k8s-operators/test-operator/internal/util"
	corev1 "k8s.io/api/core/v1"
)

// Pod - prepare pod to run AnsibleTests tests
func Pod(
	instance *testv1beta1.AnsibleTest,
	labels map[string]string,
	podName string,
	logsPVCName string,
	mountCerts bool,
	envVars map[string]env.Setter,
	externalWorkflowCounter int,
	containerImage string,
) *corev1.Pod {
	return util.BuildTestPod(
		nil, // No annotations
		PodCapabilities,
		containerImage,
		instance.Name,
		[]corev1.EnvFromSource{}, // No EnvFromSource
		envVars,
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
		GetVolumeMounts(instance, mountCerts, AnsibleTestPropagation, externalWorkflowCounter),
		GetVolumes(instance, logsPVCName, mountCerts, AnsibleTestPropagation, externalWorkflowCounter),
	)
}
