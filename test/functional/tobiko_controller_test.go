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
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Tobiko controller", func() {
	var tobikoName types.NamespacedName

	BeforeEach(func() {
		tobikoName = types.NamespacedName{
			Name:      "tobiko",
			Namespace: namespace,
		}
	})

	DescribeTable("Missing Openstack resources should set InputReady to false",
		func(createResource func()) {
			createResource()
			DeferCleanup(th.DeleteInstance, CreateTobiko(tobikoName, GetDefaultTobikoSpec()))

			th.ExpectCondition(
				tobikoName,
				ConditionGetterFunc(TobikoConditionGetter),
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

	When("A Tobiko instance is created", func() {
		BeforeEach(func() {
			openstackConfigMap, openstackSecret := CreateCommonOpenstackResources(namespace)
			Expect(k8sClient.Create(ctx, openstackConfigMap)).Should(Succeed())
			Expect(k8sClient.Create(ctx, openstackSecret)).Should(Succeed())
			DeferCleanup(th.DeleteInstance, CreateTobiko(tobikoName, GetDefaultTobikoSpec()))
		})

		It("initializes the status fields", func() {
			Eventually(func(g Gomega) {
				tobiko := GetTobiko(tobikoName)
				g.Expect(tobiko.Status.Conditions).To(HaveLen(5))
				g.Expect(tobiko.Status.Hash).To(BeEmpty())
				g.Expect(tobiko.Status.NetworkAttachments).To(BeEmpty())
			}, timeout*2, interval).Should(Succeed())
		})

		It("should have the Spec fields initialized", func() {
			tobiko := GetTobiko(tobikoName)
			Expect(tobiko.Spec.StorageClass).Should(Equal(DefaultStorageClass))
			Expect(tobiko.Spec.Testenv).Should(Equal("sanity"))
		})

		It("should have a finalizer", func() {
			Eventually(func() []string {
				return GetTobiko(tobikoName).Finalizers
			}, timeout, interval).Should(ContainElement("openstack.org/tobiko"))
		})
	})

	When("All dependencies are ready", func() {
		BeforeEach(func() {
			openstackConfigMap, openstackSecret := CreateCommonOpenstackResources(namespace)
			Expect(k8sClient.Create(ctx, openstackConfigMap)).Should(Succeed())
			Expect(k8sClient.Create(ctx, openstackSecret)).Should(Succeed())

			testOperatorConfigMap := CreateTestOperatorConfigMap(namespace)
			Expect(k8sClient.Create(ctx, testOperatorConfigMap)).Should(Succeed())

			DeferCleanup(th.DeleteInstance, CreateTobiko(tobikoName, GetDefaultTobikoSpec()))
		})

		It("should have InputReady condition true", func() {
			th.ExpectCondition(
				tobikoName,
				ConditionGetterFunc(TobikoConditionGetter),
				condition.InputReadyCondition,
				corev1.ConditionTrue,
			)
		})

		It("should create a PVC for logs", func() {
			pvc := GetTestOperatorPVC(namespace, tobikoName.Name)
			Expect(pvc.Name).ToNot(BeEmpty())
			Expect(*pvc.Spec.StorageClassName).To(Equal(DefaultStorageClass))
			Expect(pvc.Spec.AccessModes).To(ContainElement(corev1.ReadWriteOnce))
		})

		It("should create a pod", func() {
			pod := GetTestOperatorPod(namespace, tobikoName.Name)
			Expect(pod.Name).ToNot(BeEmpty())
		})
	})

	When("Tobiko is created with network attachments", func() {
		var networkAttachmentName = "ctlplane"

		BeforeEach(func() {
			openstackConfigMap, openstackSecret := CreateCommonOpenstackResources(namespace)
			Expect(k8sClient.Create(ctx, openstackConfigMap)).Should(Succeed())
			Expect(k8sClient.Create(ctx, openstackSecret)).Should(Succeed())

			testOperatorConfigMap := CreateTestOperatorConfigMap(namespace)
			Expect(k8sClient.Create(ctx, testOperatorConfigMap)).Should(Succeed())

			nad := th.CreateNetworkAttachmentDefinition(types.NamespacedName{
				Namespace: namespace,
				Name:      networkAttachmentName,
			})
			DeferCleanup(th.DeleteInstance, nad)

			spec := GetDefaultTobikoSpec()
			spec["networkAttachments"] = []string{networkAttachmentName}
			DeferCleanup(th.DeleteInstance, CreateTobiko(tobikoName, spec))
		})

		It("should add network annotation to pod", func() {
			pod := GetTestOperatorPod(namespace, tobikoName.Name)
			Expect(pod.Annotations).To(HaveKey("k8s.v1.cni.cncf.io/networks"))
			Expect(pod.Annotations["k8s.v1.cni.cncf.io/networks"]).To(ContainSubstring(networkAttachmentName))
		})
	})

	When("Tobiko is created with non-existent network attachments", func() {
		BeforeEach(func() {
			openstackConfigMap, openstackSecret := CreateCommonOpenstackResources(namespace)
			Expect(k8sClient.Create(ctx, openstackConfigMap)).Should(Succeed())
			Expect(k8sClient.Create(ctx, openstackSecret)).Should(Succeed())

			testOperatorConfigMap := CreateTestOperatorConfigMap(namespace)
			Expect(k8sClient.Create(ctx, testOperatorConfigMap)).Should(Succeed())

			spec := GetDefaultTobikoSpec()
			spec["networkAttachments"] = []string{"non-existent-nad"}
			DeferCleanup(th.DeleteInstance, CreateTobiko(tobikoName, spec))
		})

		It("should set NetworkAttachmentsReady to false", func() {
			th.ExpectCondition(
				tobikoName,
				ConditionGetterFunc(TobikoConditionGetter),
				condition.NetworkAttachmentsReadyCondition,
				corev1.ConditionFalse,
			)
		})
	})

	Context("extraMounts", func() {
		BeforeEach(func() {
			openstackConfigMap, openstackSecret := CreateCommonOpenstackResources(namespace)
			Expect(k8sClient.Create(ctx, openstackConfigMap)).Should(Succeed())
			Expect(k8sClient.Create(ctx, openstackSecret)).Should(Succeed())

			testOperatorConfigMap := CreateTestOperatorConfigMap(namespace)
			Expect(k8sClient.Create(ctx, testOperatorConfigMap)).Should(Succeed())
		})

		When("Tobiko is created with extraMounts", func() {
			BeforeEach(func() {
				CreateExtraConfigMap(namespace, ExtraConfigMapName)

				spec := GetDefaultTobikoSpec()
				spec["extraMounts"] = BuildExtraMountsSpec("Tobiko",
					GetDefaultConfigMapExtraMount())

				DeferCleanup(th.DeleteInstance, CreateTobiko(tobikoName, spec))
			})

			It("should add extra volume and volumeMount to the pod", func() {
				pod := GetTestOperatorPod(namespace, tobikoName.Name)
				ExpectPodHasConfigMapVolume(pod, ExtraConfigVolName, ExtraConfigMapName)
				ExpectPodHasVolumeMount(pod, ExtraConfigVolName, ExtraConfigMountPath)
			})
		})

		When("Tobiko is created with Secret as the source of extraMount", func() {
			BeforeEach(func() {
				CreateExtraSecret(namespace, ExtraSecretName)

				spec := GetDefaultTobikoSpec()
				spec["extraMounts"] = BuildExtraMountsSpec("Tobiko",
					GetDefaultSecretExtraMount())

				DeferCleanup(th.DeleteInstance, CreateTobiko(tobikoName, spec))
			})

			It("should add secret based extra volume and volumeMount to the pod", func() {
				pod := GetTestOperatorPod(namespace, tobikoName.Name)
				ExpectPodHasSecretVolume(pod, ExtraSecretVolName, ExtraSecretName)
				ExpectPodHasVolumeMount(pod, ExtraSecretVolName, ExtraSecretMountPath)
			})
		})

		When("Tobiko is created with multiple extraMounts configmap and secret", func() {
			BeforeEach(func() {
				CreateExtraConfigMap(namespace, ExtraConfigMapName)
				CreateExtraSecret(namespace, ExtraSecretName)

				spec := GetDefaultTobikoSpec()
				spec["extraMounts"] = BuildExtraMountsSpec("Tobiko",
					GetDefaultConfigMapExtraMount(),
					GetDefaultSecretExtraMount(),
				)

				DeferCleanup(th.DeleteInstance, CreateTobiko(tobikoName, spec))
			})

			It("should add all extra volumes and volumeMounts to the pod", func() {
				pod := GetTestOperatorPod(namespace, tobikoName.Name)
				ExpectPodHasConfigMapVolume(pod, ExtraConfigVolName, ExtraConfigMapName)
				ExpectPodHasSecretVolume(pod, ExtraSecretVolName, ExtraSecretName)
				ExpectPodHasVolumeMount(pod, ExtraConfigVolName, ExtraConfigMountPath)
				ExpectPodHasVolumeMount(pod, ExtraSecretVolName, ExtraSecretMountPath)
			})
		})

		When("Tobiko is created with no propagation field", func() {
			BeforeEach(func() {
				CreateExtraConfigMap(namespace, ExtraConfigMapName)

				spec := GetDefaultTobikoSpec()
				spec["extraMounts"] = BuildExtraMountsSpec("",
					GetDefaultConfigMapExtraMount())

				DeferCleanup(th.DeleteInstance, CreateTobiko(tobikoName, spec))
			})

			It("should add extra volume and volumeMount when propagation is omitted", func() {
				pod := GetTestOperatorPod(namespace, tobikoName.Name)
				ExpectPodHasConfigMapVolume(pod, ExtraConfigVolName, ExtraConfigMapName)
				ExpectPodHasVolumeMount(pod, ExtraConfigVolName, ExtraConfigMountPath)
			})
		})

		When("Tobiko created with extraMounts is using the wrong propagation type", func() {
			BeforeEach(func() {
				CreateExtraConfigMap(namespace, ExtraConfigMapName)

				spec := GetDefaultTobikoSpec()
				spec["extraMounts"] = BuildExtraMountsSpec("HorizonTest",
					GetDefaultConfigMapExtraMount())

				DeferCleanup(th.DeleteInstance, CreateTobiko(tobikoName, spec))
			})

			It("should not add extra volume and volumeMount to the pod", func() {
				pod := GetTestOperatorPod(namespace, tobikoName.Name)
				ExpectPodNotHasVolume(pod, ExtraConfigVolName)
				ExpectPodNotHasVolumeMount(pod, ExtraConfigVolName)
			})
		})
	})

	Context("workflow", func() {
		When("is created", func() {
			BeforeEach(func() {
				openstackConfigMap, openstackSecret := CreateCommonOpenstackResources(namespace)
				Expect(k8sClient.Create(ctx, openstackConfigMap)).Should(Succeed())
				Expect(k8sClient.Create(ctx, openstackSecret)).Should(Succeed())

				testOperatorConfigMap := CreateTestOperatorConfigMap(namespace)
				Expect(k8sClient.Create(ctx, testOperatorConfigMap)).Should(Succeed())

				DeferCleanup(th.DeleteInstance, CreateTobiko(tobikoName, GetDefaultTobikoWorkflowSpec()))
			})

			It("creates PVC with workflow step name", func() {
				pvc := GetTestOperatorPVC(namespace, tobikoName.Name)
				Expect(pvc.Name).To(ContainSubstring(tobikoName.Name + "-0-"))
			})

			It("creates pod with workflow step name", func() {
				pod := GetTestOperatorPod(namespace, tobikoName.Name)
				spec := GetDefaultTobikoWorkflowSpec()
				workflow := spec["workflow"].([]map[string]any)
				stepName := workflow[0]["stepName"].(string)
				Expect(pod.Name).To(Equal(tobikoName.Name + "-s00-" + stepName))
			})
		})

		When("overrides spec defaults", func() {
			BeforeEach(func() {
				openstackConfigMap, openstackSecret := CreateCommonOpenstackResources(namespace)
				Expect(k8sClient.Create(ctx, openstackConfigMap)).Should(Succeed())
				Expect(k8sClient.Create(ctx, openstackSecret)).Should(Succeed())

				testOperatorConfigMap := CreateTestOperatorConfigMap(namespace)
				Expect(k8sClient.Create(ctx, testOperatorConfigMap)).Should(Succeed())

				DeferCleanup(th.DeleteInstance, CreateTobiko(tobikoName, GetDefaultTobikoWorkflowSpec()))
			})

			It("workflow testenv values take precedence", func() {
				pod := GetTestOperatorPod(namespace, tobikoName.Name)
				spec := GetDefaultTobikoWorkflowSpec()
				workflow := spec["workflow"].([]map[string]any)
				expectedTestenv := workflow[0]["testenv"].(string)

				Expect(GetPodEnvVar(pod, "TOBIKO_TESTENV")).To(Equal(expectedTestenv))
				Expect(GetPodEnvVar(pod, "TOBIKO_TESTENV")).NotTo(Equal(spec["testenv"].(string)))
			})
		})

		When("inherits from spec defaults", func() {
			BeforeEach(func() {
				openstackConfigMap, openstackSecret := CreateCommonOpenstackResources(namespace)
				Expect(k8sClient.Create(ctx, openstackConfigMap)).Should(Succeed())
				Expect(k8sClient.Create(ctx, openstackSecret)).Should(Succeed())

				testOperatorConfigMap := CreateTestOperatorConfigMap(namespace)
				Expect(k8sClient.Create(ctx, testOperatorConfigMap)).Should(Succeed())

				spec := GetDefaultTobikoWorkflowSpec()
				workflow := spec["workflow"].([]map[string]any)
				delete(workflow[0], "testenv")
				DeferCleanup(th.DeleteInstance, CreateTobiko(tobikoName, spec))
			})

			It("uses spec-level values when workflow step omits them", func() {
				pod := GetTestOperatorPod(namespace, tobikoName.Name)
				spec := GetDefaultTobikoWorkflowSpec()
				expectedTestenv := spec["testenv"].(string)

				Expect(GetPodEnvVar(pod, "TOBIKO_TESTENV")).To(Equal(expectedTestenv))
			})
		})

		When("with multiple workflow steps", func() {
			BeforeEach(func() {
				openstackConfigMap, openstackSecret := CreateCommonOpenstackResources(namespace)
				Expect(k8sClient.Create(ctx, openstackConfigMap)).Should(Succeed())
				Expect(k8sClient.Create(ctx, openstackSecret)).Should(Succeed())

				testOperatorConfigMap := CreateTestOperatorConfigMap(namespace)
				Expect(k8sClient.Create(ctx, testOperatorConfigMap)).Should(Succeed())

				DeferCleanup(th.DeleteInstance, CreateTobiko(tobikoName, GetDefaultTobikoWorkflowSpec()))
			})

			It("creates second pod after first pod succeeds", func() {
				firstPod := GetTestOperatorPod(namespace, tobikoName.Name)
				Expect(firstPod.Name).To(Equal(tobikoName.Name + "-s00-first-step"))

				firstPod.Status.Phase = corev1.PodSucceeded
				Expect(k8sClient.Status().Update(ctx, firstPod)).Should(Succeed())

				Eventually(func(g Gomega) {
					podList := &corev1.PodList{}
					listOpts := []client.ListOption{
						client.InNamespace(namespace),
						client.MatchingLabels{
							"instanceName": tobikoName.Name,
							"operator":     "test-operator",
							"workflowStep": "1",
						},
					}
					g.Expect(k8sClient.List(ctx, podList, listOpts...)).Should(Succeed())
					g.Expect(podList.Items).To(HaveLen(1))
					secondPod := podList.Items[0]
					g.Expect(secondPod.Name).To(Equal(tobikoName.Name + "-s01-second-step"))

					spec := GetDefaultTobikoWorkflowSpec()
					workflow := spec["workflow"].([]map[string]any)
					expectedTestenv := workflow[1]["testenv"].(string)
					g.Expect(GetPodEnvVar(&secondPod, "TOBIKO_TESTENV")).To(Equal(expectedTestenv))
				}, timeout, interval).Should(Succeed())
			})
		})

		When("with networkAttachments", func() {
			var networkAttachmentName = "ctlplane"

			BeforeEach(func() {
				openstackConfigMap, openstackSecret := CreateCommonOpenstackResources(namespace)
				Expect(k8sClient.Create(ctx, openstackConfigMap)).Should(Succeed())
				Expect(k8sClient.Create(ctx, openstackSecret)).Should(Succeed())

				testOperatorConfigMap := CreateTestOperatorConfigMap(namespace)
				Expect(k8sClient.Create(ctx, testOperatorConfigMap)).Should(Succeed())

				nad := th.CreateNetworkAttachmentDefinition(types.NamespacedName{
					Namespace: namespace,
					Name:      networkAttachmentName,
				})
				DeferCleanup(th.DeleteInstance, nad)

				spec := GetDefaultTobikoWorkflowSpec()
				workflow := spec["workflow"].([]map[string]any)
				workflow[0]["networkAttachments"] = []string{networkAttachmentName}
				DeferCleanup(th.DeleteInstance, CreateTobiko(tobikoName, spec))
			})

			It("adds network annotation to workflow pod", func() {
				pod := GetTestOperatorPod(namespace, tobikoName.Name)
				Expect(pod.Annotations).To(HaveKey("k8s.v1.cni.cncf.io/networks"))
				Expect(pod.Annotations["k8s.v1.cni.cncf.io/networks"]).To(ContainSubstring(networkAttachmentName))
			})
		})

		When("with non-existent networkAttachments", func() {
			BeforeEach(func() {
				openstackConfigMap, openstackSecret := CreateCommonOpenstackResources(namespace)
				Expect(k8sClient.Create(ctx, openstackConfigMap)).Should(Succeed())
				Expect(k8sClient.Create(ctx, openstackSecret)).Should(Succeed())

				testOperatorConfigMap := CreateTestOperatorConfigMap(namespace)
				Expect(k8sClient.Create(ctx, testOperatorConfigMap)).Should(Succeed())

				spec := GetDefaultTobikoWorkflowSpec()
				workflow := spec["workflow"].([]map[string]any)
				workflow[0]["networkAttachments"] = []string{"non-existent-nad"}
				DeferCleanup(th.DeleteInstance, CreateTobiko(tobikoName, spec))
			})

			It("should set NetworkAttachmentsReady to false", func() {
				th.ExpectCondition(
					tobikoName,
					ConditionGetterFunc(TobikoConditionGetter),
					condition.NetworkAttachmentsReadyCondition,
					corev1.ConditionFalse,
				)
			})
		})
	})
})
