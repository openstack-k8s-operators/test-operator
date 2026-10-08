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
	"fmt"

	. "github.com/onsi/ginkgo/v2" //revive:disable:dot-imports
	. "github.com/onsi/gomega"    //revive:disable:dot-imports

	"github.com/openstack-k8s-operators/lib-common/modules/common/condition"

	//revive:disable-next-line:dot-imports
	. "github.com/openstack-k8s-operators/lib-common/modules/common/test/helpers"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Tempest controller", func() {
	var tempestName types.NamespacedName

	BeforeEach(func() {
		tempestName = types.NamespacedName{
			Name:      "tempest-tests",
			Namespace: namespace,
		}
	})

	DescribeTable("Missing Openstack resources should set InputReady to false",
		func(createResource func()) {
			createResource()
			DeferCleanup(th.DeleteInstance, CreateTempest(tempestName, GetDefaultTempestSpec()))

			th.ExpectCondition(
				tempestName,
				ConditionGetterFunc(TempestConditionGetter),
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

	When("A Tempest instance is created", func() {
		BeforeEach(func() {
			openstackConfigMap, openstackSecret := CreateCommonOpenstackResources(namespace)
			Expect(k8sClient.Create(ctx, openstackConfigMap)).Should(Succeed())
			Expect(k8sClient.Create(ctx, openstackSecret)).Should(Succeed())
			DeferCleanup(th.DeleteInstance, CreateTempest(tempestName, GetDefaultTempestSpec()))
		})

		It("initializes the status fields", func() {
			Eventually(func(g Gomega) {
				tempest := GetTempest(tempestName)
				g.Expect(tempest.Status.Conditions).To(HaveLen(5))
				g.Expect(tempest.Status.Hash).To(BeEmpty())
				g.Expect(tempest.Status.NetworkAttachments).To(BeEmpty())
			}, timeout*2, interval).Should(Succeed())
		})

		It("should have the Spec fields initialized", func() {
			tempest := GetTempest(tempestName)
			Expect(tempest.Spec.StorageClass).Should(Equal(DefaultStorageClass))
			Expect(tempest.Spec.TempestRun.IncludeList).ShouldNot(BeEmpty())
		})

		It("should have a finalizer", func() {
			Eventually(func() []string {
				return GetTempest(tempestName).Finalizers
			}, timeout, interval).Should(ContainElement("openstack.org/tempest"))
		})
	})

	When("All dependencies are ready", func() {
		var customDataConfigMapName string
		var envVarsConfigMapName string

		BeforeEach(func() {
			openstackConfigMap, openstackSecret := CreateCommonOpenstackResources(namespace)
			Expect(k8sClient.Create(ctx, openstackConfigMap)).Should(Succeed())
			Expect(k8sClient.Create(ctx, openstackSecret)).Should(Succeed())
			DeferCleanup(th.DeleteInstance, CreateTempest(tempestName, GetDefaultTempestSpec()))

			customDataConfigMapName = fmt.Sprintf("%s-custom-data-s0", tempestName.Name)
			envVarsConfigMapName = fmt.Sprintf("%s-env-vars-s0", tempestName.Name)
		})

		It("should have InputReady condition true", func() {
			th.ExpectCondition(
				tempestName,
				ConditionGetterFunc(TempestConditionGetter),
				condition.InputReadyCondition,
				corev1.ConditionTrue,
			)
		})

		It("should create a PVC for logs", func() {
			pvc := GetTestOperatorPVC(namespace, tempestName.Name)
			Expect(pvc.Name).ToNot(BeEmpty())
			Expect(*pvc.Spec.StorageClassName).To(Equal(DefaultStorageClass))
			Expect(pvc.Spec.AccessModes).To(ContainElement(corev1.ReadWriteOnce))
		})

		It("should create required ConfigMaps", func() {
			customDataCM := th.GetConfigMap(types.NamespacedName{
				Namespace: namespace,
				Name:      customDataConfigMapName,
			})
			Expect(customDataCM.Data).To(HaveKey("include.txt"))

			envVarsCM := th.GetConfigMap(types.NamespacedName{
				Namespace: namespace,
				Name:      envVarsConfigMapName,
			})
			Expect(envVarsCM.Data).NotTo(BeEmpty())
		})

		It("should create a pod", func() {
			pod := GetTestOperatorPod(namespace, tempestName.Name)
			Expect(pod.Name).ToNot(BeEmpty())
		})
	})

	When("Tempest is created with network attachments", func() {
		var networkAttachmentName = "ctlplane"

		BeforeEach(func() {
			openstackConfigMap, openstackSecret := CreateCommonOpenstackResources(namespace)
			Expect(k8sClient.Create(ctx, openstackConfigMap)).Should(Succeed())
			Expect(k8sClient.Create(ctx, openstackSecret)).Should(Succeed())

			nad := th.CreateNetworkAttachmentDefinition(types.NamespacedName{
				Namespace: namespace,
				Name:      networkAttachmentName,
			})
			DeferCleanup(th.DeleteInstance, nad)

			spec := GetDefaultTempestSpec()
			spec["networkAttachments"] = []string{networkAttachmentName}
			DeferCleanup(th.DeleteInstance, CreateTempest(tempestName, spec))
		})

		It("should add network annotation to pod", func() {
			pod := GetTestOperatorPod(namespace, tempestName.Name)
			Expect(pod.Annotations).To(HaveKey("k8s.v1.cni.cncf.io/networks"))
			Expect(pod.Annotations["k8s.v1.cni.cncf.io/networks"]).To(ContainSubstring(networkAttachmentName))
		})
	})

	When("Tempest is created with non-existent network attachments", func() {
		BeforeEach(func() {
			openstackConfigMap, openstackSecret := CreateCommonOpenstackResources(namespace)
			Expect(k8sClient.Create(ctx, openstackConfigMap)).Should(Succeed())
			Expect(k8sClient.Create(ctx, openstackSecret)).Should(Succeed())

			spec := GetDefaultTempestSpec()
			spec["networkAttachments"] = []string{"non-existent-nad"}
			DeferCleanup(th.DeleteInstance, CreateTempest(tempestName, spec))
		})

		It("should set NetworkAttachmentsReady to false", func() {
			th.ExpectCondition(
				tempestName,
				ConditionGetterFunc(TempestConditionGetter),
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

		When("Tempest is created with extraMounts", func() {
			BeforeEach(func() {
				CreateExtraConfigMap(namespace, ExtraConfigMapName)

				spec := GetDefaultTempestSpec()
				spec["extraMounts"] = BuildExtraMountsSpec("Tempest",
					GetDefaultConfigMapExtraMount())

				DeferCleanup(th.DeleteInstance, CreateTempest(tempestName, spec))
			})

			It("should add extra volume and volumeMount to the pod", func() {
				pod := GetTestOperatorPod(namespace, tempestName.Name)
				ExpectPodHasConfigMapVolume(pod, ExtraConfigVolName, ExtraConfigMapName)
				ExpectPodHasVolumeMount(pod, ExtraConfigVolName, ExtraConfigMountPath)
			})
		})

		When("Tempest is created with Secret as the source of extraMount", func() {
			BeforeEach(func() {
				CreateExtraSecret(namespace, ExtraSecretName)

				spec := GetDefaultTempestSpec()
				spec["extraMounts"] = BuildExtraMountsSpec("Tempest",
					GetDefaultSecretExtraMount())

				DeferCleanup(th.DeleteInstance, CreateTempest(tempestName, spec))
			})

			It("should add secret based extra volume and volumeMount to the pod", func() {
				pod := GetTestOperatorPod(namespace, tempestName.Name)
				ExpectPodHasSecretVolume(pod, ExtraSecretVolName, ExtraSecretName)
				ExpectPodHasVolumeMount(pod, ExtraSecretVolName, ExtraSecretMountPath)
			})
		})

		When("Tempest is created with multiple extraMounts configmap and secret", func() {
			BeforeEach(func() {
				CreateExtraConfigMap(namespace, ExtraConfigMapName)
				CreateExtraSecret(namespace, ExtraSecretName)

				spec := GetDefaultTempestSpec()
				spec["extraMounts"] = BuildExtraMountsSpec("Tempest",
					GetDefaultConfigMapExtraMount(),
					GetDefaultSecretExtraMount(),
				)

				DeferCleanup(th.DeleteInstance, CreateTempest(tempestName, spec))
			})

			It("should add all extra volumes and volumeMounts to the pod", func() {
				pod := GetTestOperatorPod(namespace, tempestName.Name)
				ExpectPodHasConfigMapVolume(pod, ExtraConfigVolName, ExtraConfigMapName)
				ExpectPodHasSecretVolume(pod, ExtraSecretVolName, ExtraSecretName)
				ExpectPodHasVolumeMount(pod, ExtraConfigVolName, ExtraConfigMountPath)
				ExpectPodHasVolumeMount(pod, ExtraSecretVolName, ExtraSecretMountPath)
			})
		})

		When("Tempest is created with no propagation field", func() {
			BeforeEach(func() {
				CreateExtraConfigMap(namespace, ExtraConfigMapName)

				spec := GetDefaultTempestSpec()
				spec["extraMounts"] = BuildExtraMountsSpec("",
					GetDefaultConfigMapExtraMount())

				DeferCleanup(th.DeleteInstance, CreateTempest(tempestName, spec))
			})

			It("should add extra volume and volumeMount when propagation is omitted", func() {
				pod := GetTestOperatorPod(namespace, tempestName.Name)
				ExpectPodHasConfigMapVolume(pod, ExtraConfigVolName, ExtraConfigMapName)
				ExpectPodHasVolumeMount(pod, ExtraConfigVolName, ExtraConfigMountPath)
			})
		})

		When("Tempest created with extraMounts is using the wrong propagation type", func() {
			BeforeEach(func() {
				CreateExtraConfigMap(namespace, ExtraConfigMapName)

				spec := GetDefaultTempestSpec()
				spec["extraMounts"] = BuildExtraMountsSpec("HorizonTest",
					GetDefaultConfigMapExtraMount())

				DeferCleanup(th.DeleteInstance, CreateTempest(tempestName, spec))
			})

			It("should not add extra volume and volumeMount to the pod", func() {
				pod := GetTestOperatorPod(namespace, tempestName.Name)
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
				DeferCleanup(th.DeleteInstance, CreateTempest(tempestName, GetDefaultTempestWorkflowSpec()))
			})

			It("creates resources with workflow step name", func() {
				customDataCM := th.GetConfigMap(types.NamespacedName{
					Namespace: namespace,
					Name:      fmt.Sprintf("%s-custom-data-s0", tempestName.Name),
				})
				Expect(customDataCM.Data).To(HaveKey("include.txt"))

				envVarsCM := th.GetConfigMap(types.NamespacedName{
					Namespace: namespace,
					Name:      fmt.Sprintf("%s-env-vars-s0", tempestName.Name),
				})
				Expect(envVarsCM.Data).NotTo(BeEmpty())
			})

			It("creates PVC with workflow step name", func() {
				pvc := GetTestOperatorPVC(namespace, tempestName.Name)
				Expect(pvc.Name).To(ContainSubstring(tempestName.Name + "-0-"))
			})

			It("creates pod with workflow step name", func() {
				pod := GetTestOperatorPod(namespace, tempestName.Name)
				spec := GetDefaultTempestWorkflowSpec()
				workflow := spec["workflow"].([]map[string]any)
				stepName := workflow[0]["stepName"].(string)
				Expect(pod.Name).To(Equal(tempestName.Name + "-s00-" + stepName))
			})
		})

		When("overrides spec defaults", func() {
			BeforeEach(func() {
				openstackConfigMap, openstackSecret := CreateCommonOpenstackResources(namespace)
				Expect(k8sClient.Create(ctx, openstackConfigMap)).Should(Succeed())
				Expect(k8sClient.Create(ctx, openstackSecret)).Should(Succeed())
				DeferCleanup(th.DeleteInstance, CreateTempest(tempestName, GetDefaultTempestWorkflowSpec()))
			})

			It("workflow tempestRun values take precedence", func() {
				customDataCM := th.GetConfigMap(types.NamespacedName{
					Namespace: namespace,
					Name:      fmt.Sprintf("%s-custom-data-s0", tempestName.Name),
				})
				Expect(customDataCM.Data["include.txt"]).To(ContainSubstring("tempest.api.compute.*"))
				Expect(customDataCM.Data["include.txt"]).NotTo(ContainSubstring("tempest.api.identity.v3.*"))
			})

			It("workflow tempestconfRun values take precedence", func() {
				envVarsCM := th.GetConfigMap(types.NamespacedName{
					Namespace: namespace,
					Name:      fmt.Sprintf("%s-env-vars-s0", tempestName.Name),
				})
				Expect(envVarsCM.Data["TEMPESTCONF_NETWORK_ID"]).To(Equal("workflow-network-id"))
				Expect(envVarsCM.Data["TEMPESTCONF_NETWORK_ID"]).NotTo(Equal("spec-network-id"))
			})
		})

		When("inherits from spec defaults", func() {
			BeforeEach(func() {
				openstackConfigMap, openstackSecret := CreateCommonOpenstackResources(namespace)
				Expect(k8sClient.Create(ctx, openstackConfigMap)).Should(Succeed())
				Expect(k8sClient.Create(ctx, openstackSecret)).Should(Succeed())

				spec := GetDefaultTempestWorkflowSpec()
				workflow := spec["workflow"].([]map[string]any)
				delete(workflow[0], "tempestRun")
				delete(workflow[0], "tempestconfRun")
				DeferCleanup(th.DeleteInstance, CreateTempest(tempestName, spec))
			})

			It("uses spec-level values when workflow step omits them", func() {
				customDataCM := th.GetConfigMap(types.NamespacedName{
					Namespace: namespace,
					Name:      fmt.Sprintf("%s-custom-data-s0", tempestName.Name),
				})
				Expect(customDataCM.Data["include.txt"]).To(ContainSubstring("tempest.api.identity.v3.*"))

				envVarsCM := th.GetConfigMap(types.NamespacedName{
					Namespace: namespace,
					Name:      fmt.Sprintf("%s-env-vars-s0", tempestName.Name),
				})
				Expect(envVarsCM.Data["TEMPESTCONF_NETWORK_ID"]).To(Equal("spec-network-id"))
			})
		})

		When("with multiple workflow steps", func() {
			BeforeEach(func() {
				openstackConfigMap, openstackSecret := CreateCommonOpenstackResources(namespace)
				Expect(k8sClient.Create(ctx, openstackConfigMap)).Should(Succeed())
				Expect(k8sClient.Create(ctx, openstackSecret)).Should(Succeed())
				DeferCleanup(th.DeleteInstance, CreateTempest(tempestName, GetDefaultTempestWorkflowSpec()))
			})

			It("creates second pod after first pod succeeds", func() {
				firstPod := GetTestOperatorPod(namespace, tempestName.Name)
				Expect(firstPod.Name).To(Equal(tempestName.Name + "-s00-first-step"))

				firstPod.Status.Phase = corev1.PodSucceeded
				Expect(k8sClient.Status().Update(ctx, firstPod)).Should(Succeed())

				Eventually(func(g Gomega) {
					podList := &corev1.PodList{}
					listOpts := []client.ListOption{
						client.InNamespace(namespace),
						client.MatchingLabels{
							"instanceName": tempestName.Name,
							"operator":     "test-operator",
							"workflowStep": "1",
						},
					}
					g.Expect(k8sClient.List(ctx, podList, listOpts...)).Should(Succeed())
					g.Expect(podList.Items).To(HaveLen(1))
					g.Expect(podList.Items[0].Name).To(Equal(tempestName.Name + "-s01-second-step"))

					customDataCM := th.GetConfigMap(types.NamespacedName{
						Namespace: namespace,
						Name:      fmt.Sprintf("%s-custom-data-s1", tempestName.Name),
					})
					g.Expect(customDataCM.Data["include.txt"]).To(ContainSubstring("tempest.api.network.*"))

					envVarsCM := th.GetConfigMap(types.NamespacedName{
						Namespace: namespace,
						Name:      fmt.Sprintf("%s-env-vars-s1", tempestName.Name),
					})
					g.Expect(envVarsCM.Data["TEMPESTCONF_NETWORK_ID"]).To(Equal("spec-network-id"))
				}, timeout, interval).Should(Succeed())
			})
		})

		When("with networkAttachments", func() {
			var networkAttachmentName = "ctlplane"

			BeforeEach(func() {
				openstackConfigMap, openstackSecret := CreateCommonOpenstackResources(namespace)
				Expect(k8sClient.Create(ctx, openstackConfigMap)).Should(Succeed())
				Expect(k8sClient.Create(ctx, openstackSecret)).Should(Succeed())

				nad := th.CreateNetworkAttachmentDefinition(types.NamespacedName{
					Namespace: namespace,
					Name:      networkAttachmentName,
				})
				DeferCleanup(th.DeleteInstance, nad)

				spec := GetDefaultTempestWorkflowSpec()
				workflow := spec["workflow"].([]map[string]any)
				workflow[0]["networkAttachments"] = []string{networkAttachmentName}
				DeferCleanup(th.DeleteInstance, CreateTempest(tempestName, spec))
			})

			It("adds network annotation to workflow pod", func() {
				pod := GetTestOperatorPod(namespace, tempestName.Name)
				Expect(pod.Annotations).To(HaveKey("k8s.v1.cni.cncf.io/networks"))
				Expect(pod.Annotations["k8s.v1.cni.cncf.io/networks"]).To(ContainSubstring(networkAttachmentName))
			})
		})

		When("with non-existent networkAttachments", func() {
			BeforeEach(func() {
				openstackConfigMap, openstackSecret := CreateCommonOpenstackResources(namespace)
				Expect(k8sClient.Create(ctx, openstackConfigMap)).Should(Succeed())
				Expect(k8sClient.Create(ctx, openstackSecret)).Should(Succeed())

				spec := GetDefaultTempestWorkflowSpec()
				workflow := spec["workflow"].([]map[string]any)
				workflow[0]["networkAttachments"] = []string{"non-existent-nad"}
				DeferCleanup(th.DeleteInstance, CreateTempest(tempestName, spec))
			})

			It("should set NetworkAttachmentsReady to false", func() {
				th.ExpectCondition(
					tempestName,
					ConditionGetterFunc(TempestConditionGetter),
					condition.NetworkAttachmentsReadyCondition,
					corev1.ConditionFalse,
				)
			})
		})
	})
})
