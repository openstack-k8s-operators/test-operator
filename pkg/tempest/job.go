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
) *batchv1.Job {

	envVars := map[string]env.Setter{}

	args := []string{
		"/var/lib/tempest/run_tempest.sh",
	}
	if instance.Spec.TempestRegex != "" {
		args = append(args, "--regex")
		args = append(args, instance.Spec.TempestRegex)
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: instance.Namespace,
			Labels:    labels,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy:      "OnFailure",
					ServiceAccountName: instance.RbacResourceName(),
					Containers: []corev1.Container{
						{
							Name:  instance.Name + "-tests-runner",
							Image: instance.Spec.ContainerImage,
							Command: []string{
								"/var/lib/tempest/run_tempest.sh",
							},
							Args: []string{},
							SecurityContext: &corev1.SecurityContext{
								Capabilities: &corev1.Capabilities{
									Add:  []corev1.Capability{},
									Drop: []corev1.Capability{"ALL"},
								},
								SeccompProfile: &corev1.SeccompProfile{
									Type: "RuntimeDefault",
								},
							},
							Env:          env.MergeEnvs([]corev1.EnvVar{}, envVars),
							VolumeMounts: GetVolumeMounts(),
						},
					},
					Volumes: GetVolumes(instance.Name),
				},
			},
		},
	}

	return job
}
