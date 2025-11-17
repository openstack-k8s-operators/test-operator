// Package util provides common utility functions and constants for test operations
package util //nolint:revive // util is a legitimate package name for utility functions

import (
	"github.com/openstack-k8s-operators/lib-common/modules/common/env"
	"github.com/openstack-k8s-operators/lib-common/modules/storage"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// TestOperatorCloudsConfigMapName is the name of the ConfigMap which contains
	// modified clouds.yaml obtained from openstack-config ConfigMap. The modified
	// CM is needed by some test frameworks (e.g., HorizonTest and Tobiko)
	TestOperatorCloudsConfigMapName = "test-operator-clouds-config"

	// TestOperatorEphemeralVolumeNameWorkdir is the name of the ephemeral workdir volume
	TestOperatorEphemeralVolumeNameWorkdir = "test-operator-ephemeral-workdir"

	// TestOperatorEphemeralVolumeNameTmp is the name of the ephemeral temporary volume
	TestOperatorEphemeralVolumeNameTmp = "test-operator-ephemeral-temporary"

	// ExtraVolTypeUndefined can be used to label an extraMount which is
	// not associated to anything in particular
	ExtraVolTypeUndefined storage.ExtraVolType = "Undefined"
)

// GetSecurityContext returns a security context with the specified configuration
func GetSecurityContext(
	runAsUser int64,
	addCapabilities []corev1.Capability,
	privileged bool,
) corev1.SecurityContext {
	falseVar := false
	trueVar := true

	securityContext := corev1.SecurityContext{
		RunAsUser:                &runAsUser,
		RunAsGroup:               &runAsUser,
		ReadOnlyRootFilesystem:   &trueVar,
		RunAsNonRoot:             &trueVar,
		AllowPrivilegeEscalation: &falseVar,
		Capabilities:             &corev1.Capabilities{},
		SeccompProfile: &corev1.SeccompProfile{
			Type: corev1.SeccompProfileTypeRuntimeDefault,
		},
	}

	if privileged {
		// Sometimes we require the test pods run sudo to be able to install
		// additional packages or run commands with elevated privileges (e.g.,
		// tcpdump in case of Tobiko)
		securityContext.RunAsNonRoot = &falseVar

		// We need to run pods with AllowPrivilegedEscalation: true to remove
		// nosuid from the pod (in order to be able to run sudo)
		securityContext.AllowPrivilegeEscalation = &trueVar

		// We need to run pods with ReadOnlyRootFileSystem: false when installing
		// additional tests using extraRPMs parameter in Tempest CR
		securityContext.ReadOnlyRootFilesystem = &falseVar
		securityContext.Capabilities.Add = addCapabilities
	}

	if !privileged {
		// We need to keep default capabilities in order to be able to use sudo
		securityContext.Capabilities.Drop = []corev1.Capability{"ALL"}
	}

	return securityContext
}

// BuildTestPod creates a pod with common structure used by all test frameworks
func BuildTestPod(
	annotations map[string]string,
	capabilities []corev1.Capability,
	containerImage string,
	containerName string,
	envFromSource []corev1.EnvFromSource,
	envVars map[string]env.Setter,
	labels map[string]string,
	namespace string,
	nodeSelector map[string]string,
	podName string,
	privileged bool,
	resources corev1.ResourceRequirements,
	runAsGroup int64,
	runAsUser int64,
	seLinuxLevel string,
	tolerations []corev1.Toleration,
	volumeMounts []corev1.VolumeMount,
	volumes []corev1.Volume,
) *corev1.Pod {
	securityContext := GetSecurityContext(runAsUser, capabilities, privileged)

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        podName,
			Namespace:   namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: corev1.PodSpec{
			AutomountServiceAccountToken: &privileged,
			RestartPolicy:                corev1.RestartPolicyNever,
			Tolerations:                  tolerations,
			NodeSelector:                 nodeSelector,
			SecurityContext: &corev1.PodSecurityContext{
				RunAsUser:  &runAsUser,
				RunAsGroup: &runAsGroup,
				FSGroup:    &runAsGroup,
			},
			Containers: []corev1.Container{
				{
					Name:            containerName,
					Image:           containerImage,
					Args:            []string{},
					Env:             env.MergeEnvs([]corev1.EnvVar{}, envVars),
					VolumeMounts:    volumeMounts,
					SecurityContext: &securityContext,
					Resources:       resources,
					EnvFrom:         envFromSource,
				},
			},
			Volumes: volumes,
		},
	}

	if len(seLinuxLevel) > 0 {
		pod.Spec.SecurityContext.SELinuxOptions = &corev1.SELinuxOptions{
			Level: seLinuxLevel,
		}
	}

	return pod
}
