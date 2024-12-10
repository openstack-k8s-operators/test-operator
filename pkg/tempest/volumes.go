package tempest

import (
	testv1beta1 "github.com/openstack-k8s-operators/test-operator/api/v1beta1"
	util "github.com/openstack-k8s-operators/test-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
)

// GetVolumes -
func GetVolumes(
	instance *testv1beta1.Tempest,
	customDataConfigMapName string,
	logsPVCName string,
	mountCerts bool,
	mountSSHKey bool,
) []corev1.Volume {

	var scriptsVolumeDefaultMode int32 = 0755
	var scriptsVolumeConfidentialMode int32 = 0420
	var tlsCertificateMode int32 = 0444
	var privateKeyMode int32 = 0600

	//source_type := corev1.HostPathDirectoryOrCreate
	volumes := []corev1.Volume{
		{
			Name: "config-data",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					DefaultMode: &scriptsVolumeDefaultMode,
					LocalObjectReference: corev1.LocalObjectReference{
						Name: customDataConfigMapName,
					},
				},
			},
		},
		{
			Name: "openstack-config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					DefaultMode: &scriptsVolumeConfidentialMode,
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "openstack-config",
					},
				},
			},
		},
		{
			Name: "openstack-config-secret",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					DefaultMode: &tlsCertificateMode,
					SecretName:  "openstack-config-secret",
				},
			},
		},
		{
			Name: "test-operator-logs",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: logsPVCName,
					ReadOnly:  false,
				},
			},
		},
		{
			Name: util.TestOperatorEphemeralVolumeNameWorkdir,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		{
			Name: util.TestOperatorEphemeralVolumeNameTmp,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
	}

	if mountCerts {
		caCertsVolume := corev1.Volume{
			Name: "ca-certs",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					DefaultMode: &scriptsVolumeConfidentialMode,
					SecretName:  "combined-ca-bundle",
				},
			},
		}

		volumes = append(volumes, caCertsVolume)
	}

	if mountSSHKey {
		sshKeyVolume := corev1.Volume{
			Name: "ssh-key",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  instance.Spec.SSHKeySecretName,
					DefaultMode: &privateKeyMode,
					Items: []corev1.KeyToPath{
						{
							Key:  "ssh-privatekey",
							Path: "ssh_key",
						},
					},
				},
			},
		}

		volumes = append(volumes, sshKeyVolume)
	}

	for _, vol := range instance.Spec.ExtraConfigmapsMounts {
		extraVol := corev1.Volume{
			Name: vol.Name,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					DefaultMode: &scriptsVolumeDefaultMode,
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

// GetVolumeMounts -
func GetVolumeMounts(mountCerts bool, mountSSHKey bool, instance *testv1beta1.Tempest) []corev1.VolumeMount {
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      util.TestOperatorEphemeralVolumeNameWorkdir,
			MountPath: "/var/lib/tempest",
			ReadOnly:  false,
		},
		{
			Name:      util.TestOperatorEphemeralVolumeNameTmp,
			MountPath: "/tmp",
			ReadOnly:  false,
		},
		{
			Name:      "config-data",
			MountPath: "/etc/test_operator",
			ReadOnly:  false,
		},
		{
			Name:      "test-operator-logs",
			MountPath: "/var/lib/tempest/external_files",
			ReadOnly:  false,
		},
		{
			Name:      "openstack-config",
			MountPath: "/etc/openstack/clouds.yaml",
			SubPath:   "clouds.yaml",
			ReadOnly:  true,
		},
		{
			Name:      "openstack-config",
			MountPath: "/var/lib/tempest/.config/openstack/clouds.yaml",
			SubPath:   "clouds.yaml",
			ReadOnly:  true,
		},
		{
			Name:      "openstack-config-secret",
			MountPath: "/etc/openstack/secure.yaml",
			ReadOnly:  false,
			SubPath:   "secure.yaml",
		},
	}

	if mountCerts {
		caCertVolumeMount := corev1.VolumeMount{
			Name:      "ca-certs",
			MountPath: "/etc/pki/ca-trust/extracted/pem/tls-ca-bundle.pem",
			ReadOnly:  true,
			SubPath:   "tls-ca-bundle.pem",
		}

		volumeMounts = append(volumeMounts, caCertVolumeMount)
	}

	if mountSSHKey {
		sshKeyMount := corev1.VolumeMount{
			Name:      "ssh-key",
			MountPath: "/var/lib/tempest/id_ecdsa",
			SubPath:   "ssh_key",
		}

		volumeMounts = append(volumeMounts, sshKeyMount)
	}

	for _, vol := range instance.Spec.ExtraConfigmapsMounts {

		extraMounts := corev1.VolumeMount{
			Name:      vol.Name,
			MountPath: vol.MountPath,
			SubPath:   vol.SubPath,
			ReadOnly:  true,
		}

		volumeMounts = append(volumeMounts, extraMounts)
	}

	return volumeMounts
}
