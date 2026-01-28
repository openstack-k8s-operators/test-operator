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

type PatchType struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Format=uri
	// URL of the Tobiko repository that a patch will be applied to.
	Repository string `json:"repository,omitempty"`

	// +kubebuilder:validation:Optional
	// Refspec specifies which change the remote repository should be
	// checked out to.
	Refspec string `json:"refspec,omitempty"`
}

// TobikoSpec defines the desired state of Tobiko
type TobikoSpec struct {
	CommonOptions         `json:",inline"`
	CommonOpenstackConfig `json:",inline"`

	// +kubebuilder:default:={limits: {cpu: "8000m", memory: "8Gi"}, requests: {cpu: "4000m", memory: "4Gi"}}
	// The desired amount of resources that should be assigned to each test pod
	// spawned using the Tobiko CR. https://pkg.go.dev/k8s.io/api/core/v1#ResourceRequirements
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// Activate debug mode. When debug mode is activated any error encountered
	// inside the test-pod causes that the pod will be kept alive indefinitely
	// (stuck in "Running" phase) or until the corresponding Tobiko CR is deleted.
	// This allows the user to debug any potential troubles with `oc rsh`.
	Debug bool `json:"debug"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:default:="py3"
	// Test environment
	Testenv string `json:"testenv"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:default:=""
	// String including any options to pass to pytest when it runs tobiko tests
	PytestAddopts string `json:"pytestAddopts"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:default:=false
	// Boolean specifying whether tobiko tests create new resources or re-use those previously created
	PreventCreate bool `json:"preventCreate"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:default:=4
	// Number of processes/workers used to run tobiko tests - value 0 results in automatic decision
	NumProcesses uint8 `json:"numProcesses"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:default:=""
	// Tobiko version
	Version string `json:"version"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// Optional patch to apply to the Tobiko repository.
	Patch PatchType `json:"patch"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:default:=""
	// tobiko.conf
	Config string `json:"config"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:default:=""
	// Private Key
	PrivateKey string `json:"privateKey"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:default:=""
	// Public Key
	PublicKey string `json:"publicKey"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:default:=false
	// By default test-operator executes the test-pods sequentially if multiple
	// instances of test-operator related CRs exist. To run test-pods in parallel
	// set this option to true.
	Parallel bool `json:"parallel"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MaxLength=253
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:number"}
	// Name of a secret that contains a kubeconfig. The kubeconfig is mounted under /var/lib/tobiko/.kube/config
	// in the test pod.
	KubeconfigSecretName string `json:"kubeconfigSecretName,omitempty"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// NetworkAttachments is a list of NetworkAttachment resource names to expose
	// the services to the given network
	NetworkAttachments []string `json:"networkAttachments,omitempty"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:number"}
	// A parameter that contains a workflow definition.
	Workflow []TobikoWorkflowSpec `json:"workflow,omitempty"`
}

type TobikoWorkflowSpec struct {
	WorkflowCommonOptions `json:",inline"`
	CommonOpenstackConfig `json:",inline"`

	// The desired amount of resources that should be assigned to each test pod
	// spawned using the Tobiko CR. https://pkg.go.dev/k8s.io/api/core/v1#ResourceRequirements
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// Test environment
	Testenv string `json:"testenv,omitempty"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// String including any options to pass to pytest when it runs tobiko tests
	PytestAddopts string `json:"pytestAddopts,omitempty"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// Boolean specifying whether tobiko tests create new resources or re-use those previously created
	PreventCreate *bool `json:"preventCreate,omitempty"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// Number of processes/workers used to run tobiko tests - value 0 results in automatic decision
	NumProcesses *uint8 `json:"numProcesses,omitempty"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// NetworkAttachments is a list of NetworkAttachment resource names to expose
	// the services to the given network
	NetworkAttachments []string `json:"networkAttachments,omitempty"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// Tobiko version
	Version string `json:"version,omitempty"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// Optional patch to apply to the Tobiko repository for this step.
	Patch *PatchType `json:"patch,omitempty"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// tobiko.conf
	Config string `json:"config,omitempty"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// Private Key
	PrivateKey string `json:"privateKey,omitempty"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// Public Key
	PublicKey string `json:"publicKey,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MaxLength=253
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:number"}
	// Name of a secret that contains a kubeconfig. The kubeconfig is mounted under /var/lib/tobiko/.kube/config
	// in the test pod.
	KubeconfigSecretName string `json:"kubeconfigSecretName,omitempty"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern:=^[a-z0-9-]+$
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:number"}
	// A parameter that contains a definition of a single workflow step.
	StepName string `json:"stepName"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.conditions[0].status",description="Status"
//+kubebuilder:printcolumn:name="Message",type="string",JSONPath=".status.conditions[0].message",description="Message"

type Tobiko struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TobikoSpec       `json:"spec,omitempty"`
	Status CommonTestStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// TobikoList contains a list of Tobiko
type TobikoList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Tobiko `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Tobiko{}, &TobikoList{})
}

// RbacConditionsSet - set the conditions for the rbac object
func (instance Tobiko) RbacConditionsSet(c *condition.Condition) {
	instance.Status.Conditions.Set(c)
}

// RbacNamespace - return the namespace
func (instance Tobiko) RbacNamespace() string {
	return instance.Namespace
}

// RbacResourceName - return the name to be used for rbac objects (serviceaccount, role, rolebinding)
func (instance Tobiko) RbacResourceName() string {
	return instance.Name
}
