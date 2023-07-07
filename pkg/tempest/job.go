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
	envVars["KOLLA_CONFIG_FILE"] = env.SetValue("/var/lib/config-data/tempest-config.json")
	envVars["KOLLA_CONFIG_STRATEGY"] = env.SetValue("COPY_ALWAYS")
	runAsUser := int64(0)

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
			BackoffLimit: instance.Spec.BackoffLimit,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy:      corev1.RestartPolicyNever,
					ServiceAccountName: instance.RbacResourceName(),
					Containers: []corev1.Container{
						{
							Name:  instance.Name + "-tests-runner",
							Image: instance.Spec.ContainerImage,
							Command: []string{
								"/usr/local/bin/kolla_start",
							},
							Args: []string{},
							SecurityContext: &corev1.SecurityContext{
								RunAsUser: &runAsUser,
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
