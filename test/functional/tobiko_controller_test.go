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
			}, timeout*2, interval).Should(Succeed())
		})

		It("is not ready", func() {
			th.ExpectCondition(
				tobikoName,
				ConditionGetterFunc(TobikoConditionGetter),
				condition.ReadyCondition,
				corev1.ConditionUnknown,
			)
		})
	})
})
