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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// AnsibleTestSpec defines the desired state of AnsibleTest
type AnsibleTestSpec struct {
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:Optional
	// Extra configmaps for mounting in the pod.
	ExtraMounts []extraConfigmapsMounts `json:"extraMounts,omitempty"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:Optional
	// +kubebuilder:default="local-storage"
	// StorageClass used to create PVCs that store the logs
	StorageClass string `json:"storageClass"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:Required
	// +kubebuilder:default="dataplane-ansible-ssh-private-key-secret"
	// ComputeSSHKeySecretName is the name of the k8s secret that contains an ssh key for computes.
	// The key is mounted to ~/.ssh/id_ecdsa in the ansible pod
	ComputesSSHKeySecretName string `json:"computeSSHKeySecretName"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:Required
	// +kubebuilder:default=""
	// WorkloadSSHKeySecretName is the name of the k8s secret that contains an ssh key for the ansible workload.
	// The key is mounted to ~/test_keypair.key in the ansible pod
	WorkloadSSHKeySecretName string `json:"workloadSSHKeySecretName"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:Required
	// +kubebuilder:default=""
	// AnsibleGitRepo - git repo to clone into container
	AnsibleGitRepo string `json:"ansibleGitRepo"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:Required
	// +kubebuilder:default=""
	// AnsiblePlaybookPath - path to ansible playbook
	AnsiblePlaybookPath string `json:"ansiblePlaybookPath"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:optional
	// +kubebuilder:default=""
	// AnsibleCollections - extra ansible collections to instal in additionn to the ones exist in the requirements.yaml
	AnsibleCollections string `json:"ansibleCollections,omitempty"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:optional
	// +kubebuilder:default=""
	// AnsibleVarFiles - interface to create ansible var files Those get added to the
	AnsibleVarFiles string `json:"ansibleVarFiles,omitempty"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:optional
	// +kubebuilder:default=""
	// AnsibleExtraVars - string to pass parameters to ansible using
	AnsibleExtraVars string `json:"ansibleExtraVars,omitempty"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:optional
	// +kubebuilder:default=""
	// AnsibleInventory - string that contains the inventory file content
	AnsibleInventory string `json:"ansibleInventory,omitempty"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=openstack-config
	// OpenStackConfigMap is the name of the ConfigMap containing the clouds.yaml
	OpenStackConfigMap string `json:"openStackConfigMap"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=openstack-config-secret
	// OpenStackConfigSecret is the name of the Secret containing the secure.yaml
	OpenStackConfigSecret string `json:"openStackConfigSecret"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=false
	// Run ansible playbook with -vvvv
	Debug bool `json:"debug"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=""
	// Container image for AnsibleTest
	ContainerImage string `json:"containerImage"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// BackoffLimimt allows to define the maximum number of retried executions (defaults to 6).
	// +kubebuilder:default:=0
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:number"}
	BackoffLimit *int32 `json:"backoffLimit,omitempty"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// A parameter  that contains a workflow definition.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:number"}
	Workflow []AnsibleTestWorkflowSpec `json:"workflow,omitempty"`
}

type AnsibleTestWorkflowSpec struct {
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:Optional
	// Extra configmaps for mounting in the pod
	ExtraMounts []extraConfigmapsMounts `json:"extraMounts,omitempty"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:Required
	// Name of a workflow step. The step name will be used for example to create
	// a logs directory.
	StepName string `json:"stepName"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:Optional
	// StorageClass used to create PVCs that store the logs
	StorageClass *string `json:"storageClass,omitempty"`

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
	// OpenStackConfigMap is the name of the ConfigMap containing the clouds.yaml
	OpenStackConfigMap *string `json:"openStackConfigMap,omitempty"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:Optional
	// OpenStackConfigSecret is the name of the Secret containing the secure.yaml
	OpenStackConfigSecret *string `json:"openStackConfigSecret,omitempty"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:Optional
	// Run ansible playbook with -vvvv
	Debug bool `json:"debug,omitempty"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:Optional
	// Container image for AnsibleTest
	ContainerImage string `json:"containerImage,omitempty"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// BackoffLimimt allows to define the maximum number of retried executions (defaults to 6).
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:number"}
	BackoffLimit *int32 `json:"backoffLimit,omitempty"`
}

// AnsibleTestStatus defines the observed state of AnsibleTest
type AnsibleTestStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Map of hashes to track e.g. job status
	Hash map[string]string `json:"hash,omitempty"`

	// Conditions
	Conditions condition.Conditions `json:"conditions,omitempty" optional:"true"`

	// NetworkAttachments status of the deployment pods
	NetworkAttachments map[string][]string `json:"networkAttachments,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.conditions[0].status",description="Status"
//+kubebuilder:printcolumn:name="Message",type="string",JSONPath=".status.conditions[0].message",description="Message"

// AnsibleTestStatus is the Schema for the AnsibleTestStatus API
type AnsibleTest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AnsibleTestSpec   `json:"spec,omitempty"`
	Status AnsibleTestStatus `json:"status,omitempty"`
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
