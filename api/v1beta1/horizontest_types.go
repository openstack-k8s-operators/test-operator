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

// HorizonTestSpec defines the desired state of HorizonTest
type HorizonTestSpec struct {
	// +kubebuilder:validation:Required
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:default="local-storage"
	// StorageClass used to create PVCs that store the logs
	StorageClass string `json:"storageClass"`

	// AdminUsername is the username for the OpenStack admin user.
	// +kubebuilder:validation:Required
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:default="admin"
	AdminUsername string `json:"adminUsername,omitempty"`

	// AdminPassword is the password for the OpenStack admin user.
	// +kubebuilder:validation:Required
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:default="admin"
	AdminPassword string `json:"adminPassword,omitempty"`

	// DashboardUrl is the URL of the Horizon dashboard.
	// +kubebuilder:validation:Required
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	DashboardUrl string `json:"dashboardUrl,omitempty"`

	// AuthUrl is the authentication URL for OpenStack.
	// +kubebuilder:validation:Required
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	AuthUrl string `json:"authUrl,omitempty"`

	// RepoUrl is the URL of the Horizon repository.
	// +kubebuilder:validation:Required
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:default="https://review.opendev.org/openstack/horizon"
	RepoUrl string `json:"repoUrl,omitempty"`

	// HorizonRepoBranch is the branch of the Horizon repository to checkout.
	// +kubebuilder:validation:Required
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:default="master"
	HorizonRepoBranch string `json:"horizonRepoBranch,omitempty"`

	// ImageUrl is the URL to download the Cirros image.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:default="http://download.cirros-cloud.net/0.6.2/cirros-0.6.2-x86_64-disk.img"
	ImageUrl string `json:"imageUrl,omitempty"`

	// ProjectName is the name of the OpenStack project for Horizon tests.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:default="horizontest"
	ProjectName string `json:"projectName,omitempty"`

	// User is the username under which the Horizon tests will run.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:default="horizontest"
	User string `json:"user,omitempty"`

	// Password is the password for the user running the Horizon tests.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:default="horizontest"
	Password string `json:"password,omitempty"`

	// FlavorName is the name of the OpenStack flavor to create for Horizon tests.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:default="m1.tiny"
	FlavorName string `json:"flavorName,omitempty"`

	// LogsDirectoryName is the name of the directory to store test logs.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:default="horizon"
	LogsDirectoryName string `json:"logsDirectoryName,omitempty"`

	// HorizonTestDir is the directory path for Horizon tests.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:default="/var/lib/horizontest"
	HorizonTestDir string `json:"horizonTestDir,omitempty"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:default:=""
	// Container image for horizontest
	ContainerImage string `json:"containerImage,omitempty"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:default:=false
	// Parallel
	Parallel bool `json:"parallel,omitempty"`

	// BackoffLimimt allows to define the maximum number of retried executions.
	// +kubebuilder:default:=0
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:number"}
	BackoffLimit *int32 `json:"backoffLimit,omitempty"`

	// Name of a secret that contains a kubeconfig. The kubeconfig is mounted under /var/lib/horizontest/.kube/config
	// in the test pod.
	// +kubebuilder:default:=""
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:number"}
	KubeconfigSecretName string `json:"kubeconfigSecretName,omitempty"`
}

// HorizonTestStatus defines the observed state of HorizonTest
type HorizonTestStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Map of hashes to track e.g. job status
	Hash map[string]string `json:"hash,omitempty"`

	// Conditions
	Conditions condition.Conditions `json:"conditions,omitempty" optional:"true"`

	// NetworkAttachments status of the deployment pods
	NetworkAttachments map[string][]string `json:"networkAttachments,omitempty"`

}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// HorizonTest is the Schema for the horizontests API
type HorizonTest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HorizonTestSpec   `json:"spec,omitempty"`
	Status HorizonTestStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// HorizonTestList contains a list of HorizonTest
type HorizonTestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HorizonTest `json:"items"`
}

func init() {
	SchemeBuilder.Register(&HorizonTest{}, &HorizonTestList{})
}

// RbacConditionsSet - set the conditions for the rbac object
func (instance HorizonTest) RbacConditionsSet(c *condition.Condition) {
	instance.Status.Conditions.Set(c)
}

// RbacNamespace - return the namespace
func (instance HorizonTest) RbacNamespace() string {
	return instance.Namespace
}

// RbacResourceName - return the name to be used for rbac objects (serviceaccount, role, rolebinding)
func (instance HorizonTest) RbacResourceName() string {
	return instance.Name
}
