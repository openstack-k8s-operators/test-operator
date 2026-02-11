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
				g.Expect(tobiko.Status.Conditions).To(HaveLen(4))
				g.Expect(tobiko.Status.Hash).To(BeEmpty())
				g.Expect(tobiko.Status.NetworkAttachments).To(BeEmpty())
			}, timeout*2, interval).Should(Succeed())
		})

		It("should have the Spec fields initialized", func() {
			tobiko := GetTobiko(tobikoName)
			Expect(tobiko.Spec.StorageClass).Should(Equal(DefaultStorageClass))
			Expect(tobiko.Spec.Testenv).Should(Equal("sanity"))
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
})
