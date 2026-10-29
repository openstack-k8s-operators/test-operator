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
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
				g.Expect(horizonTest.Status.Conditions).To(HaveLen(6))
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

		It("should have a finalizer", func() {
			Eventually(func() []string {
				return GetHorizonTest(horizonTestName).Finalizers
			}, timeout, interval).Should(ContainElement("openstack.org/horizontest"))
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

		It("should create RBAC resources for HorizonTest", func() {
			saName := types.NamespacedName{
				Name:      "horizontest-" + horizonTestName.Name,
				Namespace: namespace,
			}

			serviceAccount := th.GetServiceAccount(saName)
			Expect(serviceAccount.Name).To(Equal(saName.Name))

			roleName := types.NamespacedName{
				Name:      saName.Name + "-role",
				Namespace: namespace,
			}
			role := th.GetRole(roleName)
			Expect(role.Rules).To(Equal([]rbacv1.PolicyRule{{
				APIGroups:     []string{"security.openshift.io"},
				ResourceNames: []string{"nonroot-v2"},
				Resources:     []string{"securitycontextconstraints"},
				Verbs:         []string{"use"},
			}}))

			binding := th.GetRoleBinding(types.NamespacedName{
				Name:      saName.Name + "-rolebinding",
				Namespace: namespace,
			})
			Expect(binding.RoleRef).To(Equal(rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     "Role",
				Name:     roleName.Name,
			}))
			Expect(binding.Subjects).To(Equal([]rbacv1.Subject{{
				Kind:      "ServiceAccount",
				Name:      saName.Name,
				Namespace: namespace,
			}}))
			ownerUID := GetHorizonTest(horizonTestName).UID
			for _, resource := range []client.Object{serviceAccount, role, binding} {
				Expect(resource.GetOwnerReferences()).To(HaveLen(1))
				Expect(resource.GetOwnerReferences()[0].UID).To(Equal(ownerUID))
			}

			pod := GetTestOperatorPod(namespace, horizonTestName.Name)
			Expect(pod.Spec.ServiceAccountName).To(Equal(saName.Name))

			for _, conditionType := range []condition.Type{
				condition.ServiceAccountReadyCondition,
				condition.RoleReadyCondition,
				condition.RoleBindingReadyCondition,
			} {
				th.ExpectCondition(
					horizonTestName,
					ConditionGetterFunc(HorizonTestConditionGetter),
					conditionType,
					corev1.ConditionTrue,
				)
			}
		})

		It("should create a pod", func() {
			pod := GetTestOperatorPod(namespace, horizonTestName.Name)
			Expect(pod.Name).ToNot(BeEmpty())
		})
	})

	DescribeTable("Pod SCC request leaves its service account on nonroot-v2",
		func(privileged bool, expectedSCC string, finalPhase corev1.PodPhase) {
			openstackConfigMap, openstackSecret := CreateCommonOpenstackResources(namespace)
			Expect(k8sClient.Create(ctx, openstackConfigMap)).Should(Succeed())
			Expect(k8sClient.Create(ctx, openstackSecret)).Should(Succeed())
			Expect(k8sClient.Create(ctx, CreateTestOperatorConfigMap(namespace))).Should(Succeed())

			spec := GetDefaultHorizonTestSpec()
			spec["privileged"] = privileged
			DeferCleanup(th.DeleteInstance, CreateHorizonTest(horizonTestName, spec))

			roleName := types.NamespacedName{
				Name:      "horizontest-" + horizonTestName.Name + "-role",
				Namespace: namespace,
			}
			role := th.GetRole(roleName)
			Expect(role.Rules).To(Equal([]rbacv1.PolicyRule{{
				APIGroups:     []string{"security.openshift.io"},
				ResourceNames: []string{"nonroot-v2"},
				Resources:     []string{"securitycontextconstraints"},
				Verbs:         []string{"use"},
			}}))

			pod := GetTestOperatorPod(namespace, horizonTestName.Name)
			Expect(pod.Spec.ServiceAccountName).To(Equal("horizontest-" + horizonTestName.Name))
			Expect(pod.Annotations).To(HaveKeyWithValue("openshift.io/required-scc", expectedSCC))

			if finalPhase != "" {
				pod.Status.Phase = finalPhase
				Expect(k8sClient.Status().Update(ctx, pod)).Should(Succeed())
				Eventually(func(g Gomega) {
					g.Expect(k8sClient.Get(ctx, roleName, role)).Should(Succeed())
					g.Expect(role.Rules).To(Equal([]rbacv1.PolicyRule{{
						APIGroups:     []string{"security.openshift.io"},
						ResourceNames: []string{"nonroot-v2"},
						Resources:     []string{"securitycontextconstraints"},
						Verbs:         []string{"use"},
					}}))
				}, timeout*10, interval).Should(Succeed())
				th.GetServiceAccount(types.NamespacedName{Name: pod.Spec.ServiceAccountName, Namespace: namespace})
				th.GetRoleBinding(types.NamespacedName{Name: pod.Spec.ServiceAccountName + "-rolebinding", Namespace: namespace})
				Eventually(func() bool {
					return GetHorizonTest(horizonTestName).Status.Conditions.IsTrue(condition.DeploymentReadyCondition)
				}, timeout*10, interval).Should(BeTrue())
			}
		},
		Entry("without privileged access", false, "nonroot-v2", corev1.PodPhase("")),
		Entry("after a successful privileged Pod", true, "privileged", corev1.PodSucceeded),
		Entry("after a failed privileged Pod", true, "privileged", corev1.PodFailed),
	)

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
