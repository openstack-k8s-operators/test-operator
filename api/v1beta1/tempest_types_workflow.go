/*
Copyright 2024.

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
This file contains an extension of the Tempest CR. Ultimately it is a copy of
tempest_types.go that removes all default values for each config options. This
is necessary to be able to detect when the user explicitly sets a value in the
`workflow` section.
*/

package v1beta1

import corev1 "k8s.io/api/core/v1"

// WorkflowTempestRunSpec - is used to configure execution of tempest.
// Please refer to https://docs.openstack.org/tempest/latest/ for further
// explanation of the CLI parameters.
type WorkflowTempestRunSpec struct {
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// A content of include.txt file that is passed to tempest via --include-list
	IncludeList *string `json:"includeList,omitempty"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// A content of exclude.txt file that is passed to tempest via --exclude-list
	ExcludeList *string `json:"excludeList,omitempty"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// The expectedFailuresList parameter contains tests that should not count
	// as failures. When a test from this list fails, the test pod ends with
	// Completed state rather than with Error state.
	ExpectedFailuresList *string `json:"expectedFailuresList,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=128
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// Concurrency value that is passed to tempest via --concurrency
	Concurrency *int64 `json:"concurrency,omitempty"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// Indicate whether tempest should be executed with --smoke
	Smoke *bool `json:"smoke,omitempty"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// Indicate whether tempest should be executed with --parallel
	Parallel *bool `json:"parallel,omitempty"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// Indicate whether tempest should be executed with --serial
	Serial *bool `json:"serial,omitempty"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// A content of worker_file.yaml that is passed to tempest via --worker-file
	WorkerFile *string `json:"workerFile,omitempty"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// ExternalPlugin contains information about plugin that should be installed
	// within the tempest test pod. If this option is specified then only tests
	// that are part of the external plugin can be executed.
	ExternalPlugin *[]ExternalPluginType `json:"externalPlugin,omitempty"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// A list of URLs that point to RPMs that should be downloaded and installed
	// inside the tempest test pod.
	ExtraRPMs *[]string `json:"extraRPMs,omitempty"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// Extra images that should be downloaded inside the test pod and uploaded to
	// openstack.
	ExtraImages *[]ExtraImagesType `json:"extraImages,omitempty"`
}

// WorkflowTempestconfRunSpec - is used to configure execution of discover-tempest-config
// Please refer to https://docs.opendev.org/openinfra/python-tempestconf for
// further explanation of the CLI parameters.
type WorkflowTempestconfRunSpec struct {
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// Indicate whether discover-tempest-config should be executed with --create
	Create *bool `json:"create,omitempty"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// Indicate whether discover-tempest-config should be executed with
	// --collect-timing
	CollectTiming *bool `json:"collectTiming,omitempty"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// Indicate whether discover-tempest-config should be executed with --insecure
	Insecure *bool `json:"insecure,omitempty"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// Indicate whether discover-tempest-config should be executed with
	// --no-default-deployer
	NoDefaultDeployer *bool `json:"noDefaultDeployer,omitempty"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// Indicate whether discover-tempest-config should be executed with --debug
	Debug *bool `json:"debug,omitempty"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// Indicate whether discover-tempest-config should be executed with --verbose
	Verbose *bool `json:"verbose,omitempty"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// Indicate whether discover-tempest-config should be executed with --non-admin
	NonAdmin *bool `json:"nonAdmin,omitempty"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// Indicate whether discover-tempest-config should be executed with --retry-image
	RetryImage *bool `json:"retryImage,omitempty"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// Indicate whether discover-tempest-config should be executed with
	// --convert-to-raw
	ConvertToRaw *bool `json:"convertToRaw,omitempty"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// The content of this variable will be passed to discover-tempest-config via
	// the --out parameter
	Out *string `json:"out,omitempty"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// A content of deployer_input.ini that is passed to tempest via --deployer-input
	DeployerInput *string `json:"deployerInput,omitempty"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// A content of accounts.yaml that is passed to tempest via --test-accounts
	TestAccounts *string `json:"testAccounts,omitempty"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// The content of this variable will be passed to discover-tempest-config via
	// the --create-accounts-file
	CreateAccountsFile *string `json:"createAccountsFile,omitempty"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// A content of profile.yaml that is passed to tempest via --profile
	Profile *string `json:"profile,omitempty"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// The content of this variable will be passed to discover-tempest-config via
	// --generate-profile
	GenerateProfile *string `json:"generateProfile,omitempty"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// The content of this variable will be passed to discover-tempest-config via
	// --image-disk-format
	ImageDiskFormat *string `json:"imageDiskFormat,omitempty"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// The content of this variable will be passed to discover-tempest-config via
	// --image
	Image *string `json:"image,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=0
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// The content of this variable will be passed to discover-tempest-config via
	// --flavor-min-mem
	FlavorMinMem *int64 `json:"flavorMinMem,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=0
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// The content of this variable will be passed to discover-tempest-config via
	// --flavor-min-disk
	FlavorMinDisk *int64 `json:"flavorMinDisk,omitempty"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// The content of this variable will be passed to discover-tempest-config via
	// --network-id
	NetworkID *string `json:"networkID,omitempty"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// The content of this variable will be passed to discover-tempest-config via
	// --append
	Append *string `json:"append,omitempty"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// The content of this variable will be passed to discover-tempest-config via
	// --remove
	Remove *string `json:"remove,omitempty"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// The content of this variable will be appended at the end of the command
	// that executes discover-tempest-config (override values).
	Overrides *string `json:"overrides,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=0
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// The content of this variable will be passed to discover-tempest-config via
	// --timeout
	Timeout *int64 `json:"timeout,omitempty"`
}

// TempestSpec - configuration of execution of tempest. For specific configuration
// of tempest see TempestRunSpec and for discover-tempest-config see TempestconfRunSpec.
type WorkflowTempestSpec struct {
	WorkflowCommonOptions `json:",inline"`
	CommonOpenstackConfig `json:",inline"`

	// The desired amount of resources that should be assigned to each test pod
	// spawned using the Tempest CR. https://pkg.go.dev/k8s.io/api/core/v1#ResourceRequirements
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern:=^[a-z0-9-]+$
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// Name of a workflow step. The step name will be used for example to create
	// a logs directory.
	StepName string `json:"stepName"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// By default test-operator executes the test-pods sequentially if multiple
	// instances of test-operator related CRs exist. If you want to turn off this
	// behaviour then set this option to true.
	Parallel *bool `json:"parallel,omitempty"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// Activate tempest re-run feature. When activated, tempest will perform
	// another run of the tests that failed during the first execution.
	RerunFailedTests *bool `json:"rerunFailedTests,omitempty"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// Allow override of exit status with the tempest re-run feature.
	// When activated, the original return value of the tempest run will be
	// overridden with a result of the tempest run on the set of failed tests.
	RerunOverrideStatus *bool `json:"rerunOverrideStatus,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Format=uri
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// An URL pointing to an archive that contains the saved stestr timing data.
	// This data is used to optimize the tests order, which helps to reduce the
	// total Tempest execution time.
	TimingDataUrl *string `json:"timingDataUrl,omitempty"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// NetworkAttachments is a list of NetworkAttachment resource names to expose
	// the services to the given network
	NetworkAttachments *[]string `json:"networkAttachments,omitempty"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	TempestRun WorkflowTempestRunSpec `json:"tempestRun,omitempty"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	TempestconfRun WorkflowTempestconfRunSpec `json:"tempestconfRun,omitempty"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// SSHKeySecretName is the name of the k8s secret that contains an ssh key.
	// The key is mounted to ~/.ssh/id_ecdsa in the tempest pod
	SSHKeySecretName *string `json:"SSHKeySecretName,omitempty"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// ConfigOverwrite - interface to overwrite default config files like e.g. logging.conf
	// But can also be used to add additional files. Those get added to the
	// service config dir in /etc/test_operator/<file>
	ConfigOverwrite *map[string]string `json:"configOverwrite,omitempty"`
}
