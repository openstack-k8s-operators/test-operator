package tempest

import (
	"github.com/openstack-k8s-operators/lib-common/modules/common/env"

	testv1beta1 "github.com/openstack-k8s-operators/test-operator/api/v1beta1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strconv"
)

// Job - prepare job to run Tempest tests
func Job(
	instance *testv1beta1.Tempest,
	labels map[string]string,
) *batchv1.Job {

	envVars := map[string]env.Setter{}
	runAsUser := int64(42480)
	runAsGroup := int64(42480)
	if instance.Spec.TempestRun.Concurrency != nil {
		envVars["TEMPEST_CONCURRENCY"] = env.SetValue(strconv.FormatInt(*instance.Spec.TempestRun.Concurrency, 10))
	} else {
		envVars["TEMPEST_CONCURRENCY"] = env.SetValue("0")
	}

	// NOTE: validate also having pv ?
	// When having PV the path also should work when the home dir is different
	if instance.Spec.PersistentVolumePath != "" {
		envVars["TEMPEST_OUTPUTDIR"] = env.SetValue("/var/lib/tempest/output" + instance.Spec.PersistentVolumePath)
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
					SecurityContext: &corev1.PodSecurityContext{
						RunAsUser:  &runAsUser,
						RunAsGroup: &runAsGroup,
						FSGroup:    &runAsGroup,
					},
					Containers: []corev1.Container{
						{
							Name:  instance.Name + "-tests-runner",
							Image: instance.Spec.ContainerImage,
							Command: []string{
								"/usr/local/bin/container-scripts/invoke_tempest",
							},
							Args:         []string{},
							Env:          env.MergeEnvs([]corev1.EnvVar{}, envVars),
							VolumeMounts: GetVolumeMounts(instance),
						},
					},
					Volumes: GetVolumes(instance),
				},
			},
		},
	}

	return job
}
