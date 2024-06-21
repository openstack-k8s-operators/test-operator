package tempest

import (
	"github.com/openstack-k8s-operators/lib-common/modules/common/env"

	testv1beta1 "github.com/openstack-k8s-operators/test-operator/api/v1beta1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Job - prepare job to run Tempest tests
func Job(
	instance *testv1beta1.Tempest,
	labels map[string]string,
	annotations map[string]string,
	jobName string,
	envVarsConfigMapName string,
	customDataConfigMapName string,
	logsPVCName string,
	containerImage string,
	mountCerts bool,
	mountSSHKey bool,
) *batchv1.Job {

	envVars := map[string]env.Setter{}
	runAsUser := int64(42480)
	runAsGroup := int64(42480)

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: instance.Namespace,
			Labels:    labels,
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: instance.Spec.BackoffLimit,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: annotations,
					Labels:      labels,
				},
				Spec: corev1.PodSpec{
					RestartPolicy:      corev1.RestartPolicyNever,
					ServiceAccountName: instance.RbacResourceName(),
					SecurityContext: &corev1.PodSecurityContext{
						RunAsUser:  &runAsUser,
						RunAsGroup: &runAsGroup,
						FSGroup:    &runAsGroup,
						SELinuxOptions: &corev1.SELinuxOptions{
							Level: instance.Spec.SELinuxLevel,
						},
					},
					Tolerations:  instance.Spec.Tolerations,
					NodeSelector: instance.Spec.NodeSelector,
					Containers: []corev1.Container{
						{
							Name:         instance.Name + "-tests-runner",
							Image:        containerImage,
							Args:         []string{},
							Env:          env.MergeEnvs([]corev1.EnvVar{}, envVars),
							VolumeMounts: GetVolumeMounts(mountCerts, mountSSHKey),
							SecurityContext: &corev1.SecurityContext{
								Capabilities: &corev1.Capabilities{
									Add: []corev1.Capability{"CAP_AUDIT_WRITE"},
								},
							},
							EnvFrom: []corev1.EnvFromSource{
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
							},
						},
					},
					Volumes: GetVolumes(
						instance,
						customDataConfigMapName,
						logsPVCName,
						mountCerts,
						mountSSHKey,
					),
				},
			},
		},
	}

	return job
}
