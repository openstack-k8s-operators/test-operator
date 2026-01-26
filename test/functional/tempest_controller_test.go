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

var _ = Describe("Tempest controller", func() {
	var tempestName types.NamespacedName

	BeforeEach(func() {
		tempestName = types.NamespacedName{
			Name:      "tempest",
			Namespace: namespace,
		}
	})

	When("A Tempest intance is created", func() {
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
			}, timeout*2, interval).Should(Succeed())
		})

		It("is not ready", func() {
			th.ExpectCondition(
				tempestName,
				ConditionGetterFunc(TempestConditionGetter),
				condition.ReadyCondition,
				corev1.ConditionUnknown,
			)
		})

		It("should have a finalizer", func() {
			// the reconciler loop adds the finalizer so we have to wait for
			// it to run
			Eventually(func() []string {
				return GetTempest(tempestName).Finalizers
			}, timeout, interval).Should(ContainElement("openstack.org/tempest"))
		})
	})
})
