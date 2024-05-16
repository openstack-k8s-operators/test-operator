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

/*
This file contains an extension of the Tempest CR. Ultimataly it is a copy of
tempest_types.go that removes all default values for each config options. This
is necessary to be able to detect when the user explicitly set a value in the
`workflow` setcion.
*/

package v1beta1

import (
	"github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	corev1 "k8s.io/api/core/v1"
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

// ExtraImagesType - is used to specify extra images that should be downloaded
// inside the test pod and uploaded to openstack
type ExtraImagesType struct {
	// +kubebuilder:validation:Required
	// URL that points to a location where the image is located
	URL string `json:"URL"`

	// +kubebuilder:validation:Required
	// Name of the image
	Name string `json:"name"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="-"
	// Cloud that should be used for authentication
	OsCloud string `json:"osCloud"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="-"
	// Image container format
	ContainerFormat string `json:"containerFormat"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="-"
	// Image disk format
	DiskFormat string `json:"diskFormat"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="-"
	// ID that should be assigned to the newly created image
	ID string `json:"ID"`

	// +kubebuilder:validation:Optional
	// Information about flavor that should be created together with the image
	Flavor ExtraImagesFlavorType `json:"flavor"`
}

type ExtraImagesFlavorType struct {
	// +kubebuilder:validation:Required
	// Name of the flavor that should be created
	Name string `json:"name"`

	// +kubebuilder:validation:Required
	// How much RAM should be allocated when this flavor is used
	RAM int64 `json:"RAM"`

	// +kubebuilder:validation:Required
	// How much disk space should be allocated when this flavor is used
	Disk int64 `json:"disk"`

	// +kubebuilder:validation:Required
	// How many vcpus should be be allocated when this flavor is used
	Vcpus int64 `json:"vcpus"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="-"
	// ID that should be assigned to the newly created flavor
	ID string `json:"ID"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="-"
	// Cloud that should be used for authentication
	OsCloud string `json:"osCloud"`
}

// ExternalPluginType - is used to specify a plugin that should be installed
// from an external resource
type ExternalPluginType struct {
	// +kubebuilder:validation:Required
	// URL that points to a git repository containing an external plugin.
	Repository string `json:"repository,omitempty"`

	// +kubebuilder:validation:Optional
	// URL that points to a repository that contains a change that should be
	// applied to the repository defined by Repository (ChangeRefspec must be
	// defined as well).
	ChangeRepository string `json:"changeRepository,omitempty"`

	// +kubebuilder:validation:Optional
	// ChangeRefspec specifies which change the remote repository should be
	// checked out to (ChangeRepository must be defined as well).
	ChangeRefspec string `json:"changeRefspec,omitempty"`
}

// TempestRunSpec - is used to configure execution of tempest. Please refer to
// Please refer to https://docs.openstack.org/tempest/latest/ for the further
// explanation of the CLI parameters.
type TempestRunSpec struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default="tempest.api.identity.v3"
	// A content of include.txt file that is passed to tempest via --include-list
	IncludeList string `json:"includeList,omitempty"`

	// +kubebuilder:validation:Optional
	// A content of exclude.txt file that is passed to tempest via --exclude-list
	ExcludeList string `json:"excludeList,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=0
	// Concurrency value that is passed to tempest via --concurrency
	Concurrency int64 `json:"concurrency,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=false
	// Indicate whether tempest should be executed with --smoke
	Smoke bool `json:"smoke,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=true
	// Indicate whether tempest should be executed with --parallel
	Parallel bool `json:"parallel,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=false
	// Indicate whether tempest should be executed with --serial
	Serial bool `json:"serial,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=""
	// A content of worker_file.yaml that is passed to tempest via --worker-file
	WorkerFile string `json:"workerFile,omitempty"`

	// +kubebuilder:validation:Optional
	// ExternalPlugin contains information about plugin that should be installed
	// within the tempest test pod. If this option is specified then only tests
	// that are part of the external plugin can be executed.
	ExternalPlugin []ExternalPluginType `json:"externalPlugin,omitempty"`

	// +kubebuilder:validation:Optional
	// Extra images that should be downloaded inside the test pod and uploaded to
	// openstack.
	ExtraImages []ExtraImagesType `json:"extraImages,omitempty"`
}

// TempestconfRunSpec - is used to configure execution of discover-tempest-config
// Please refer to https://docs.opendev.org/openinfra/python-tempestconf for the
// further explanation of the CLI parameters.
type TempestconfRunSpec struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=true
	// Indicate whether discover-tempest-config should be executed with --create
	Create bool `json:"create"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// Indicate whether discover-tempest-config should be executed with
	// --collect-timing
	CollectTiming bool `json:"collectTiming"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// Indicate whether discover-tempest-config should be executed with --insecure
	Insecure bool `json:"insecure"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// Indicate whether discover-tempest-config should be executed with
	// --no-default-deployer
	NoDefaultDeployer bool `json:"noDefaultDeployer"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// Indicate whether discover-tempest-config should be executed with --debug
	Debug bool `json:"debug"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// Indicate whether discover-tempest-config should be executed with --verbose
	Verbose bool `json:"verbose"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// Indicate whether discover-tempest-config should be executed with --non-admin
	NonAdmin bool `json:"nonAdmin"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// Indicate whether discover-tempest-config should be executed with --retry-image
	RetryImage bool `json:"retryImage"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// Indicate whether discover-tempest-config should be executed with
	// --convert-to-raw
	ConvertToRaw bool `json:"convertToRaw"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=""
	// The content of this variable will be passed to discover-tempest-config via
	// the --out parameter
	Out string `json:"out"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=""
	// A content of deployer_input.ini that is passed to tempest via --deployer-input
	DeployerInput string `json:"deployerInput"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=""
	// A content of accounts.yaml that is passed to tempest via --test-acounts
	TestAccounts string `json:"testAccounts"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=""
	// The content of this variable will be passed to discover-tempest-config via
	// the --create-accounts-file
	CreateAccountsFile string `json:"createAccountsFile"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=""
	// A content of profile.yaml that is passed to tempest via --profile
	Profile string `json:"profile"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=""
	// The content of this variable will be passed to discover-tempest-config via
	// --generate-profile
	GenerateProfile string `json:"generateProfile"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=""
	// The content of this variable will be passed to discover-tempest-config via
	// --image-disk-format
	ImageDiskFormat string `json:"imageDiskFormat"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=""
	// The content of this variable will be passed to discover-tempest-config via
	// --image
	Image string `json:"image"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=0
	// The content of this variable will be passed to discover-tempest-config via
	// --flavor-min-mem
	FlavorMinMem int64 `json:"flavorMinMem"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=0
	// The content of this variable will be passed to discover-tempest-config via
	// --flavor-min-disk
	FlavorMinDisk int64 `json:"flavorMinDisk"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=""
	// The content of this variable will be passed to discover-tempest-config via
	// --network-id
	NetworkID string `json:"networkID"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=""
	// The content of this variable will be passed to discover-tempest-config via
	// --append
	Append string `json:"append"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=""
	// The content of this variable will be passed to discover-tempest-config via
	// --remove
	Remove string `json:"remove"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="identity.v3_endpoint_type public"
	// The content of this variable will be appended at the end of the command
	// that executes discover-tempest-config (override values).
	Overrides string `json:"overrides"`

	// +kubebuilder:default=0
	// The content of this variable will be passed to discover-tempest-config via
	// --timeout
	Timeout int64 `json:"timeout"`
}

// TempestSpec - configuration of execution of tempest. For specific configuration
// of tempest see TempestRunSpec and for discover-tempest-config see TempestconfRunSpec.
type TempestSpec struct {

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="s0:c478,c978"
	// A SELinuxLevel that is used for all the tempest test pods.
	SELinuxLevel string `json:"SELinuxLevel"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="local-storage"
	// Name of a storage class that is used to create PVCs for logs storage. Required
	// if default storage class does not exist.
	StorageClass string `json:"storageClass"`

	// +kubebuilder:validation:Required
	// An URL of a tempest container image that should be used for the execution
	// of tempest tests.
	ContainerImage string `json:"containerImage"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// By default test-operator executes the test-pods sequentially if multiple
	// instances of test-operator related CRs exist. If you want to turn off this
	// behaviour then set this option to true.
	Parallel bool `json:"parallel,omitempty"`

	// +kubebuilder:validation:Optional
	// This value contains a nodeSelector value that is applied to test pods
	// spawned by the test operator.
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// +kubebuilder:validation:Optional
        // +kubebuilder:default=false
        // Activate debug mode. When debug mode is activated any error encountered
        // inside the test-pod causes that the pod will be kept alive indefinitely
        // (stuck in "Running" phase) or until the corresponding Tempest CR is deleted.
        // This allows the user to debug any potential troubles with `oc rsh`.
        Debug bool `json:"debug,omitempty"`

	// +kubebuilder:validation:Optional
	// This value contains a toleration that is applied to pods spawned by the
	// test pods that are spawned by the test-operator.
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// +kubebuilder:validation:Required
	// +kubebuilder:default=openstack-config
	// OpenStackConfigMap is the name of the ConfigMap containing the clouds.yaml
	OpenStackConfigMap string `json:"openStackConfigMap"`

	// +kubebuilder:validation:Required
	// +kubebuilder:default=openstack-config-secret
	// OpenStackConfigSecret is the name of the Secret containing the secure.yaml
	OpenStackConfigSecret string `json:"openStackConfigSecret"`

	// +kubebuilder:validation:Optional
	// NetworkAttachments is a list of NetworkAttachment resource names to expose
	// the services to the given network
	NetworkAttachments []string `json:"networkAttachments,omitempty"`

	// BackoffLimimt allows to define the maximum number of retried executions (defaults to 6).
	// +kubebuilder:default:=0
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:number"}
	BackoffLimit *int32 `json:"backoffLimit,omitempty"`

	// +kubebuilder:validation:Optional
	TempestRun TempestRunSpec `json:"tempestRun,omitempty"`

	// +kubebuilder:validation:Optional
	TempestconfRun TempestconfRunSpec `json:"tempestconfRun,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=""
	// SSHKeySecretName is the name of the k8s secret that contains an ssh key.
	// The key is mounted to ~/.ssh/id_ecdsa in the tempest pod
	SSHKeySecretName string `json:"SSHKeySecretName,omitempty"`

	// +kubebuilder:validation:Optional
	// ConfigOverwrite - interface to overwrite default config files like e.g. logging.conf
	// But can also be used to add additional files. Those get added to the
	// service config dir in /etc/test_operator/<file>
	ConfigOverwrite map[string]string `json:"configOverwrite,omitempty"`

	// +kubebuilder:validation:Optional
	// Workflow - can be used to specify a multiple executions of tempest with
	// a different configuration in a single CR. Accepts a list of dictionaries
	// where each member of the list accepts the same values as the Tempest CR
	// does in the `spec`` section. Values specified using the workflow section have
	// a higher precedence than the values specified higher in the Tempest CR
	// hierarchy.
	Workflow []WorkflowTempestSpec `json:"workflow,omitempty"`
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
