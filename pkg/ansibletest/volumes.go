package ansibletest

import (
	testv1beta1 "github.com/openstack-k8s-operators/test-operator/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
)

// GetVolumes -
func GetVolumes(
	instance *testv1beta1.AnsibleTest,
	logsPVCName string,
	mountCerts bool,
	workflowOverrideParams map[string]string,
	externalWorkflowCounter int,
) []corev1.Volume {

	var scriptsVolumeConfidentialMode int32 = 0420
	var tlsCertificateMode int32 = 0444
	var privateKeyMode int32 = 0600
	var publicInfoMode int32 = 0744

	//source_type := corev1.HostPathDirectoryOrCreate
	volumes := []corev1.Volume{
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

	keysVolume := corev1.Volume{
		Name: "compute-ssh-secret",
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName:  workflowOverrideParams["ComputesSSHKeySecretName"],
				DefaultMode: &privateKeyMode,
			},
		},
	}

	volumes = append(volumes, keysVolume)

	keysVolume = corev1.Volume{
		Name: "workload-ssh-secret",
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName:  workflowOverrideParams["WorkloadSSHKeySecretName"],
				DefaultMode: &privateKeyMode,
			},
		},
	}

	volumes = append(volumes, keysVolume)

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

	if len(instance.Spec.Workflow) > 0 && instance.Spec.Workflow[externalWorkflowCounter].ExtraConfigmapsMounts != nil {
		for _, vol := range *instance.Spec.Workflow[externalWorkflowCounter].ExtraConfigmapsMounts {
			extraWorkflowVol := corev1.Volume{
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

			volumes = append(volumes, extraWorkflowVol)
		}
	}
	return volumes
}

// GetVolumeMounts -
func GetVolumeMounts(mountCerts bool, instance *testv1beta1.AnsibleTest, externalWorkflowCounter int) []corev1.VolumeMount {
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      "test-operator-logs",
			MountPath: "/var/lib/AnsibleTests/external_files",
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
			MountPath: "/var/lib/ansible/.config/openstack/clouds.yaml",
			SubPath:   "clouds.yaml",
			ReadOnly:  true,
		},
		{
			Name:      "openstack-config-secret",
			MountPath: "/var/lib/ansible/.config/openstack/secure.yaml",
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

		caCertVolumeMount = corev1.VolumeMount{
			Name:      "ca-certs",
			MountPath: "/etc/pki/tls/certs/ca-bundle.trust.crt",
			ReadOnly:  true,
			SubPath:   "tls-ca-bundle.pem",
		}

		volumeMounts = append(volumeMounts, caCertVolumeMount)
	}

	workloadSSHKeyMount := corev1.VolumeMount{
		Name:      "workload-ssh-secret",
		MountPath: "/var/lib/ansible/test_keypair.key",
		SubPath:   "ssh-privatekey",
		ReadOnly:  true,
	}

	volumeMounts = append(volumeMounts, workloadSSHKeyMount)

	computeSSHKeyMount := corev1.VolumeMount{
		Name:      "compute-ssh-secret",
		MountPath: "/var/lib/ansible/.ssh/compute_id",
		SubPath:   "ssh-privatekey",
		ReadOnly:  true,
	}

	volumeMounts = append(volumeMounts, computeSSHKeyMount)

	for _, vol := range instance.Spec.ExtraConfigmapsMounts {

		extraConfigmapsMounts := corev1.VolumeMount{
			Name:      vol.Name,
			MountPath: vol.MountPath,
			SubPath:   vol.SubPath,
			ReadOnly:  true,
		}

		volumeMounts = append(volumeMounts, extraConfigmapsMounts)
	}

	if len(instance.Spec.Workflow) > 0 && instance.Spec.Workflow[externalWorkflowCounter].ExtraConfigmapsMounts != nil {
		for _, vol := range *instance.Spec.Workflow[externalWorkflowCounter].ExtraConfigmapsMounts {

			extraConfigmapsMounts := corev1.VolumeMount{
				Name:      vol.Name,
				MountPath: vol.MountPath,
				SubPath:   vol.SubPath,
				ReadOnly:  true,
			}

			volumeMounts = append(volumeMounts, extraConfigmapsMounts)
		}
	}

	return volumeMounts
}
