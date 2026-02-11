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
			// the reconciler loop adds the finalizer so we have to wait for
			// it to run
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
})
