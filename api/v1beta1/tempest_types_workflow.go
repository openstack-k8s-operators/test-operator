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

// TempestRunSpec - is used to configure execution of tempest. Please refer to
// Please refer to https://docs.openstack.org/tempest/latest/ for the further
// explanation of the CLI parameters.
type WorkflowTempestRunSpec struct {
	// +kubebuilder:validation:Optional
	// A content of include.txt file that is passed to tempest via --include-list
	IncludeList *string `json:"includeList,omitempty"`

	// +kubebuilder:validation:Optional
	// A content of exclude.txt file that is passed to tempest via --exclude-list
	ExcludeList *string `json:"excludeList,omitempty"`

	// +kubebuilder:validation:Optional
	// Concurrency value that is passed to tempest via --concurrency
	Concurrency *int64 `json:"concurrency,omitempty"`

	// +kubebuilder:validation:Optional
	// Indicate whether tempest should be executed with --smoke
	Smoke *bool `json:"smoke,omitempty"`

	// +kubebuilder:validation:Optional
	// Indicate whether tempest should be executed with --parallel
	Parallel *bool `json:"parallel,omitempty"`

	// +kubebuilder:validation:Optional
	// Indicate whether tempest should be executed with --serial
	Serial *bool `json:"serial,omitempty"`

	// +kubebuilder:validation:Optional
	// A content of worker_file.yaml that is passed to tempest via --worker-file
	WorkerFile *string `json:"workerFile,omitempty"`

	// +kubebuilder:validation:Optional
	// ExternalPlugin contains information about plugin that should be installed
	// within the tempest test pod. If this option is specified then only tests
	// that are part of the external plugin can be executed.
	ExternalPlugin *[]ExternalPluginType `json:"externalPlugin,omitempty"`
}

// TempestconfRunSpec - is used to configure execution of discover-tempest-config
// Please refer to https://docs.opendev.org/openinfra/python-tempestconf for the
// further explanation of the CLI parameters.
type WorkflowTempestconfRunSpec struct {
	// +kubebuilder:validation:Optional
	// Indicate whether discover-tempest-config should be executed with --create
	Create *bool `json:"create"`

	// +kubebuilder:validation:Optional
	// Indicate whether discover-tempest-config should be executed with
	// --collect-timing
	CollectTiming *bool `json:"collectTiming"`

	// +kubebuilder:validation:Optional
	// Indicate whether discover-tempest-config should be executed with --insecure
	Insecure *bool `json:"insecure"`

	// +kubebuilder:validation:Optional
	// Indicate whether discover-tempest-config should be executed with
	// --no-default-deployer
	NoDefaultDeployer *bool `json:"noDefaultDeployer"`

	// +kubebuilder:validation:Optional
	// Indicate whether discover-tempest-config should be executed with --debug
	Debug *bool `json:"debug"`

	// +kubebuilder:validation:Optional
	// Indicate whether discover-tempest-config should be executed with --verbose
	Verbose *bool `json:"verbose"`

	// +kubebuilder:validation:Optional
	// Indicate whether discover-tempest-config should be executed with --non-admin
	NonAdmin *bool `json:"nonAdmin"`

	// +kubebuilder:validation:Optional
	// Indicate whether discover-tempest-config should be executed with --retry-image
	RetryImage *bool `json:"retryImage"`

	// +kubebuilder:validation:Optional
	// Indicate whether discover-tempest-config should be executed with
	// --convert-to-raw
	ConvertToRaw *bool `json:"convertToRaw"`

	// +kubebuilder:validation:Optional
	// The content of this variable will be passed to discover-tempest-config via
	// the --out parameter
	Out *string `json:"out"`

	// +kubebuilder:validation:Optional
	// A content of deployer_input.ini that is passed to tempest via --deployer-input
	DeployerInput *string `json:"deployerInput"`

	// +kubebuilder:validation:Optional
	// A content of accounts.yaml that is passed to tempest via --test-acounts
	TestAccounts *string `json:"testAccounts"`

	// +kubebuilder:validation:Optional
	// The content of this variable will be passed to discover-tempest-config via
	// the --create-accounts-file
	CreateAccountsFile *string `json:"createAccountsFile"`

	// +kubebuilder:validation:Optional
	// A content of profile.yaml that is passed to tempest via --profile
	Profile *string `json:"profile"`

	// +kubebuilder:validation:Optional
	// The content of this variable will be passed to discover-tempest-config via
	// --generate-profile
	GenerateProfile *string `json:"generateProfile"`

	// +kubebuilder:validation:Optional
	// The content of this variable will be passed to discover-tempest-config via
	// --image-disk-format
	ImageDiskFormat *string `json:"imageDiskFormat"`

	// +kubebuilder:validation:Optional
	// The content of this variable will be passed to discover-tempest-config via
	// --image
	Image *string `json:"image"`

	// +kubebuilder:validation:Optional
	// The content of this variable will be passed to discover-tempest-config via
	// --flavor-min-mem
	FlavorMinMem *int64 `json:"flavorMinMem"`

	// +kubebuilder:validation:Optional
	// The content of this variable will be passed to discover-tempest-config via
	// --flavor-min-disk
	FlavorMinDisk *int64 `json:"flavorMinDisk"`

	// +kubebuilder:validation:Optional
	// The content of this variable will be passed to discover-tempest-config via
	// --network-id
	NetworkID *string `json:"networkID"`

	// +kubebuilder:validation:Optional
	// The content of this variable will be passed to discover-tempest-config via
	// --append
	Append *string `json:"append"`

	// +kubebuilder:validation:Optional
	// The content of this variable will be passed to discover-tempest-config via
	// --remove
	Remove *string `json:"remove"`

	// +kubebuilder:validation:Optional
	// The content of this variable will be appended at the end of the command
	// that executes discover-tempest-config (override values).
	Overrides *string `json:"overrides"`

	// +kubebuilder:validation:Optional
	// The content of this variable will be passed to discover-tempest-config via
	// --timeout
	Timeout *int64 `json:"timeout"`
}

// TempestSpec - configuration of execution of tempest. For specific configuration
// of tempest see TempestRunSpec and for discover-tempest-config see TempestconfRunSpec.
type WorkflowTempestSpec struct {
	// +kubebuilder:validation:Required
	// Name of a workflow step. The step name will be used for example to create
	// a logs directory.
	StepName string `json:"stepName"`

	// +kubebuilder:validation:Optional
	// Name of a storage class that is used to create PVCs for logs storage. Required
	// if default storage class does not exist.
	StorageClass *string `json:"storageClass"`

	// +kubebuilder:validation:Optional
	// An URL of a tempest container image that should be used for the execution
	// of tempest tests.
	ContainerImage *string `json:"containerImage"`

	// +kubebuilder:validation:Optional
	// By default test-operator executes the test-pods sequentially if multiple
	// instances of test-operator related CRs exist. If you want to turn off this
	// behaviour then set this option to true.
	Parallel *bool `json:"parallel,omitempty"`

	// +kubebuilder:validation:Optional
	// NodeSelector to target subset of worker nodes running this service
	NodeSelector *map[string]string `json:"nodeSelector,omitempty"`

	// +kubebuilder:validation:Optional
	// OpenStackConfigMap is the name of the ConfigMap containing the clouds.yaml
	OpenStackConfigMap *string `json:"openStackConfigMap"`

	// +kubebuilder:validation:Optional
	// OpenStackConfigSecret is the name of the Secret containing the secure.yaml
	OpenStackConfigSecret *string `json:"openStackConfigSecret"`

	// +kubebuilder:validation:Optional
	// NetworkAttachments is a list of NetworkAttachment resource names to expose
	// the services to the given network
	NetworkAttachments *[]string `json:"networkAttachments,omitempty"`

	// BackoffLimimt allows to define the maximum number of retried executions (defaults to 6).
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:number"}
	BackoffLimit *int32 `json:"backoffLimit,omitempty"`

	// +kubebuilder:validation:Optional
	TempestRun WorkflowTempestRunSpec `json:"tempestRun,omitempty"`

	// +kubebuilder:validation:Optional
	TempestconfRun WorkflowTempestconfRunSpec `json:"tempestconfRun,omitempty"`

	// +kubebuilder:validation:Optional
	// SSHKeySecretName is the name of the k8s secret that contains an ssh key.
	// The key is mounted to ~/.ssh/id_ecdsa in the tempest pod
	SSHKeySecretName *string `json:"SSHKeySecretName,omitempty"`

	// +kubebuilder:validation:Optional
	// ConfigOverwrite - interface to overwrite default config files like e.g. logging.conf
	// But can also be used to add additional files. Those get added to the
	// service config dir in /etc/test_operator/<file>
	ConfigOverwrite *map[string]string `json:"configOverwrite,omitempty"`
}
