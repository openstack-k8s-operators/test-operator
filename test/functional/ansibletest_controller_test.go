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

		It("should have a finalizer", func() {
			Eventually(func() []string {
				return GetAnsibleTest(ansibleTestName).Finalizers
			}, timeout, interval).Should(ContainElement("openstack.org/ansibletest"))
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

	Context("extraMounts", func() {
		BeforeEach(func() {
			openstackConfigMap, openstackSecret := CreateCommonOpenstackResources(namespace)
			Expect(k8sClient.Create(ctx, openstackConfigMap)).Should(Succeed())
			Expect(k8sClient.Create(ctx, openstackSecret)).Should(Succeed())

			testOperatorConfigMap := CreateTestOperatorConfigMap(namespace)
			Expect(k8sClient.Create(ctx, testOperatorConfigMap)).Should(Succeed())
		})

		When("AnsibleTest is created with extraMounts", func() {
			BeforeEach(func() {
				CreateExtraConfigMap(namespace, ExtraConfigMapName)

				spec := GetDefaultAnsibleTestSpec()
				spec["extraMounts"] = BuildExtraMountsSpec("AnsibleTest",
					GetDefaultConfigMapExtraMount())

				DeferCleanup(th.DeleteInstance, CreateAnsibleTest(ansibleTestName, spec))
			})

			It("should add extra volume and volumeMount to the pod", func() {
				pod := GetTestOperatorPod(namespace, ansibleTestName.Name)
				ExpectPodHasConfigMapVolume(pod, ExtraConfigVolName, ExtraConfigMapName)
				ExpectPodHasVolumeMount(pod, ExtraConfigVolName, ExtraConfigMountPath)
			})
		})

		When("AnsibleTest is created with Secret as the source of extraMount", func() {
			BeforeEach(func() {
				CreateExtraSecret(namespace, ExtraSecretName)

				spec := GetDefaultAnsibleTestSpec()
				spec["extraMounts"] = BuildExtraMountsSpec("AnsibleTest",
					GetDefaultSecretExtraMount())

				DeferCleanup(th.DeleteInstance, CreateAnsibleTest(ansibleTestName, spec))
			})

			It("should add secret based extra volume and volumeMount to the pod", func() {
				pod := GetTestOperatorPod(namespace, ansibleTestName.Name)
				ExpectPodHasSecretVolume(pod, ExtraSecretVolName, ExtraSecretName)
				ExpectPodHasVolumeMount(pod, ExtraSecretVolName, ExtraSecretMountPath)
			})
		})

		When("AnsibleTest is created with multiple extraMounts configmap and secret", func() {
			BeforeEach(func() {
				CreateExtraConfigMap(namespace, ExtraConfigMapName)
				CreateExtraSecret(namespace, ExtraSecretName)

				spec := GetDefaultAnsibleTestSpec()
				spec["extraMounts"] = BuildExtraMountsSpec("AnsibleTest",
					GetDefaultConfigMapExtraMount(),
					GetDefaultSecretExtraMount(),
				)

				DeferCleanup(th.DeleteInstance, CreateAnsibleTest(ansibleTestName, spec))
			})

			It("should add all extra volumes and volumeMounts to the pod", func() {
				pod := GetTestOperatorPod(namespace, ansibleTestName.Name)
				ExpectPodHasConfigMapVolume(pod, ExtraConfigVolName, ExtraConfigMapName)
				ExpectPodHasSecretVolume(pod, ExtraSecretVolName, ExtraSecretName)
				ExpectPodHasVolumeMount(pod, ExtraConfigVolName, ExtraConfigMountPath)
				ExpectPodHasVolumeMount(pod, ExtraSecretVolName, ExtraSecretMountPath)
			})
		})

		When("AnsibleTest is created with no propagation field", func() {
			BeforeEach(func() {
				CreateExtraConfigMap(namespace, ExtraConfigMapName)

				spec := GetDefaultAnsibleTestSpec()
				spec["extraMounts"] = BuildExtraMountsSpec("",
					GetDefaultConfigMapExtraMount())

				DeferCleanup(th.DeleteInstance, CreateAnsibleTest(ansibleTestName, spec))
			})

			It("should add extra volume and volumeMount when propagation is omitted", func() {
				pod := GetTestOperatorPod(namespace, ansibleTestName.Name)
				ExpectPodHasConfigMapVolume(pod, ExtraConfigVolName, ExtraConfigMapName)
				ExpectPodHasVolumeMount(pod, ExtraConfigVolName, ExtraConfigMountPath)
			})
		})

		When("AnsibleTest created with extraMounts is using the wrong propagation type", func() {
			BeforeEach(func() {
				CreateExtraConfigMap(namespace, ExtraConfigMapName)

				spec := GetDefaultAnsibleTestSpec()
				spec["extraMounts"] = BuildExtraMountsSpec("Tempest",
					GetDefaultConfigMapExtraMount())

				DeferCleanup(th.DeleteInstance, CreateAnsibleTest(ansibleTestName, spec))
			})

			It("should not add extra volume and volumeMount to the pod", func() {
				pod := GetTestOperatorPod(namespace, ansibleTestName.Name)
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

				DeferCleanup(th.DeleteInstance, CreateAnsibleTest(ansibleTestName, GetDefaultAnsibleTestWorkflowSpec()))
			})

			It("creates PVC with workflow step name", func() {
				pvc := GetTestOperatorPVC(namespace, ansibleTestName.Name)
				Expect(pvc.Name).To(ContainSubstring(ansibleTestName.Name + "-0-"))
			})

			It("creates pod with workflow step name", func() {
				pod := GetTestOperatorPod(namespace, ansibleTestName.Name)
				spec := GetDefaultAnsibleTestWorkflowSpec()
				workflow := spec["workflow"].([]map[string]any)
				stepName := workflow[0]["stepName"].(string)
				Expect(pod.Name).To(Equal(ansibleTestName.Name + "-s00-" + stepName))
			})
		})

		When("overrides spec defaults", func() {
			BeforeEach(func() {
				openstackConfigMap, openstackSecret := CreateCommonOpenstackResources(namespace)
				Expect(k8sClient.Create(ctx, openstackConfigMap)).Should(Succeed())
				Expect(k8sClient.Create(ctx, openstackSecret)).Should(Succeed())

				testOperatorConfigMap := CreateTestOperatorConfigMap(namespace)
				Expect(k8sClient.Create(ctx, testOperatorConfigMap)).Should(Succeed())

				DeferCleanup(th.DeleteInstance, CreateAnsibleTest(ansibleTestName, GetDefaultAnsibleTestWorkflowSpec()))
			})

			It("workflow values take precedence", func() {
				pod := GetTestOperatorPod(namespace, ansibleTestName.Name)
				spec := GetDefaultAnsibleTestWorkflowSpec()
				workflow := spec["workflow"].([]map[string]any)
				expectedRepo := workflow[0]["ansibleGitRepo"].(string)
				expectedPlaybook := workflow[0]["ansiblePlaybookPath"].(string)

				Expect(GetPodEnvVar(pod, "POD_ANSIBLE_GIT_REPO")).To(Equal(expectedRepo))
				Expect(GetPodEnvVar(pod, "POD_ANSIBLE_GIT_REPO")).NotTo(Equal(spec["ansibleGitRepo"].(string)))

				Expect(GetPodEnvVar(pod, "POD_ANSIBLE_PLAYBOOK")).To(Equal(expectedPlaybook))
				Expect(GetPodEnvVar(pod, "POD_ANSIBLE_PLAYBOOK")).NotTo(Equal(spec["ansiblePlaybookPath"].(string)))
			})
		})

		When("inherits from spec defaults", func() {
			BeforeEach(func() {
				openstackConfigMap, openstackSecret := CreateCommonOpenstackResources(namespace)
				Expect(k8sClient.Create(ctx, openstackConfigMap)).Should(Succeed())
				Expect(k8sClient.Create(ctx, openstackSecret)).Should(Succeed())

				testOperatorConfigMap := CreateTestOperatorConfigMap(namespace)
				Expect(k8sClient.Create(ctx, testOperatorConfigMap)).Should(Succeed())

				spec := GetDefaultAnsibleTestWorkflowSpec()
				workflow := spec["workflow"].([]map[string]any)
				delete(workflow[0], "ansibleGitRepo")
				delete(workflow[0], "ansiblePlaybookPath")

				DeferCleanup(th.DeleteInstance, CreateAnsibleTest(ansibleTestName, spec))
			})

			It("uses spec-level values when workflow step omits them", func() {
				pod := GetTestOperatorPod(namespace, ansibleTestName.Name)
				spec := GetDefaultAnsibleTestWorkflowSpec()
				expectedRepo := spec["ansibleGitRepo"].(string)
				expectedPlaybook := spec["ansiblePlaybookPath"].(string)

				Expect(GetPodEnvVar(pod, "POD_ANSIBLE_GIT_REPO")).To(Equal(expectedRepo))
				Expect(GetPodEnvVar(pod, "POD_ANSIBLE_PLAYBOOK")).To(Equal(expectedPlaybook))
			})
		})

		When("with multiple workflow steps", func() {
			BeforeEach(func() {
				openstackConfigMap, openstackSecret := CreateCommonOpenstackResources(namespace)
				Expect(k8sClient.Create(ctx, openstackConfigMap)).Should(Succeed())
				Expect(k8sClient.Create(ctx, openstackSecret)).Should(Succeed())

				testOperatorConfigMap := CreateTestOperatorConfigMap(namespace)
				Expect(k8sClient.Create(ctx, testOperatorConfigMap)).Should(Succeed())

				DeferCleanup(th.DeleteInstance, CreateAnsibleTest(ansibleTestName, GetDefaultAnsibleTestWorkflowSpec()))
			})

			It("creates second pod after first pod succeeds", func() {
				firstPod := GetTestOperatorPod(namespace, ansibleTestName.Name)
				Expect(firstPod.Name).To(Equal(ansibleTestName.Name + "-s00-first-step"))

				firstPod.Status.Phase = corev1.PodSucceeded
				Expect(k8sClient.Status().Update(ctx, firstPod)).Should(Succeed())

				Eventually(func(g Gomega) {
					podList := &corev1.PodList{}
					listOpts := []client.ListOption{
						client.InNamespace(namespace),
						client.MatchingLabels{
							"instanceName": ansibleTestName.Name,
							"operator":     "test-operator",
							"workflowStep": "1",
						},
					}
					g.Expect(k8sClient.List(ctx, podList, listOpts...)).Should(Succeed())
					g.Expect(podList.Items).To(HaveLen(1))
					secondPod := podList.Items[0]
					g.Expect(secondPod.Name).To(Equal(ansibleTestName.Name + "-s01-second-step"))

					spec := GetDefaultAnsibleTestWorkflowSpec()
					workflow := spec["workflow"].([]map[string]any)
					expectedRepo := workflow[1]["ansibleGitRepo"].(string)
					expectedPlaybook := workflow[1]["ansiblePlaybookPath"].(string)

					g.Expect(GetPodEnvVar(&secondPod, "POD_ANSIBLE_GIT_REPO")).To(Equal(expectedRepo))
					g.Expect(GetPodEnvVar(&secondPod, "POD_ANSIBLE_PLAYBOOK")).To(Equal(expectedPlaybook))
				}, timeout, interval).Should(Succeed())
			})
		})
	})

})
