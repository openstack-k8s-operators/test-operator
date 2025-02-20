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

package v1beta1

import (
	"github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// AnsibleTestSpec defines the desired state of AnsibleTest
type AnsibleTestSpec struct {
	CommonOptions         `json:",inline"`
	CommonOpenstackConfig `json:",inline"`

	// +kubebuilder:default:={limits: {cpu: "4000m", memory: "4Gi"}, requests: {cpu: "2000m", memory: "2Gi"}}
	// The desired amount of resources that should be assigned to each test pod
	// spawned using the AnsibleTest CR. https://pkg.go.dev/k8s.io/api/core/v1#ResourceRequirements
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:="dataplane-ansible-ssh-private-key-secret"
	// ComputeSSHKeySecretName is the name of the k8s secret that contains an ssh key for computes.
	// The key is mounted to ~/.ssh/id_ecdsa in the ansible pod
	ComputesSSHKeySecretName string `json:"computeSSHKeySecretName"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=""
	// WorkloadSSHKeySecretName is the name of the k8s secret that contains an ssh key for the ansible workload.
	// The key is mounted to ~/test_keypair.key in the ansible pod
	WorkloadSSHKeySecretName string `json:"workloadSSHKeySecretName"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:Required
	// +kubebuilder:default:=""
	// AnsibleGitRepo - git repo to clone into container
	AnsibleGitRepo string `json:"ansibleGitRepo"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:Required
	// +kubebuilder:default:=""
	// AnsiblePlaybookPath - path to ansible playbook
	AnsiblePlaybookPath string `json:"ansiblePlaybookPath"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:optional
	// +kubebuilder:default:=""
	// AnsibleCollections - extra ansible collections to instal in additionn to the ones exist in the requirements.yaml
	AnsibleCollections string `json:"ansibleCollections,omitempty"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:optional
	// +kubebuilder:default:=""
	// AnsibleVarFiles - interface to create ansible var files Those get added to the
	AnsibleVarFiles string `json:"ansibleVarFiles,omitempty"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:optional
	// +kubebuilder:default:=""
	// AnsibleExtraVars - string to pass parameters to ansible using
	AnsibleExtraVars string `json:"ansibleExtraVars,omitempty"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:optional
	// +kubebuilder:default:=""
	// AnsibleInventory - string that contains the inventory file content
	AnsibleInventory string `json:"ansibleInventory,omitempty"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=false
	// Run ansible playbook with -vvvv
	Debug bool `json:"debug"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// A parameter that contains a workflow definition.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:number"}
	Workflow []AnsibleTestWorkflowSpec `json:"workflow,omitempty"`
}

type AnsibleTestWorkflowSpec struct {
	WorkflowCommonParameters `json:",inline"`
	CommonOpenstackConfig    `json:",inline"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MaxLength:=100
	// Name of a workflow step. The step name will be used for example to create
	// a logs directory.
	StepName string `json:"stepName"`

	// The desired amount of resources that should be assigned to each test pod
	// spawned using the AnsibleTest CR. https://pkg.go.dev/k8s.io/api/core/v1#ResourceRequirements
	// +kubebuilder:default:={limits: {cpu: "2000m", memory: "2Gi"}, requests: {cpu: "1000m", memory: "2Gi"}}
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:Optional
	// ComputeSSHKeySecretName is the name of the k8s secret that contains an ssh key for computes.
	// The key is mounted to ~/.ssh/id_ecdsa in the ansible pod
	ComputesSSHKeySecretName string `json:"computeSSHKeySecretName"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:Optional
	// WorkloadSSHKeySecretName is the name of the k8s secret that contains an ssh key for the ansible workload.
	// The key is mounted to ~/test_keypair.key in the ansible pod
	WorkloadSSHKeySecretName string `json:"workloadSSHKeySecretName"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:Optional
	// AnsibleGitRepo - git repo to clone into container
	AnsibleGitRepo string `json:"ansibleGitRepo,omitempty"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:Optional
	// AnsiblePlaybookPath - path to ansible playbook
	AnsiblePlaybookPath string `json:"ansiblePlaybookPath,omitempty"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:optional
	// AnsibleCollections - extra ansible collections to instal in additionn to the ones exist in the requirements.yaml
	AnsibleCollections string `json:"ansibleCollections,omitempty"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:optional
	// AnsibleVarFiles - interface to create ansible var files Those get added to the
	// service config dir in /etc/test_operator/<file> and passed to the ansible command using -e @/etc/test_operator/<file>
	AnsibleVarFiles string `json:"ansibleVarFiles,omitempty"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:optional
	// AnsibleExtraVars - interface to pass parameters to ansible using -e
	AnsibleExtraVars string `json:"ansibleExtraVars,omitempty"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:optional
	// AnsibleInventory - string that contains the inventory file content
	AnsibleInventory string `json:"ansibleInventory,omitempty"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:Optional
	// Run ansible playbook with -vvvv
	Debug bool `json:"debug,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.conditions[0].status",description="Status"
//+kubebuilder:printcolumn:name="Message",type="string",JSONPath=".status.conditions[0].message",description="Message"

type AnsibleTest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AnsibleTestSpec  `json:"spec,omitempty"`
	Status CommonTestStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// AnsibleTestList contains a list of AnsibleTest
type AnsibleTestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AnsibleTest `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AnsibleTest{}, &AnsibleTestList{})
}

// RbacConditionsSet - set the conditions for the rbac object
func (instance AnsibleTest) RbacConditionsSet(c *condition.Condition) {
	instance.Status.Conditions.Set(c)
}

// RbacNamespace - return the namespace
func (instance AnsibleTest) RbacNamespace() string {
	return instance.Namespace
}

// RbacResourceName - return the name to be used for rbac objects (serviceaccount, role, rolebinding)
func (instance AnsibleTest) RbacResourceName() string {
	return instance.Name
}
