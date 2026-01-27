// Package util provides common utility functions and constants for test operations
package util //nolint:revive // util is a legitimate package name for utility functions

import (
	"github.com/openstack-k8s-operators/lib-common/modules/storage"
	testv1beta1 "github.com/openstack-k8s-operators/test-operator/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
)

const (
	// ScriptsVolumeDefaultMode is the default permission for script volumes
	ScriptsVolumeDefaultMode int32 = 0755

	// ScriptsVolumeConfidentialMode is the permission for confidential volumes
	ScriptsVolumeConfidentialMode int32 = 0420

	// TLSCertificateMode is the permission for TLS certificates
	TLSCertificateMode int32 = 0444

	// PrivateKeyMode is the permission for private keys
	PrivateKeyMode int32 = 0600

	// PublicKeyMode is the permission for public keys
	PublicKeyMode int32 = 0644

	// PublicInfoMode is the permission for public information
	PublicInfoMode int32 = 0744
)

const (
	volumeNameCACerts    = "ca-certs"
	volumeNameKubeconfig = "kubeconfig"

	secretNameCombinedCABundle = "combined-ca-bundle" // #nosec G101

	subPathCloudsYAML  = "clouds.yaml"
	subPathConfig      = "config"
	subPathSecureYAML  = "secure.yaml"
	subPathTLSCABundle = "tls-ca-bundle.pem"
)

// CreateConfigMapVolume creates a ConfigMap volume with the specified name and mode
func CreateConfigMapVolume(volumeName string, configMapName string, mode int32) corev1.Volume {
	return corev1.Volume{
		Name: volumeName,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				DefaultMode: &mode,
				LocalObjectReference: corev1.LocalObjectReference{
					Name: configMapName,
				},
			},
		},
	}
}

// CreateOpenstackConfigMapVolume creates the openstack-config ConfigMap volume
func CreateOpenstackConfigMapVolume(configMapName string) corev1.Volume {
	mode := ScriptsVolumeConfidentialMode
	return corev1.Volume{
		Name: configMapName,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				DefaultMode: &mode,
				LocalObjectReference: corev1.LocalObjectReference{
					Name: configMapName,
				},
			},
		},
	}
}

// CreateOpenstackConfigSecretVolume creates the openstack-config-secret volume
func CreateOpenstackConfigSecretVolume(secretName string) corev1.Volume {
	mode := TLSCertificateMode
	return corev1.Volume{
		Name: secretName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				DefaultMode: &mode,
				SecretName:  secretName,
			},
		},
	}
}

// CreateLogsPVCVolume creates the test-operator-logs PVC volume
func CreateLogsPVCVolume(logsPVCName string) corev1.Volume {
	return corev1.Volume{
		Name: TestOperatorLogsVolumeName,
		VolumeSource: corev1.VolumeSource{
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
				ClaimName: logsPVCName,
				ReadOnly:  false,
			},
		},
	}
}

// CreateWorkdirVolume creates the ephemeral workdir volume
func CreateWorkdirVolume() corev1.Volume {
	return corev1.Volume{
		Name: TestOperatorEphemeralVolumeNameWorkdir,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}
}

// CreateTmpVolume creates the ephemeral tmp volume
func CreateTmpVolume() corev1.Volume {
	return corev1.Volume{
		Name: TestOperatorEphemeralVolumeNameTmp,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}
}

// AppendCACertsVolume appends the CA certificates volume
func AppendCACertsVolume(volumes []corev1.Volume) []corev1.Volume {
	mode := ScriptsVolumeConfidentialMode
	caCertsVolume := corev1.Volume{
		Name: volumeNameCACerts,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				DefaultMode: &mode,
				SecretName:  secretNameCombinedCABundle,
			},
		},
	}

	return append(volumes, caCertsVolume)
}

// AppendSSHKeyVolume appends an SSH key volume from a secret
func AppendSSHKeyVolume(volumes []corev1.Volume, volumeName, secretName string) []corev1.Volume {
	mode := PrivateKeyMode
	keysVolume := corev1.Volume{
		Name: volumeName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName:  secretName,
				DefaultMode: &mode,
			},
		},
	}

	return append(volumes, keysVolume)
}

// AppendSSHKeyVolumeWithPath appends an SSH key volume from a secret with key path
func AppendSSHKeyVolumeWithPath(volumes []corev1.Volume, volumeName, secretName, keyName, keyPath string) []corev1.Volume {
	mode := PrivateKeyMode
	sshKeyVolume := corev1.Volume{
		Name: volumeName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName:  secretName,
				DefaultMode: &mode,
				Items: []corev1.KeyToPath{
					{
						Key:  keyName,
						Path: keyPath,
					},
				},
			},
		},
	}

	return append(volumes, sshKeyVolume)
}

// AppendKubeconfigVolume appends a kubeconfig volume from a secret
func AppendKubeconfigVolume(volumes []corev1.Volume, secretName string) []corev1.Volume {
	kubeconfigVolume := corev1.Volume{
		Name: volumeNameKubeconfig,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: secretName,
				Items: []corev1.KeyToPath{
					{
						Key:  subPathConfig,
						Path: subPathConfig,
					},
				},
			},
		},
	}

	return append(volumes, kubeconfigVolume)
}

// AppendExtraMountsVolumes appends volumes from ExtraMounts spec
func AppendExtraMountsVolumes(
	volumes []corev1.Volume,
	extraMounts []testv1beta1.ExtraVolMounts,
	svc []storage.PropagationType,
) []corev1.Volume {
	for _, exv := range extraMounts {
		for _, vol := range exv.Propagate(svc) {
			for _, v := range vol.Volumes {
				volumeSource, _ := v.ToCoreVolumeSource()
				convertedVolume := corev1.Volume{
					Name:         v.Name,
					VolumeSource: *volumeSource,
				}
				volumes = append(volumes, convertedVolume)
			}
		}
	}

	return volumes
}

// AppendExtraConfigmapsVolumes appends volumes from ExtraConfigmapsMounts spec
func AppendExtraConfigmapsVolumes(
	volumes []corev1.Volume,
	extraConfigmaps []testv1beta1.ExtraConfigmapsMounts,
	defaultMode int32,
) []corev1.Volume {
	for _, vol := range extraConfigmaps {
		mode := defaultMode
		extraVol := corev1.Volume{
			Name: vol.Name,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					DefaultMode: &mode,
					LocalObjectReference: corev1.LocalObjectReference{
						Name: vol.Name,
					},
				},
			},
		}

		volumes = append(volumes, extraVol)
	}

	return volumes
}

// CreateVolumeMount creates a basic VolumeMount
func CreateVolumeMount(name string, mountPath string, readOnly bool) corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      name,
		MountPath: mountPath,
		ReadOnly:  readOnly,
	}
}

// CreateVolumeMountWithSubPath creates a VolumeMount with a SubPath
func CreateVolumeMountWithSubPath(name string, mountPath string, subPath string, readOnly bool) corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      name,
		MountPath: mountPath,
		SubPath:   subPath,
		ReadOnly:  readOnly,
	}
}

// CreateCACertVolumeMount creates a CA certificate volume mount
func CreateCACertVolumeMount(mountPath string) corev1.VolumeMount {
	return CreateVolumeMountWithSubPath(volumeNameCACerts, mountPath, subPathTLSCABundle, true)
}

// CreateOpenstackConfigVolumeMount creates an openstack config volume mount
func CreateOpenstackConfigVolumeMount(configMapName string, mountPath string) corev1.VolumeMount {
	return CreateVolumeMountWithSubPath(configMapName, mountPath, subPathCloudsYAML, true)
}

// CreateOpenstackConfigSecretVolumeMount creates an openstack config secret volume mount
func CreateOpenstackConfigSecretVolumeMount(secretName string, mountPath string) corev1.VolumeMount {
	return CreateVolumeMountWithSubPath(secretName, mountPath, subPathSecureYAML, false)
}

// CreateTestOperatorCloudsConfigVolumeMount creates a test-operator-clouds-config volume mount
func CreateTestOperatorCloudsConfigVolumeMount(mountPath string) corev1.VolumeMount {
	return CreateVolumeMountWithSubPath(TestOperatorCloudsConfigMapName, mountPath, subPathCloudsYAML, true)
}

// AppendExtraMountsVolumeMounts appends volume mounts from ExtraMounts spec
func AppendExtraMountsVolumeMounts(
	volumeMounts []corev1.VolumeMount,
	extraMounts []testv1beta1.ExtraVolMounts,
	svc []storage.PropagationType,
) []corev1.VolumeMount {
	for _, exv := range extraMounts {
		for _, vol := range exv.Propagate(svc) {
			volumeMounts = append(volumeMounts, vol.Mounts...)
		}
	}

	return volumeMounts
}

// AppendExtraConfigmapsVolumeMounts appends volume mounts from ExtraConfigmapsMounts spec
func AppendExtraConfigmapsVolumeMounts(
	volumeMounts []corev1.VolumeMount,
	extraConfigmaps []testv1beta1.ExtraConfigmapsMounts,
) []corev1.VolumeMount {
	for _, vol := range extraConfigmaps {
		extraMount := corev1.VolumeMount{
			Name:      vol.Name,
			MountPath: vol.MountPath,
			SubPath:   vol.SubPath,
			ReadOnly:  true,
		}

		volumeMounts = append(volumeMounts, extraMount)
	}

	return volumeMounts
}
