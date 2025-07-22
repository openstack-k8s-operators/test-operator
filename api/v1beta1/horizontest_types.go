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

// HorizonTestSpec defines the desired state of HorizonTest
type HorizonTestSpec struct {
	CommonOptions `json:",inline"`

	// +kubebuilder:default:={limits: {cpu: "2000m", memory: "4Gi"}, requests: {cpu: "1000m", memory: "2Gi"}}
	// The desired amount of resources that should be assigned to each test pod
	// spawned using the HorizonTest CR. https://pkg.go.dev/k8s.io/api/core/v1#ResourceRequirements
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// Activate debug mode. When debug mode is activated any error encountered
	// inside the test-pod causes that the pod will be kept alive indefinitely
	// (stuck in "Running" phase) or until the corresponding HorizonTest CR is deleted.
	// This allows the user to debug any potential troubles with `oc rsh`.
	Debug bool `json:"debug"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// ExtraFlag is an extra flag that can be set to modify pytest command to
	// exclude or include particular test(s)
	ExtraFlag string `json:"extraFlag"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// ProjectNameXpath is the xpath to select project name
	// on the horizon dashboard based on the u/s or d/s theme
	ProjectNameXpath string `json:"projectNameXpath"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:default:="admin"
	// AdminUsername is the username for the OpenStack admin user.
	AdminUsername string `json:"adminUsername"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:default:="admin"
	// AdminPassword is the password for the OpenStack admin user.
	AdminPassword string `json:"adminPassword"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Format=uri
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// DashboardUrl is the URL of the Horizon dashboard.
	DashboardUrl string `json:"dashboardUrl"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Format=uri
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// AuthUrl is the authentication URL for OpenStack.
	AuthUrl string `json:"authUrl"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Format=uri
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:default:="https://review.opendev.org/openstack/horizon"
	// RepoUrl is the URL of the Horizon repository.
	RepoUrl string `json:"repoUrl"`

	// +kubebuilder:validation:Required
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:default:="master"
	// HorizonRepoBranch is the branch of the Horizon repository to checkout.
	HorizonRepoBranch string `json:"horizonRepoBranch"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Format=uri
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:default:="http://download.cirros-cloud.net/0.6.2/cirros-0.6.2-x86_64-disk.img"
	// ImageUrl is the URL to download the Cirros image.
	ImageUrl string `json:"imageUrl"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:default:="horizontest"
	// ProjectName is the name of the OpenStack project for Horizon tests.
	ProjectName string `json:"projectName"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MaxLength=253
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:default:="horizontest"
	// User is the username under which the Horizon tests will run.
	User string `json:"user"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MaxLength=253
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:default:="horizontest"
	// Password is the password for the user running the Horizon tests.
	Password string `json:"password"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:default:="m1.tiny"
	// FlavorName is the name of the OpenStack flavor to create for Horizon tests.
	FlavorName string `json:"flavorName"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:default:="horizon"
	// LogsDirectoryName is the name of the directory to store test logs.
	LogsDirectoryName string `json:"logsDirectoryName"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:default:="/var/lib/horizontest"
	// HorizonTestDir is the directory path for Horizon tests.
	HorizonTestDir string `json:"horizonTestDir"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:default:=false
	// Parallel
	Parallel bool `json:"parallel"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:number"}
	// Name of a secret that contains a kubeconfig. The kubeconfig is mounted under /var/lib/horizontest/.kube/config
	// in the test pod.
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
