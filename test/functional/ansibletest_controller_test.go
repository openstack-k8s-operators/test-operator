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
	. "github.com/onsi/ginkgo/v2" //revive:disable:dot-imports
	. "github.com/onsi/gomega"    //revive:disable:dot-imports

	"github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	//revive:disable-next-line:dot-imports
	. "github.com/openstack-k8s-operators/lib-common/modules/common/test/helpers"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("AnsibleTest controller", func() {
	var ansibleTestName types.NamespacedName

	BeforeEach(func() {
		ansibleTestName = types.NamespacedName{
			Name:      "ansibletest",
			Namespace: namespace,
		}
	})

	DescribeTable("Missing Openstack resources should set InputReady to false",
		func(createResource func()) {
			createResource()
			DeferCleanup(th.DeleteInstance, CreateAnsibleTest(ansibleTestName, GetDefaultAnsibleTestSpec()))

			th.ExpectCondition(
				ansibleTestName,
				ConditionGetterFunc(AnsibleTestConditionGetter),
				condition.InputReadyCondition,
				corev1.ConditionFalse,
			)
		},
		Entry("when config map is missing", func() {
			_, secret := CreateCommonOpenstackResources(namespace)
			Expect(k8sClient.Create(ctx, secret)).Should(Succeed())
		}),
		Entry("when secret is missing", func() {
			cm, _ := CreateCommonOpenstackResources(namespace)
			Expect(k8sClient.Create(ctx, cm)).Should(Succeed())
		}),
	)

	When("An AnsibleTest instance is created", func() {
		BeforeEach(func() {
			openstackConfigMap, openstackSecret := CreateCommonOpenstackResources(namespace)
			Expect(k8sClient.Create(ctx, openstackConfigMap)).Should(Succeed())
			Expect(k8sClient.Create(ctx, openstackSecret)).Should(Succeed())
			DeferCleanup(th.DeleteInstance, CreateAnsibleTest(ansibleTestName, GetDefaultAnsibleTestSpec()))
		})

		It("initializes the status fields", func() {
			Eventually(func(g Gomega) {
				ansibleTest := GetAnsibleTest(ansibleTestName)
				g.Expect(ansibleTest.Status.Conditions).To(HaveLen(3))
				g.Expect(ansibleTest.Status.Hash).To(BeEmpty())
			}, timeout*2, interval).Should(Succeed())
		})

		It("should have the Spec fields initialized", func() {
			ansibleTest := GetAnsibleTest(ansibleTestName)
			Expect(ansibleTest.Spec.StorageClass).Should(Equal(DefaultStorageClass))
			Expect(ansibleTest.Spec.AnsibleGitRepo).ShouldNot(BeEmpty())
			Expect(ansibleTest.Spec.AnsiblePlaybookPath).ShouldNot(BeEmpty())
		})
	})

	When("All dependencies are ready", func() {
		BeforeEach(func() {
			openstackConfigMap, openstackSecret := CreateCommonOpenstackResources(namespace)
			Expect(k8sClient.Create(ctx, openstackConfigMap)).Should(Succeed())
			Expect(k8sClient.Create(ctx, openstackSecret)).Should(Succeed())

			testOperatorConfigMap := CreateTestOperatorConfigMap(namespace)
			Expect(k8sClient.Create(ctx, testOperatorConfigMap)).Should(Succeed())

			DeferCleanup(th.DeleteInstance, CreateAnsibleTest(ansibleTestName, GetDefaultAnsibleTestSpec()))
		})

		It("should have InputReady condition true", func() {
			th.ExpectCondition(
				ansibleTestName,
				ConditionGetterFunc(AnsibleTestConditionGetter),
				condition.InputReadyCondition,
				corev1.ConditionTrue,
			)
		})

		It("should create a PVC for logs", func() {
			pvc := GetTestOperatorPVC(namespace, ansibleTestName.Name)
			Expect(pvc.Name).ToNot(BeEmpty())
			Expect(*pvc.Spec.StorageClassName).To(Equal(DefaultStorageClass))
			Expect(pvc.Spec.AccessModes).To(ContainElement(corev1.ReadWriteOnce))
		})

		It("should create a pod", func() {
			pod := GetTestOperatorPod(namespace, ansibleTestName.Name)
			Expect(pod.Name).ToNot(BeEmpty())
		})
	})

	When("AnsibleTest is created with extraMounts", func() {
		BeforeEach(func() {
			openstackConfigMap, openstackSecret := CreateCommonOpenstackResources(namespace)
			Expect(k8sClient.Create(ctx, openstackConfigMap)).Should(Succeed())
			Expect(k8sClient.Create(ctx, openstackSecret)).Should(Succeed())

			testOperatorConfigMap := CreateTestOperatorConfigMap(namespace)
			Expect(k8sClient.Create(ctx, testOperatorConfigMap)).Should(Succeed())

			extraConfigMap := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "extra-config",
					Namespace: namespace,
				},
				Data: map[string]string{
					"config.conf": "key=value",
				},
			}
			Expect(k8sClient.Create(ctx, extraConfigMap)).Should(Succeed())

			spec := GetDefaultAnsibleTestSpec()
			spec["extraMounts"] = []map[string]any{
				{
					"extraVol": []map[string]any{
						{
							"propagation": []string{"AnsibleTest"},
							"volumes": []map[string]any{
								{
									"name": "extra-config-vol",
									"configMap": map[string]any{
										"name": "extra-config",
									},
								},
							},
							"mounts": []map[string]any{
								{
									"name":      "extra-config-vol",
									"mountPath": "/etc/extra-config",
									"readOnly":  true,
								},
							},
						},
					},
				},
			}
			DeferCleanup(th.DeleteInstance, CreateAnsibleTest(ansibleTestName, spec))
		})

		It("should add extra volume and volumeMount to the pod", func() {
			pod := GetTestOperatorPod(namespace, ansibleTestName.Name)

			foundVolume := false
			for _, vol := range pod.Spec.Volumes {
				if vol.Name == "extra-config-vol" {
					foundVolume = true
					Expect(vol.VolumeSource.ConfigMap).NotTo(BeNil())
					Expect(vol.VolumeSource.ConfigMap.Name).To(Equal("extra-config"))
					break
				}
			}
			Expect(foundVolume).To(BeTrue(), "expected pod to have volume 'extra-config-vol'")

			container := pod.Spec.Containers[0]
			foundMount := false
			for _, mount := range container.VolumeMounts {
				if mount.Name == "extra-config-vol" {
					foundMount = true
					Expect(mount.MountPath).To(Equal("/etc/extra-config"))
					Expect(mount.ReadOnly).To(BeTrue())
					break
				}
			}
			Expect(foundMount).To(BeTrue(), "expected container to have volumeMount 'extra-config-vol'")
		})
	})

	When("AnsibleTest is created with Secret as the source of extraMount", func() {
		BeforeEach(func() {
			openstackConfigMap, openstackSecret := CreateCommonOpenstackResources(namespace)
			Expect(k8sClient.Create(ctx, openstackConfigMap)).Should(Succeed())
			Expect(k8sClient.Create(ctx, openstackSecret)).Should(Succeed())

			testOperatorConfigMap := CreateTestOperatorConfigMap(namespace)
			Expect(k8sClient.Create(ctx, testOperatorConfigMap)).Should(Succeed())

			extraSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "extra-secret",
					Namespace: namespace,
				},
				StringData: map[string]string{
					"secret.conf": "key=value",
				},
			}
			Expect(k8sClient.Create(ctx, extraSecret)).Should(Succeed())

			spec := GetDefaultAnsibleTestSpec()
			spec["extraMounts"] = []map[string]any{
				{
					"extraVol": []map[string]any{
						{
							"propagation": []string{"AnsibleTest"},
							"volumes": []map[string]any{
								{
									"name": "extra-secret-vol",
									"secret": map[string]any{
										"secretName": "extra-secret",
									},
								},
							},
							"mounts": []map[string]any{
								{
									"name":      "extra-secret-vol",
									"mountPath": "/etc/extra-secret",
									"readOnly":  true,
								},
							},
						},
					},
				},
			}
			DeferCleanup(th.DeleteInstance, CreateAnsibleTest(ansibleTestName, spec))
		})

		It("should add secret based extra volume and volumeMount to the pod", func() {
			pod := GetTestOperatorPod(namespace, ansibleTestName.Name)

			foundVolume := false
			for _, vol := range pod.Spec.Volumes {
				if vol.Name == "extra-secret-vol" {
					foundVolume = true
					Expect(vol.VolumeSource.Secret).NotTo(BeNil())
					Expect(vol.VolumeSource.Secret.SecretName).To(Equal("extra-secret"))
					break
				}
			}
			Expect(foundVolume).To(BeTrue(), "expected pod to have volume 'extra-secret-vol'")

			container := pod.Spec.Containers[0]
			foundMount := false
			for _, mount := range container.VolumeMounts {
				if mount.Name == "extra-secret-vol" {
					foundMount = true
					Expect(mount.MountPath).To(Equal("/etc/extra-secret"))
					Expect(mount.ReadOnly).To(BeTrue())
					break
				}
			}
			Expect(foundMount).To(BeTrue(), "expected container to have volumeMount 'extra-secret-vol'")
		})
	})

	When("AnsibleTest is created with multiple extraMounts", func() {
		BeforeEach(func() {
			openstackConfigMap, openstackSecret := CreateCommonOpenstackResources(namespace)
			Expect(k8sClient.Create(ctx, openstackConfigMap)).Should(Succeed())
			Expect(k8sClient.Create(ctx, openstackSecret)).Should(Succeed())

			testOperatorConfigMap := CreateTestOperatorConfigMap(namespace)
			Expect(k8sClient.Create(ctx, testOperatorConfigMap)).Should(Succeed())

			extraConfigMap := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "extra-config",
					Namespace: namespace,
				},
				Data: map[string]string{
					"config.conf": "key=value",
				},
			}
			Expect(k8sClient.Create(ctx, extraConfigMap)).Should(Succeed())

			extraDataMap := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "extra-data",
					Namespace: namespace,
				},
				Data: map[string]string{
					"data.conf": "key=value",
				},
			}
			Expect(k8sClient.Create(ctx, extraDataMap)).Should(Succeed())

			spec := GetDefaultAnsibleTestSpec()
			spec["extraMounts"] = []map[string]any{
				{
					"extraVol": []map[string]any{
						{
							"propagation": []string{"AnsibleTest"},
							"volumes": []map[string]any{
								{
									"name": "extra-config-vol",
									"configMap": map[string]any{
										"name": "extra-config",
									},
								},
								{
									"name": "extra-data-vol",
									"configMap": map[string]any{
										"name": "extra-data",
									},
								},
							},
							"mounts": []map[string]any{
								{
									"name":      "extra-config-vol",
									"mountPath": "/etc/extra-config",
									"readOnly":  true,
								},
								{
									"name":      "extra-data-vol",
									"mountPath": "/etc/extra-data",
									"readOnly":  true,
								},
							},
						},
					},
				},
			}
			DeferCleanup(th.DeleteInstance, CreateAnsibleTest(ansibleTestName, spec))
		})

		It("should add all extra volumes and volumeMounts to the pod", func() {
			pod := GetTestOperatorPod(namespace, ansibleTestName.Name)

			volumeNames := []string{}
			for _, vol := range pod.Spec.Volumes {
				volumeNames = append(volumeNames, vol.Name)
			}
			Expect(volumeNames).To(ContainElement("extra-config-vol"))
			Expect(volumeNames).To(ContainElement("extra-data-vol"))

			mountNames := []string{}
			container := pod.Spec.Containers[0]
			for _, mount := range container.VolumeMounts {
				mountNames = append(mountNames, mount.Name)
			}
			Expect(mountNames).To(ContainElement("extra-config-vol"))
			Expect(mountNames).To(ContainElement("extra-data-vol"))
		})
	})

	When("AnsibleTest is created with no propagation field", func() {
		BeforeEach(func() {
			openstackConfigMap, openstackSecret := CreateCommonOpenstackResources(namespace)
			Expect(k8sClient.Create(ctx, openstackConfigMap)).Should(Succeed())
			Expect(k8sClient.Create(ctx, openstackSecret)).Should(Succeed())

			testOperatorConfigMap := CreateTestOperatorConfigMap(namespace)
			Expect(k8sClient.Create(ctx, testOperatorConfigMap)).Should(Succeed())

			extraConfigMap := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "extra-config",
					Namespace: namespace,
				},
				Data: map[string]string{
					"config.conf": "key=value",
				},
			}
			Expect(k8sClient.Create(ctx, extraConfigMap)).Should(Succeed())

			spec := GetDefaultAnsibleTestSpec()
			spec["extraMounts"] = []map[string]any{
				{
					"extraVol": []map[string]any{
						{
							"volumes": []map[string]any{
								{
									"name": "extra-config-vol",
									"configMap": map[string]any{
										"name": "extra-config",
									},
								},
							},
							"mounts": []map[string]any{
								{
									"name":      "extra-config-vol",
									"mountPath": "/etc/extra-config",
									"readOnly":  true,
								},
							},
						},
					},
				},
			}
			DeferCleanup(th.DeleteInstance, CreateAnsibleTest(ansibleTestName, spec))
		})

		It("should add extra volume and volumeMount when propagation is omitted", func() {
			pod := GetTestOperatorPod(namespace, ansibleTestName.Name)

			foundVolume := false
			for _, vol := range pod.Spec.Volumes {
				if vol.Name == "extra-config-vol" {
					foundVolume = true
					Expect(vol.VolumeSource.ConfigMap).NotTo(BeNil())
					Expect(vol.VolumeSource.ConfigMap.Name).To(Equal("extra-config"))
					break
				}
			}
			Expect(foundVolume).To(BeTrue(), "expected pod to have volume 'extra-config-vol'")

			container := pod.Spec.Containers[0]
			foundMount := false
			for _, mount := range container.VolumeMounts {
				if mount.Name == "extra-config-vol" {
					foundMount = true
					Expect(mount.MountPath).To(Equal("/etc/extra-config"))
					Expect(mount.ReadOnly).To(BeTrue())
					break
				}
			}
			Expect(foundMount).To(BeTrue(), "expected container to have volumeMount 'extra-config-vol'")
		})
	})

	When("AnsibleTest created with extraMounts using the wrong propagation type", func() {
		BeforeEach(func() {
			openstackConfigMap, openstackSecret := CreateCommonOpenstackResources(namespace)
			Expect(k8sClient.Create(ctx, openstackConfigMap)).Should(Succeed())
			Expect(k8sClient.Create(ctx, openstackSecret)).Should(Succeed())

			testOperatorConfigMap := CreateTestOperatorConfigMap(namespace)
			Expect(k8sClient.Create(ctx, testOperatorConfigMap)).Should(Succeed())

			extraConfigMap := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "extra-config",
					Namespace: namespace,
				},
				Data: map[string]string{
					"config.conf": "key=value",
				},
			}
			Expect(k8sClient.Create(ctx, extraConfigMap)).Should(Succeed())

			spec := GetDefaultAnsibleTestSpec()
			spec["extraMounts"] = []map[string]any{
				{
					"extraVol": []map[string]any{
						{
							"propagation": []string{"Tempest"},
							"volumes": []map[string]any{
								{
									"name": "extra-config-vol",
									"configMap": map[string]any{
										"name": "extra-config",
									},
								},
							},
							"mounts": []map[string]any{
								{
									"name":      "extra-config-vol",
									"mountPath": "/etc/extra-config",
									"readOnly":  true,
								},
							},
						},
					},
				},
			}
			DeferCleanup(th.DeleteInstance, CreateAnsibleTest(ansibleTestName, spec))
		})

		It("should not add extra volume and volumeMount to the pod", func() {
			pod := GetTestOperatorPod(namespace, ansibleTestName.Name)
			for _, vol := range pod.Spec.Volumes {
				Expect(vol.Name).NotTo(Equal("extra-config-vol"),
					"volume should not be propagated with wrong propagation type")
			}

			container := pod.Spec.Containers[0]
			for _, mount := range container.VolumeMounts {
				Expect(mount.Name).NotTo(Equal("extra-config-vol"),
					"volumeMount should not be propagated with wrong propagation type")
			}
		})
	})

})
