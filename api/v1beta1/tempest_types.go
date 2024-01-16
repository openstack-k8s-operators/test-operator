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
	"github.com/openstack-k8s-operators/lib-common/modules/common/endpoint"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// ConfigHash - TempestConfigHash key
	ConfigHash = "TempestConfigHash"
)

// Hash - struct to add hashes to status
type Hash struct {
	// Name of hash referencing the parameter
	Name string `json:"name,omitempty"`
	// Hash
	Hash string `json:"hash,omitempty"`
}


type ExternalPluginType struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=""
	// URL that points to a git repository containing
	// the external plugin.
	Repository string `json:"repository,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=""
	// URL that points to a repository that contains a change
	// that should be applied to the repository defined by Repository
	// (ChangeRefspec must be defined as well).
	ChangeRepository string `json:"changeRepository,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=""
	// ChangeRefspec specifies which change the remote repository
	// should be checked out to (ChangeRepository must be defined
	// as well).
	ChangeRefspec string `json:"changeRefspec,omitempty"`
}

// TempestSpec TempestRun parts
type TempestRunSpec struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default="tempest.api.identity.v3"
	// IncludeList
	IncludeList string `json:"includeList,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=""
	// ExcludeList
	ExcludeList string `json:"excludeList,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=-1
	// Concurrency is the Default concurrency
	Concurrency int64 `json:"concurrency,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=false
	// Smoke tests
	Smoke bool `json:"smoke,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=true
	// Run tests in parallel
	Parallel bool `json:"parallel,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=false
	// Serial run
	Serial bool `json:"serial,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=""
	// WorkerFile is the detailed concurrency spec file
	WorkerFile string `json:"workerFile,omitempty"`

        // +kubebuilder:validation:Optional
        // ExternalPlugin contains information about plugin
        // that should be installed within the tempest container.
        // If this option is specified then only tests that are part of
        // the external plugin can be executed.
        ExternalPlugin []ExternalPluginType `json:"externalPlugin,omitempty"`
}

// TempestSpec PythonTempestconf parts
type TempestconfRunSpec struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=true
	// Create Tempest resources
	Create bool `json:"create"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// Collect per-API call timing information.
	CollectTiming bool `json:"collectTiming"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// Explicitly allow client to perform “insecure” TLS (https) requests.
	Insecure bool `json:"insecure"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// Do not check for the default deployer input in
	NoDefaultDeployer bool `json:"noDefaultDeployer"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// Print debugging information.
	Debug bool `json:"debug"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// Print more information about the execution.
	Verbose bool `json:"verbose"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// Simulate non-admin credentials.
	NonAdmin bool `json:"nonAdmin"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// Allow tempestconf to retry download an image, in case of failure.
	RetryImage bool `json:"retryImage"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// Convert images to raw format before uploading.
	ConvertToRaw bool `json:"convertToRaw"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=""
	// Output file
	Out string `json:"out"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=""
	// Path to deployer file
	DeployerInput string `json:"deployerInput"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=""
	// Tempest accounts.yaml file
	TestAccounts string `json:"testAccounts"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=""
	// Generate Tempest accounts file.
	CreateAccountsFile string `json:"createAccountsFile"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=""
	// python-tempestconf’s profile.yaml file
	Profile string `json:"profile"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=""
	// Generate a sample profile.yaml file.
	GenerateProfile string `json:"generateProfile"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=""
	// A format of an image to be uploaded to glance.
	ImageDiskFormat string `json:"imageDiskFormat"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=""
	// An image name/path/url to be uploaded to glance if it’s not already there.
	Image string `json:"image"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=-1
	// Specify minimum memory for new flavors
	FlavorMinMem int64 `json:"flavorMinMem"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=-1
	// Specify minimum disk size for new flavors
	FlavorMinDisk int64 `json:"flavorMinDisk"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=""
	// Specify which network with external connectivity should be used by the test.
	NetworkID string `json:"networkID"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=""
	// Append values to tempest.conf
	Append string `json:"append"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=""
	// Append values to tempest.conf
	Remove string `json:"remove"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="identity.v3_endpoint_type public"
	// Override options
	Overrides string `json:"overrides"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=-1
	// Set request timeout (in seconds).
	Timeout int64 `json:"timeout"`
}

// TempestSpec defines the desired state of Tempest
type TempestSpec struct {
	// +kubebuilder:validation:Required
	// Tempest Container Image URL (will be set to environmental default if empty)
	ContainerImage string `json:"containerImage"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
        // Execute tests parallely
	Parallel bool `json:"parallel,omitempty"`

	// +kubebuilder:validation:Optional
	// NodeSelector to target subset of worker nodes running this service
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// +kubebuilder:validation:Required
	// +kubebuilder:default=openstack-config
	// OpenStackConfigMap is the name of the ConfigMap containing the clouds.yaml
	OpenStackConfigMap string `json:"openStackConfigMap"`

	// +kubebuilder:validation:Required
	// +kubebuilder:default=openstack-config-secret
	// OpenStackConfigSecret is the name of the Secret containing the secure.yaml
	OpenStackConfigSecret string `json:"openStackConfigSecret"`

	// +kubebuilder:validation:Optional
	// NetworkAttachments is a list of NetworkAttachment resource names to expose the services to the given network
	NetworkAttachments []string `json:"networkAttachments,omitempty"`

	// +kubebuilder:validation:Optional
	// ExternalEndpoints, expose a VIP using a pre-created IPAddressPool
	ExternalEndpoints []MetalLBConfig `json:"externalEndpoints,omitempty"`

	// BackoffLimimt allows to define the maximum number of retried executions (defaults to 6).
	// +kubebuilder:default:=0
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:number"}
	BackoffLimit *int32 `json:"backoffLimit,omitempty"`

	// +kubebuilder:validation:Optional
	TempestRun *TempestRunSpec `json:"tempestRun,omitempty"`

	// +kubebuilder:validation:Optional
	TempestconfRun *TempestconfRunSpec `json:"tempestconfRun,omitempty"`

	// TODO(slaweq): add more tempest run parameters here
}

// MetalLBConfig to configure the MetalLB loadbalancer service
type MetalLBConfig struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=internal;public
	// Endpoint, OpenStack endpoint this service maps to
	Endpoint endpoint.Endpoint `json:"endpoint"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// IPAddressPool expose VIP via MetalLB on the IPAddressPool
	IPAddressPool string `json:"ipAddressPool"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=true
	// SharedIP if true, VIP/VIPs get shared with multiple services
	SharedIP bool `json:"sharedIP"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=""
	// SharedIPKey specifies the sharing key which gets set as the annotation on the LoadBalancer service.
	// Services which share the same VIP must have the same SharedIPKey. Defaults to the IPAddressPool if
	// SharedIP is true, but no SharedIPKey specified.
	SharedIPKey string `json:"sharedIPKey"`

	// +kubebuilder:validation:Optional
	// LoadBalancerIPs, request given IPs from the pool if available. Using a list to allow dual stack (IPv4/IPv6) support
	LoadBalancerIPs []string `json:"loadBalancerIPs,omitempty"`
}

// TempestStatus defines the observed state of Tempest
type TempestStatus struct {

	// Map of hashes to track e.g. job status
	Hash map[string]string `json:"hash,omitempty"`

	// Conditions
	Conditions condition.Conditions `json:"conditions,omitempty" optional:"true"`

	// NetworkAttachments status of the deployment pods
	NetworkAttachments map[string][]string `json:"networkAttachments,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Tempest is the Schema for the tempests API
type Tempest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TempestSpec   `json:"spec,omitempty"`
	Status TempestStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// TempestList contains a list of Tempest
type TempestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Tempest `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Tempest{}, &TempestList{})
}

// RbacConditionsSet - set the conditions for the rbac object
func (instance Tempest) RbacConditionsSet(c *condition.Condition) {
	instance.Status.Conditions.Set(c)
}

// RbacNamespace - return the namespace
func (instance Tempest) RbacNamespace() string {
	return instance.Namespace
}

// RbacResourceName - return the name to be used for rbac objects (serviceaccount, role, rolebinding)
func (instance Tempest) RbacResourceName() string {
	return instance.Name
}
