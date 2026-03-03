package tobiko

import (
	"strconv"

	"github.com/openstack-k8s-operators/lib-common/modules/storage"
	testv1beta1 "github.com/openstack-k8s-operators/test-operator/api/v1beta1"
	"github.com/openstack-k8s-operators/test-operator/internal/util"
	corev1 "k8s.io/api/core/v1"
)

// Volume names for ConfigMap mounts
const (
	tobikoConfig     = "tobiko-config"
	tobikoPrivateKey = "tobiko-private-key"
	tobikoPublicKey  = "tobiko-public-key"
)

// ConfigMap name infixes used to construct workflow-step-specific ConfigMap names
const (
	ConfigMapInfixConfig     = "-" + tobikoConfig + "-"
	ConfigMapInfixPrivateKey = "-" + tobikoPrivateKey + "-"
	ConfigMapInfixPublicKey  = "-" + tobikoPublicKey + "-"
)

// ConfigMap data key names for file contents
const (
	ConfigFileName     = "tobiko.conf"
	PrivateKeyFileName = "id_ecdsa"
	PublicKeyFileName  = "id_ecdsa.pub"
)

// GetConfigMapName returns the name of the custom data ConfigMap for the given workflow step
func GetConfigMapName(instance *testv1beta1.Tobiko, infix string, workflowStepIndex int) string {
	return instance.Name + infix + strconv.Itoa(workflowStepIndex)
}

// GetVolumes - returns a list of volumes for the test pod
func GetVolumes(
	instance *testv1beta1.Tobiko,
	logsPVCName string,
	mountCerts bool,
	mountKeys bool,
	mountKubeconfig bool,
	workflowStepIndex int,
	svc []storage.PropagationType,
) []corev1.Volume {

	volumes := []corev1.Volume{
		util.CreateConfigMapVolume(tobikoConfig, GetConfigMapName(instance, ConfigMapInfixConfig, workflowStepIndex), util.ScriptsVolumeDefaultMode),
		util.CreateOpenstackConfigMapVolume(util.TestOperatorCloudsConfigMapName),
		util.CreateOpenstackConfigSecretVolume(instance.Spec.OpenStackConfigSecret),
		util.CreateLogsPVCVolume(logsPVCName),
		util.CreateWorkdirVolume(),
		util.CreateTmpVolume(),
	}

	if mountCerts {
		volumes = util.AppendCACertsVolume(volumes)
	}

	if mountKeys {
		volumes = append(volumes,
			util.CreateConfigMapVolume(tobikoPrivateKey, GetConfigMapName(instance, ConfigMapInfixPrivateKey, workflowStepIndex), util.PrivateKeyMode),
			util.CreateConfigMapVolume(tobikoPublicKey, GetConfigMapName(instance, ConfigMapInfixPublicKey, workflowStepIndex), util.PublicKeyMode),
		)
	}

	if mountKubeconfig {
		volumes = util.AppendKubeconfigVolume(volumes, instance.Spec.KubeconfigSecretName)
	}

	volumes = util.AppendExtraMountsVolumes(volumes, instance.Spec.ExtraMounts, svc)
	volumes = util.AppendExtraConfigmapsVolumes(volumes, instance.Spec.ExtraConfigmapsMounts, util.PublicInfoMode)

	return volumes
}

// GetVolumeMounts - returns a list of volume mounts for the test container
func GetVolumeMounts(
	instance *testv1beta1.Tobiko,
	mountCerts bool,
	mountKeys bool,
	mountKubeconfig bool,
	svc []storage.PropagationType,
) []corev1.VolumeMount {
	volumeMounts := []corev1.VolumeMount{
		util.CreateVolumeMount(util.TestOperatorEphemeralVolumeNameWorkdir, "/var/lib/tobiko", false),
		util.CreateVolumeMount(util.TestOperatorEphemeralVolumeNameTmp, "/tmp", false),
		util.CreateVolumeMount(util.TestOperatorLogsVolumeName, "/var/lib/tobiko/external_files", false),
		util.CreateTestOperatorCloudsConfigVolumeMount("/var/lib/tobiko/.config/openstack/clouds.yaml"),
		util.CreateTestOperatorCloudsConfigVolumeMount("/etc/openstack/clouds.yaml"),
		util.CreateOpenstackConfigSecretVolumeMount(instance.Spec.OpenStackConfigSecret, "/etc/openstack/secure.yaml"),
		util.CreateVolumeMountWithSubPath(tobikoConfig, "/etc/tobiko/tobiko.conf", ConfigFileName, false),
	}

	if mountCerts {
		volumeMounts = append(volumeMounts,
			util.CreateCACertVolumeMount("/etc/pki/ca-trust/extracted/pem/tls-ca-bundle.pem"),
			util.CreateCACertVolumeMount("/etc/pki/tls/certs/ca-bundle.trust.crt"),
		)
	}

	if mountKeys {
		volumeMounts = append(volumeMounts,
			util.CreateVolumeMountWithSubPath(tobikoPrivateKey, "/etc/test_operator/id_ecdsa", PrivateKeyFileName, true),
			util.CreateVolumeMountWithSubPath(tobikoPublicKey, "/etc/test_operator/id_ecdsa.pub", PublicKeyFileName, true),
		)
	}

	if mountKubeconfig {
		volumeMounts = append(volumeMounts,
			util.CreateVolumeMountWithSubPath("kubeconfig", "/var/lib/tobiko/.kube/config", "config", true),
		)
	}

	volumeMounts = util.AppendExtraMountsVolumeMounts(volumeMounts, instance.Spec.ExtraMounts, svc)
	volumeMounts = util.AppendExtraConfigmapsVolumeMounts(volumeMounts, instance.Spec.ExtraConfigmapsMounts)

	return volumeMounts
}
