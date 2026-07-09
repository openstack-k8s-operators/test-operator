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
)

var _ = Describe("HorizonTest controller", func() {
	var horizonTestName types.NamespacedName

	BeforeEach(func() {
		horizonTestName = types.NamespacedName{
			Name:      "horizontest",
			Namespace: namespace,
		}
	})

	DescribeTable("Missing Openstack resources should set InputReady to false",
		func(createResource func()) {
			createResource()
			DeferCleanup(th.DeleteInstance, CreateHorizonTest(horizonTestName, GetDefaultHorizonTestSpec()))

			th.ExpectCondition(
				horizonTestName,
				ConditionGetterFunc(HorizonTestConditionGetter),
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

	When("A HorizonTest instance is created", func() {
		BeforeEach(func() {
			openstackConfigMap, openstackSecret := CreateCommonOpenstackResources(namespace)
			Expect(k8sClient.Create(ctx, openstackConfigMap)).Should(Succeed())
			Expect(k8sClient.Create(ctx, openstackSecret)).Should(Succeed())
			DeferCleanup(th.DeleteInstance, CreateHorizonTest(horizonTestName, GetDefaultHorizonTestSpec()))
		})

		It("initializes the status fields", func() {
			Eventually(func(g Gomega) {
				horizonTest := GetHorizonTest(horizonTestName)
				g.Expect(horizonTest.Status.Conditions).To(HaveLen(4))
				g.Expect(horizonTest.Status.Hash).To(BeEmpty())
			}, timeout*2, interval).Should(Succeed())
		})

		It("should have the Spec fields initialized", func() {
			horizonTest := GetHorizonTest(horizonTestName)
			Expect(horizonTest.Spec.StorageClass).Should(Equal(DefaultStorageClass))
			Expect(horizonTest.Spec.AdminUsername).Should(Equal("admin"))
			Expect(horizonTest.Spec.AdminPassword).Should(Equal("password"))
			Expect(horizonTest.Spec.DashboardUrl).ShouldNot(BeEmpty())
			Expect(horizonTest.Spec.AuthUrl).ShouldNot(BeEmpty())
		})
	})

	When("All dependencies are ready", func() {
		BeforeEach(func() {
			openstackConfigMap, openstackSecret := CreateCommonOpenstackResources(namespace)
			Expect(k8sClient.Create(ctx, openstackConfigMap)).Should(Succeed())
			Expect(k8sClient.Create(ctx, openstackSecret)).Should(Succeed())

			testOperatorConfigMap := CreateTestOperatorConfigMap(namespace)
			Expect(k8sClient.Create(ctx, testOperatorConfigMap)).Should(Succeed())

			DeferCleanup(th.DeleteInstance, CreateHorizonTest(horizonTestName, GetDefaultHorizonTestSpec()))
		})

		It("should have InputReady condition true", func() {
			th.ExpectCondition(
				horizonTestName,
				ConditionGetterFunc(HorizonTestConditionGetter),
				condition.InputReadyCondition,
				corev1.ConditionTrue,
			)
		})

		It("should create a PVC for logs", func() {
			pvc := GetTestOperatorPVC(namespace, horizonTestName.Name)
			Expect(pvc.Name).ToNot(BeEmpty())
			Expect(*pvc.Spec.StorageClassName).To(Equal(DefaultStorageClass))
			Expect(pvc.Spec.AccessModes).To(ContainElement(corev1.ReadWriteOnce))
		})

		It("should create a pod", func() {
			pod := GetTestOperatorPod(namespace, horizonTestName.Name)
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

		When("HorizonTest is created with extraMounts", func() {
			BeforeEach(func() {
				CreateExtraConfigMap(namespace, ExtraConfigMapName)

				spec := GetDefaultHorizonTestSpec()
				spec["extraMounts"] = BuildExtraMountsSpec("HorizonTest",
					GetDefaultConfigMapExtraMount())

				DeferCleanup(th.DeleteInstance, CreateHorizonTest(horizonTestName, spec))
			})

			It("should add extra volume and volumeMount to the pod", func() {
				pod := GetTestOperatorPod(namespace, horizonTestName.Name)
				ExpectPodHasConfigMapVolume(pod, ExtraConfigVolName, ExtraConfigMapName)
				ExpectPodHasVolumeMount(pod, ExtraConfigVolName, ExtraConfigMountPath)
			})
		})

		When("HorizonTest is created with Secret as the source of extraMount", func() {
			BeforeEach(func() {
				CreateExtraSecret(namespace, ExtraSecretName)

				spec := GetDefaultHorizonTestSpec()
				spec["extraMounts"] = BuildExtraMountsSpec("HorizonTest",
					GetDefaultSecretExtraMount())

				DeferCleanup(th.DeleteInstance, CreateHorizonTest(horizonTestName, spec))
			})

			It("should add secret based extra volume and volumeMount to the pod", func() {
				pod := GetTestOperatorPod(namespace, horizonTestName.Name)
				ExpectPodHasSecretVolume(pod, ExtraSecretVolName, ExtraSecretName)
				ExpectPodHasVolumeMount(pod, ExtraSecretVolName, ExtraSecretMountPath)
			})
		})

		When("HorizonTest is created with multiple extraMounts configmap and secret", func() {
			BeforeEach(func() {
				CreateExtraConfigMap(namespace, ExtraConfigMapName)
				CreateExtraSecret(namespace, ExtraSecretName)

				spec := GetDefaultHorizonTestSpec()
				spec["extraMounts"] = BuildExtraMountsSpec("HorizonTest",
					GetDefaultConfigMapExtraMount(),
					GetDefaultSecretExtraMount(),
				)

				DeferCleanup(th.DeleteInstance, CreateHorizonTest(horizonTestName, spec))
			})

			It("should add all extra volumes and volumeMounts to the pod", func() {
				pod := GetTestOperatorPod(namespace, horizonTestName.Name)
				ExpectPodHasConfigMapVolume(pod, ExtraConfigVolName, ExtraConfigMapName)
				ExpectPodHasSecretVolume(pod, ExtraSecretVolName, ExtraSecretName)
				ExpectPodHasVolumeMount(pod, ExtraConfigVolName, ExtraConfigMountPath)
				ExpectPodHasVolumeMount(pod, ExtraSecretVolName, ExtraSecretMountPath)
			})
		})

		When("HorizonTest is created with no propagation field", func() {
			BeforeEach(func() {
				CreateExtraConfigMap(namespace, ExtraConfigMapName)

				spec := GetDefaultHorizonTestSpec()
				spec["extraMounts"] = BuildExtraMountsSpec("",
					GetDefaultConfigMapExtraMount())

				DeferCleanup(th.DeleteInstance, CreateHorizonTest(horizonTestName, spec))
			})

			It("should add extra volume and volumeMount when propagation is omitted", func() {
				pod := GetTestOperatorPod(namespace, horizonTestName.Name)
				ExpectPodHasConfigMapVolume(pod, ExtraConfigVolName, ExtraConfigMapName)
				ExpectPodHasVolumeMount(pod, ExtraConfigVolName, ExtraConfigMountPath)
			})
		})

		When("HorizonTest created with extraMounts is using the wrong propagation type", func() {
			BeforeEach(func() {
				CreateExtraConfigMap(namespace, ExtraConfigMapName)

				spec := GetDefaultHorizonTestSpec()
				spec["extraMounts"] = BuildExtraMountsSpec("Tempest",
					GetDefaultConfigMapExtraMount())

				DeferCleanup(th.DeleteInstance, CreateHorizonTest(horizonTestName, spec))
			})

			It("should not add extra volume and volumeMount to the pod", func() {
				pod := GetTestOperatorPod(namespace, horizonTestName.Name)
				ExpectPodNotHasVolume(pod, ExtraConfigVolName)
				ExpectPodNotHasVolumeMount(pod, ExtraConfigVolName)
			})
		})
	})

})
