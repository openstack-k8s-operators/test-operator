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

// AnsibleTestSpec defines the desired state of AnsibleTest
type AnsibleTestSpec struct {
	CommonOptions         `json:",inline"`
	CommonOpenstackConfig `json:",inline"`

	// +kubebuilder:default:={limits: {cpu: "4000m", memory: "4Gi"}, requests: {cpu: "2000m", memory: "2Gi"}}
	// The desired amount of resources that should be assigned to each test pod
	// spawned using the AnsibleTest CR. https://pkg.go.dev/k8s.io/api/core/v1#ResourceRequirements
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MaxLength=253
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:default:="dataplane-ansible-ssh-private-key-secret"
	// ComputeSSHKeySecretName is the name of the k8s secret that contains an ssh key for computes.
	// The key is mounted to ~/.ssh/id_ecdsa in the ansible pod
	ComputeSSHKeySecretName string `json:"computeSSHKeySecretName"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MaxLength=253
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:default:=""
	// WorkloadSSHKeySecretName is the name of the k8s secret that contains an ssh key for the ansible workload.
	// The key is mounted to ~/test_keypair.key in the ansible pod
	WorkloadSSHKeySecretName string `json:"workloadSSHKeySecretName"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Format=uri
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// AnsibleGitRepo - git repo to clone into container
	AnsibleGitRepo string `json:"ansibleGitRepo"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Format=string
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// AnsibleGitBranch - git branch to check out in the cloned repo
	AnsibleGitBranch string `json:"ansibleGitBranch,omitempty"`

	// +kubebuilder:validation:Required
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:default:=""
	// AnsiblePlaybookPath - path to ansible playbook
	AnsiblePlaybookPath string `json:"ansiblePlaybookPath"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:default:=""
	// AnsibleCollections - extra ansible collections to install in addition
	// to the ones existing in the requirements.yaml
	AnsibleCollections string `json:"ansibleCollections"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:default:=""
	// AnsibleVarFiles - interface to create ansible var files. Those get added
	// to the service config dir in /etc/test_operator/<file> and passed to the
	// ansible command using -e @/etc/test_operator/<file>
	AnsibleVarFiles string `json:"ansibleVarFiles"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:default:=""
	// AnsibleExtraVars - string to pass parameters to ansible
	AnsibleExtraVars string `json:"ansibleExtraVars"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:default:=""
	// AnsibleInventory - string that contains the inventory file content
	AnsibleInventory string `json:"ansibleInventory"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:default:=false
	// Run ansible playbook with -vvvv
	Debug bool `json:"debug"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:number"}
	// A parameter that contains a workflow definition.
	Workflow []AnsibleTestWorkflowSpec `json:"workflow,omitempty"`
}

type AnsibleTestWorkflowSpec struct {
	WorkflowCommonOptions `json:",inline"`
	CommonOpenstackConfig `json:",inline"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern:=^[a-z0-9-]+$
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// Name of a workflow step. The step name will be used for example to create
	// a logs directory.
	StepName string `json:"stepName"`

	// The desired amount of resources that should be assigned to each test pod
	// spawned using the AnsibleTest CR. https://pkg.go.dev/k8s.io/api/core/v1#ResourceRequirements
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MaxLength=253
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// ComputeSSHKeySecretName is the name of the k8s secret that contains an ssh key for computes.
	// The key is mounted to ~/.ssh/id_ecdsa in the ansible pod
	ComputeSSHKeySecretName string `json:"computeSSHKeySecretName"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MaxLength=253
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// WorkloadSSHKeySecretName is the name of the k8s secret that contains an ssh key for the ansible workload.
	// The key is mounted to ~/test_keypair.key in the ansible pod
	WorkloadSSHKeySecretName string `json:"workloadSSHKeySecretName"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Format=uri
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// AnsibleGitRepo - git repo to clone into container
	AnsibleGitRepo string `json:"ansibleGitRepo,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Format=string
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// AnsibleGitBranch - git branch to check out in the cloned repo
	AnsibleGitBranch string `json:"ansibleGitBranch,omitempty"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// AnsiblePlaybookPath - path to ansible playbook
	AnsiblePlaybookPath string `json:"ansiblePlaybookPath,omitempty"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// AnsibleCollections - extra ansible collections to install in addition
	// to the ones existing in the requirements.yaml
	AnsibleCollections string `json:"ansibleCollections,omitempty"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// AnsibleVarFiles - interface to create ansible var files. Those get added
	// to the service config dir in /etc/test_operator/<file> and passed to the
	// ansible command using -e @/etc/test_operator/<file>
	AnsibleVarFiles string `json:"ansibleVarFiles,omitempty"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// AnsibleExtraVars - interface to pass parameters to ansible using -e
	AnsibleExtraVars string `json:"ansibleExtraVars,omitempty"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// AnsibleInventory - string that contains the inventory file content
	AnsibleInventory string `json:"ansibleInventory,omitempty"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
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

// GetConditions - return the conditions from the status
func (instance *AnsibleTest) GetConditions() *condition.Conditions {
	return &instance.Status.Conditions
}
