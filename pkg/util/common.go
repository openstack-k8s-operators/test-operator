package util

import (
	corev1 "k8s.io/api/core/v1"
)

const (
	// TestOperatorCloudsConfigMapName is name of the ConfigMap which contains
	// modified clouds.yaml obtained from openstack-config ConfigMap. The modified
	// CM is needed by some test frameworks (e.g., HorizonTest and Tobiko)
	TestOperatorCloudsConfigMapName = "test-operator-clouds-config"
)

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
		AllowPrivilegeEscalation: &falseVar,
		Capabilities:             &corev1.Capabilities{},
		SeccompProfile: &corev1.SeccompProfile{
			Type: corev1.SeccompProfileTypeRuntimeDefault,
		},
	}

	if privileged {
		// We need to run pods with AllowPrivilegedEscalation: true to remove
		// nosuid from the pod (in order to be able to run sudo)
		securityContext.AllowPrivilegeEscalation = &trueVar
		securityContext.Capabilities.Add = addCapabilities
	}

	if !privileged {
		// We need to keep default capabilities in order to be able to use sudo
		securityContext.Capabilities.Drop = []corev1.Capability{"ALL"}
	}

	return securityContext
}
