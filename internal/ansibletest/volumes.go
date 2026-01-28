package ansibletest

import (
	"github.com/openstack-k8s-operators/lib-common/modules/storage"
	testv1beta1 "github.com/openstack-k8s-operators/test-operator/api/v1beta1"
	util "github.com/openstack-k8s-operators/test-operator/internal/util"
	corev1 "k8s.io/api/core/v1"
)

const (
	computeName  = "compute-ssh-secret"
	workloadName = "workload-ssh-secret"
)

// GetVolumes - returns a list of volumes for the test pod
func GetVolumes(
	instance *testv1beta1.AnsibleTest,
	logsPVCName string,
	mountCerts bool,
	svc []storage.PropagationType,
	externalWorkflowCounter int,
) []corev1.Volume {

	volumes := []corev1.Volume{
		util.CreateOpenstackConfigMapVolume(instance.Spec.OpenStackConfigMap),
		util.CreateOpenstackConfigSecretVolume(instance.Spec.OpenStackConfigSecret),
		util.CreateLogsPVCVolume(logsPVCName),
		util.CreateWorkdirVolume(),
		util.CreateTmpVolume(),
	}

	if mountCerts {
		volumes = util.AppendCACertsVolume(volumes)
	}

	volumes = util.AppendSSHKeyVolume(volumes, computeName, instance.Spec.ComputeSSHKeySecretName)

	if instance.Spec.WorkloadSSHKeySecretName != "" {
		volumes = util.AppendSSHKeyVolume(volumes, workloadName, instance.Spec.WorkloadSSHKeySecretName)
	}

	volumes = util.AppendExtraMountsVolumes(volumes, instance.Spec.ExtraMounts, svc)
	volumes = util.AppendExtraConfigmapsVolumes(volumes, instance.Spec.ExtraConfigmapsMounts, util.PublicInfoMode)

	if len(instance.Spec.Workflow) > 0 {
		cmMounts := instance.Spec.Workflow[externalWorkflowCounter].ExtraConfigmapsMounts
		if cmMounts != nil {
			volumes = util.AppendExtraConfigmapsVolumes(volumes, *cmMounts, util.PublicInfoMode)
		}
	}

	return volumes
}

// GetVolumeMounts - returns a list of volume mounts for the test container
func GetVolumeMounts(
	instance *testv1beta1.AnsibleTest,
	mountCerts bool,
	svc []storage.PropagationType,
	externalWorkflowCounter int,
) []corev1.VolumeMount {
	volumeMounts := []corev1.VolumeMount{
		util.CreateVolumeMount(util.TestOperatorEphemeralVolumeNameWorkdir, "/var/lib/ansible", false),
		util.CreateVolumeMount(util.TestOperatorEphemeralVolumeNameTmp, "/tmp", false),
		util.CreateVolumeMount(util.TestOperatorLogsVolumeName, "/var/lib/AnsibleTests/external_files", false),
		util.CreateOpenstackConfigVolumeMount(instance.Spec.OpenStackConfigMap, "/etc/openstack/clouds.yaml"),
		util.CreateOpenstackConfigVolumeMount(instance.Spec.OpenStackConfigMap, "/var/lib/ansible/.config/openstack/clouds.yaml"),
		util.CreateOpenstackConfigSecretVolumeMount(instance.Spec.OpenStackConfigSecret, "/var/lib/ansible/.config/openstack/secure.yaml"),
	}

	if mountCerts {
		volumeMounts = append(volumeMounts,
			util.CreateCACertVolumeMount("/etc/pki/ca-trust/extracted/pem/tls-ca-bundle.pem"),
			util.CreateCACertVolumeMount("/etc/pki/tls/certs/ca-bundle.trust.crt"),
		)
	}

	volumeMounts = append(volumeMounts,
		util.CreateVolumeMountWithSubPath(computeName, "/var/lib/ansible/.ssh/compute_id", "ssh-privatekey", true),
	)

	if instance.Spec.WorkloadSSHKeySecretName != "" {
		volumeMounts = append(volumeMounts,
			util.CreateVolumeMountWithSubPath(workloadName, "/var/lib/ansible/test_keypair.key", "ssh-privatekey", true),
		)
	}

	volumeMounts = util.AppendExtraMountsVolumeMounts(volumeMounts, instance.Spec.ExtraMounts, svc)
	volumeMounts = util.AppendExtraConfigmapsVolumeMounts(volumeMounts, instance.Spec.ExtraConfigmapsMounts)

	if len(instance.Spec.Workflow) > 0 {
		cmMounts := instance.Spec.Workflow[externalWorkflowCounter].ExtraConfigmapsMounts
		if cmMounts != nil {
			volumeMounts = util.AppendExtraConfigmapsVolumeMounts(volumeMounts, *cmMounts)
		}
	}

	return volumeMounts
}
