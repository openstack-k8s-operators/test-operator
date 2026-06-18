/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package functional_test

import (
	. "github.com/onsi/gomega" //revive:disable:dot-imports
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	testv1 "github.com/openstack-k8s-operators/test-operator/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
)

const (
	TestOperatorConfig          = "test-operator-config"
	OpenStackConfigMapName      = "openstack-config"
	OpenStackConfigSecretName   = "openstack-config-secret" // #nosec G101
	DefaultStorageClass         = "local-storage"
	DefaultComputeSSHKeySecret  = "dataplane-ansible-ssh-private-key-secret" // #nosec G101
	DefaultWorkloadSSHKeySecret = "dataplane-ansible-ssh-private-key-secret" // #nosec G101
	ExtraConfigMapName          = "extra-config"
	ExtraConfigVolName          = "extra-config-vol"
	ExtraConfigMountPath        = "/etc/extra-config"
	ExtraSecretName             = "extra-secret"      // #nosec G101
	ExtraSecretVolName          = "extra-secret-vol"  // #nosec G101
	ExtraSecretMountPath        = "/etc/extra-secret" // #nosec G101
)

type ExtraMount struct {
	VolName    string
	SourceName string // ConfigMap or Secret name
	MountPath  string
	IsSecret   bool
}

func CreateUnstructured(rawObj map[string]any) *unstructured.Unstructured {
	logger.Info("Creating", "raw", rawObj)
	unstructuredObj := &unstructured.Unstructured{Object: rawObj}
	_, err := controllerutil.CreateOrPatch(
		ctx, k8sClient, unstructuredObj, func() error { return nil })
	Expect(err).ShouldNot(HaveOccurred())
	return unstructuredObj
}

// CreateCommonOpenstackResources creates ConfigMap and Secret needed by all tests
func CreateCommonOpenstackResources(namespace string) (*corev1.ConfigMap, *corev1.Secret) {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      OpenStackConfigMapName,
			Namespace: namespace,
		},
		Data: map[string]string{
			"clouds.yaml": "clouds:\n  default:\n    auth:\n      username: admin",
		},
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      OpenStackConfigSecretName,
			Namespace: namespace,
		},
		StringData: map[string]string{
			"secure.yaml": "clouds:\n  default:\n    auth:\n      password: '12345678'",
		},
	}

	return cm, secret
}

func CreateTestOperatorConfigMap(namespace string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      TestOperatorConfig,
			Namespace: namespace,
		},
		Data: map[string]string{
			"ansibletest-image": "quay.io/podified-antelope-centos9/openstack-ansibletest:current-podified",
			"horizontest-image": "quay.io/podified-antelope-centos9/openstack-horizontest:current-podified",
			"tempest-image":     "quay.io/podified-antelope-centos9/openstack-tempest:current-podified",
			"tobiko-image":      "quay.io/podified-antelope-centos9/openstack-tobiko:current-podified",
		},
	}
}

func CreateExtraConfigMap(namespace string, name string) {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: map[string]string{
			"config.conf": "key=value",
		},
	}
	Expect(k8sClient.Create(ctx, cm)).Should(Succeed())
}

func CreateExtraSecret(namespace string, name string) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		StringData: map[string]string{
			"secret.conf": "key=value",
		},
	}
	Expect(k8sClient.Create(ctx, secret)).Should(Succeed())
}

func BuildExtraMountsSpec(propagation string, mounts ...ExtraMount) []map[string]any {
	volumes := []map[string]any{}
	volumeMounts := []map[string]any{}

	for _, m := range mounts {
		vol := map[string]any{"name": m.VolName}
		if m.IsSecret {
			vol["secret"] = map[string]any{"secretName": m.SourceName}
		} else {
			vol["configMap"] = map[string]any{"name": m.SourceName}
		}
		volumes = append(volumes, vol)

		volumeMounts = append(volumeMounts, map[string]any{
			"name":      m.VolName,
			"mountPath": m.MountPath,
			"readOnly":  true,
		})
	}

	extraVol := map[string]any{
		"volumes": volumes,
		"mounts":  volumeMounts,
	}
	if propagation != "" {
		extraVol["propagation"] = []string{propagation}
	}
	return []map[string]any{
		{"extraVol": []map[string]any{extraVol}},
	}
}

func ExpectPodHasConfigMapVolume(pod *corev1.Pod, volName string, cmName string) {
	foundVolume := false
	for _, vol := range pod.Spec.Volumes {
		if vol.Name == volName {
			foundVolume = true
			Expect(vol.VolumeSource.ConfigMap).NotTo(BeNil())
			Expect(vol.VolumeSource.ConfigMap.Name).To(Equal(cmName))
			break
		}
	}
	Expect(foundVolume).To(BeTrue(), "expected pod to have volume '%s'", volName)
}

func ExpectPodHasSecretVolume(pod *corev1.Pod, volName string, secretName string) {
	foundVolume := false
	for _, vol := range pod.Spec.Volumes {
		if vol.Name == volName {
			foundVolume = true
			Expect(vol.VolumeSource.Secret).NotTo(BeNil())
			Expect(vol.VolumeSource.Secret.SecretName).To(Equal(secretName))
			break
		}
	}
	Expect(foundVolume).To(BeTrue(), "expected pod to have volume '%s'", volName)
}

func ExpectPodHasVolumeMount(pod *corev1.Pod, volName string, mountPath string) {
	container := pod.Spec.Containers[0]
	foundMount := false
	for _, mount := range container.VolumeMounts {
		if mount.Name == volName {
			foundMount = true
			Expect(mount.MountPath).To(Equal(mountPath))
			Expect(mount.ReadOnly).To(BeTrue())
			break
		}
	}
	Expect(foundMount).To(BeTrue(), "expected container to have volumeMount '%s'", volName)
}

func ExpectPodNotHasVolume(pod *corev1.Pod, volName string) {
	for _, vol := range pod.Spec.Volumes {
		Expect(vol.Name).NotTo(Equal(volName),
			"volume should not be propagated with wrong propagation type")
	}
}

func ExpectPodNotHasVolumeMount(pod *corev1.Pod, volName string) {
	container := pod.Spec.Containers[0]
	for _, mount := range container.VolumeMounts {
		Expect(mount.Name).NotTo(Equal(volName),
			"volumeMount should not be propagated with wrong propagation type")
	}
}

func GetDefaultConfigMapExtraMount() ExtraMount {
	return ExtraMount{
		VolName:    ExtraConfigVolName,
		SourceName: ExtraConfigMapName,
		MountPath:  ExtraConfigMountPath,
	}
}

func GetDefaultSecretExtraMount() ExtraMount {
	return ExtraMount{
		VolName:    ExtraSecretVolName,
		SourceName: ExtraSecretName,
		MountPath:  ExtraSecretMountPath,
		IsSecret:   true,
	}
}

func GetTestOperatorPVC(namespace string, instanceName string) *corev1.PersistentVolumeClaim {
	var pvc corev1.PersistentVolumeClaim
	Eventually(func(g Gomega) {
		pvcList := &corev1.PersistentVolumeClaimList{}
		listOpts := []client.ListOption{
			client.InNamespace(namespace),
			client.MatchingLabels{
				"instanceName": instanceName,
				"operator":     "test-operator",
			},
		}
		g.Expect(k8sClient.List(ctx, pvcList, listOpts...)).Should(Succeed())
		g.Expect(pvcList.Items).To(HaveLen(1))
		pvc = pvcList.Items[0]
	}, timeout*2, interval).Should(Succeed())
	return &pvc
}

func GetTestOperatorPod(namespace string, instanceName string) *corev1.Pod {
	var pod corev1.Pod
	Eventually(func(g Gomega) {
		podList := &corev1.PodList{}
		listOpts := []client.ListOption{
			client.InNamespace(namespace),
			client.MatchingLabels{
				"instanceName": instanceName,
				"operator":     "test-operator",
			},
		}
		g.Expect(k8sClient.List(ctx, podList, listOpts...)).Should(Succeed())
		g.Expect(podList.Items).To(HaveLen(1))
		pod = podList.Items[0]
	}, timeout*3, interval).Should(Succeed())
	return &pod
}

// AnsibleTest helpers
func CreateAnsibleTest(name types.NamespacedName, spec map[string]any) client.Object {
	raw := map[string]any{
		"apiVersion": "test.openstack.org/v1beta1",
		"kind":       "AnsibleTest",
		"metadata": map[string]any{
			"name":      name.Name,
			"namespace": name.Namespace,
		},
		"spec": spec,
	}
	return CreateUnstructured(raw)
}

func GetAnsibleTest(name types.NamespacedName) *testv1.AnsibleTest {
	instance := &testv1.AnsibleTest{}
	Eventually(func(g Gomega) {
		g.Expect(k8sClient.Get(ctx, name, instance)).Should(Succeed())
	}, timeout, interval).Should(Succeed())
	return instance
}

func GetDefaultAnsibleTestSpec() map[string]any {
	return map[string]any{
		"storageClass":        DefaultStorageClass,
		"ansibleGitRepo":      "https://github.com/example/test-repo",
		"ansiblePlaybookPath": "tests/playbook.yaml",
	}
}

func AnsibleTestConditionGetter(name types.NamespacedName) condition.Conditions {
	instance := GetAnsibleTest(name)
	return instance.Status.Conditions
}

// HorizonTest helpers
func CreateHorizonTest(name types.NamespacedName, spec map[string]any) client.Object {
	raw := map[string]any{
		"apiVersion": "test.openstack.org/v1beta1",
		"kind":       "HorizonTest",
		"metadata": map[string]any{
			"name":      name.Name,
			"namespace": name.Namespace,
		},
		"spec": spec,
	}
	return CreateUnstructured(raw)
}

func GetHorizonTest(name types.NamespacedName) *testv1.HorizonTest {
	instance := &testv1.HorizonTest{}
	Eventually(func(g Gomega) {
		g.Expect(k8sClient.Get(ctx, name, instance)).Should(Succeed())
	}, timeout, interval).Should(Succeed())
	return instance
}

func GetDefaultHorizonTestSpec() map[string]any {
	return map[string]any{
		"storageClass":  DefaultStorageClass,
		"adminUsername": "admin",
		"adminPassword": "password",
		"dashboardUrl":  "http://horizon.example.com",
		"authUrl":       "http://keystone.example.com:5000/v3",
	}
}

func HorizonTestConditionGetter(name types.NamespacedName) condition.Conditions {
	instance := GetHorizonTest(name)
	return instance.Status.Conditions
}

// Tempest helpers
func CreateTempest(name types.NamespacedName, spec map[string]any) client.Object {
	raw := map[string]any{
		"apiVersion": "test.openstack.org/v1beta1",
		"kind":       "Tempest",
		"metadata": map[string]any{
			"name":      name.Name,
			"namespace": name.Namespace,
		},
		"spec": spec,
	}
	return CreateUnstructured(raw)
}

func GetTempest(name types.NamespacedName) *testv1.Tempest {
	instance := &testv1.Tempest{}
	Eventually(func(g Gomega) {
		g.Expect(k8sClient.Get(ctx, name, instance)).Should(Succeed())
	}, timeout, interval).Should(Succeed())
	return instance
}

func GetDefaultTempestSpec() map[string]any {
	return map[string]any{
		"storageClass": DefaultStorageClass,
		"tempestRun": map[string]any{
			"includeList": "tempest.api.identity.v3",
		},
	}
}

func TempestConditionGetter(name types.NamespacedName) condition.Conditions {
	instance := GetTempest(name)
	return instance.Status.Conditions
}

// Tobiko helpers
func CreateTobiko(name types.NamespacedName, spec map[string]any) client.Object {
	raw := map[string]any{
		"apiVersion": "test.openstack.org/v1beta1",
		"kind":       "Tobiko",
		"metadata": map[string]any{
			"name":      name.Name,
			"namespace": name.Namespace,
		},
		"spec": spec,
	}
	return CreateUnstructured(raw)
}

func GetTobiko(name types.NamespacedName) *testv1.Tobiko {
	instance := &testv1.Tobiko{}
	Eventually(func(g Gomega) {
		g.Expect(k8sClient.Get(ctx, name, instance)).Should(Succeed())
	}, timeout, interval).Should(Succeed())
	return instance
}

func GetDefaultTobikoSpec() map[string]any {
	return map[string]any{
		"storageClass": DefaultStorageClass,
		"testenv":      "sanity",
	}
}

func TobikoConditionGetter(name types.NamespacedName) condition.Conditions {
	instance := GetTobiko(name)
	return instance.Status.Conditions
}
