package tobiko

import (
	testv1beta1 "github.com/openstack-k8s-operators/test-operator/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
)

// GetVolumes -
func GetVolumes(
	instance *testv1beta1.Tobiko,
	logsPVCName string,
	mountCerts bool,
	mountKeys bool,
	mountKubeconfig bool,
) []corev1.Volume {

	var scriptsVolumeDefaultMode int32 = 0755
	var scriptsVolumeConfidentialMode int32 = 0420
	var privateKeyMode int32 = 0600
	var publicKeyMode int32 = 0644
	var tlsCertificateMode int32 = 0444
	var publicInfoMode int32 = 0744

	volumes := []corev1.Volume{
		{
			Name: "tobiko-config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					DefaultMode: &scriptsVolumeDefaultMode,
					LocalObjectReference: corev1.LocalObjectReference{
						Name: instance.Name + "tobiko-config",
					},
				},
			},
		},
		{
			Name: "tobiko-clouds-config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					DefaultMode: &scriptsVolumeConfidentialMode,
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "tobiko-clouds-config",
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

	if mountKeys {
		keysVolume := corev1.Volume{
			Name: "tobiko-private-key",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					DefaultMode: &privateKeyMode,
					LocalObjectReference: corev1.LocalObjectReference{
						Name: instance.Name + "tobiko-private-key",
					},
				},
			},
		}

		volumes = append(volumes, keysVolume)

		keysVolume = corev1.Volume{
			Name: "tobiko-public-key",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					DefaultMode: &publicKeyMode,
					LocalObjectReference: corev1.LocalObjectReference{
						Name: instance.Name + "tobiko-public-key",
					},
				},
			},
		}

		volumes = append(volumes, keysVolume)
	}

	if mountKubeconfig {
		kubeconfigVolume := corev1.Volume{
			Name: "kubeconfig",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: instance.Spec.KubeconfigSecretName,
					Items: []corev1.KeyToPath{
						{
							Key:  "config",
							Path: "config",
						},
					},
				},
			},
		}

		volumes = append(volumes, kubeconfigVolume)
	}

	for _, vol := range instance.Spec.ExtraConfigmapsMounts {
		extraVol := corev1.Volume{
			Name: vol.Name,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					DefaultMode: &publicInfoMode,
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
func GetVolumeMounts(mountCerts bool, mountKeys bool, mountKubeconfig bool, instance *testv1beta1.Tobiko) []corev1.VolumeMount {
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      "test-operator-logs",
			MountPath: "/var/lib/tobiko/external_files",
			ReadOnly:  false,
		},
		{
			Name:      "tobiko-clouds-config",
			MountPath: "/var/lib/tobiko/.config/openstack/clouds.yaml",
			SubPath:   "clouds.yaml",
			ReadOnly:  true,
		},
		{
			Name:      "tobiko-clouds-config",
			MountPath: "/etc/openstack/clouds.yaml",
			SubPath:   "clouds.yaml",
			ReadOnly:  true,
		},
		{
			Name:      "openstack-config-secret",
			MountPath: "/etc/openstack/secure.yaml",
			ReadOnly:  false,
			SubPath:   "secure.yaml",
		},
		{
			Name:      "tobiko-config",
			MountPath: "/etc/tobiko/tobiko.conf",
			SubPath:   "tobiko.conf",
			ReadOnly:  false,
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

		caCertVolumeMount = corev1.VolumeMount{
			Name:      "ca-certs",
			MountPath: "/etc/pki/tls/certs/ca-bundle.trust.crt",
			ReadOnly:  true,
			SubPath:   "tls-ca-bundle.pem",
		}

		volumeMounts = append(volumeMounts, caCertVolumeMount)
	}

	if mountKeys {
		keysMount := corev1.VolumeMount{
			Name:      "tobiko-private-key",
			MountPath: "/etc/test_operator/id_ecdsa",
			SubPath:   "id_ecdsa",
			ReadOnly:  true,
		}

		volumeMounts = append(volumeMounts, keysMount)

		keysMount = corev1.VolumeMount{
			Name:      "tobiko-public-key",
			MountPath: "/etc/test_operator/id_ecdsa.pub",
			SubPath:   "id_ecdsa.pub",
			ReadOnly:  true,
		}

		volumeMounts = append(volumeMounts, keysMount)
	}

	if mountKubeconfig {
		kubeconfigMount := corev1.VolumeMount{
			Name:      "kubeconfig",
			MountPath: "/var/lib/tobiko/.kube/config",
			SubPath:   "config",
			ReadOnly:  true,
		}

		volumeMounts = append(volumeMounts, kubeconfigMount)
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
