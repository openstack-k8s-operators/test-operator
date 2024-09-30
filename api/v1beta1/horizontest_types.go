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
	CommonOptions `json:",inline"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// Activate debug mode. When debug mode is activated any error encountered
	// inside the test-pod causes that the pod will be kept alive indefinitely
	// (stuck in "Running" phase) or until the corresponding HorizonTest CR is deleted.
	// This allows the user to debug any potential troubles with `oc rsh`.
	Debug bool `json:"debug"`

	// AdminUsername is the username for the OpenStack admin user.
	// +kubebuilder:validation:Required
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:default:="admin"
	AdminUsername string `json:"adminUsername"`

	// AdminPassword is the password for the OpenStack admin user.
	// +kubebuilder:validation:Required
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:default:="admin"
	AdminPassword string `json:"adminPassword"`

	// DashboardUrl is the URL of the Horizon dashboard.
	// +kubebuilder:validation:Required
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	DashboardUrl string `json:"dashboardUrl"`

	// AuthUrl is the authentication URL for OpenStack.
	// +kubebuilder:validation:Required
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	AuthUrl string `json:"authUrl"`

	// RepoUrl is the URL of the Horizon repository.
	// +kubebuilder:validation:Required
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:default:="https://review.opendev.org/openstack/horizon"
	RepoUrl string `json:"repoUrl"`

	// HorizonRepoBranch is the branch of the Horizon repository to checkout.
	// +kubebuilder:validation:Required
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:default:="master"
	HorizonRepoBranch string `json:"horizonRepoBranch"`

	// ImageUrl is the URL to download the Cirros image.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:default:="http://download.cirros-cloud.net/0.6.2/cirros-0.6.2-x86_64-disk.img"
	ImageUrl string `json:"imageUrl"`

	// ProjectName is the name of the OpenStack project for Horizon tests.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:default:="horizontest"
	ProjectName string `json:"projectName"`

	// User is the username under which the Horizon tests will run.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:default:="horizontest"
	User string `json:"user"`

	// Password is the password for the user running the Horizon tests.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:default:="horizontest"
	Password string `json:"password"`

	// FlavorName is the name of the OpenStack flavor to create for Horizon tests.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:default:="m1.tiny"
	FlavorName string `json:"flavorName"`

	// LogsDirectoryName is the name of the directory to store test logs.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:default:="horizon"
	LogsDirectoryName string `json:"logsDirectoryName"`

	// HorizonTestDir is the directory path for Horizon tests.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:default:="/var/lib/horizontest"
	HorizonTestDir string `json:"horizonTestDir"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:default:=false
	// Parallel
	Parallel bool `json:"parallel"`

	// Name of a secret that contains a kubeconfig. The kubeconfig is mounted under /var/lib/horizontest/.kube/config
	// in the test pod.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:number"}
	KubeconfigSecretName string `json:"kubeconfigSecretName,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.conditions[0].status",description="Status"
//+kubebuilder:printcolumn:name="Message",type="string",JSONPath=".status.conditions[0].message",description="Message"

type HorizonTest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HorizonTestSpec  `json:"spec,omitempty"`
	Status CommonTestStatus `json:"status,omitempty"`
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
