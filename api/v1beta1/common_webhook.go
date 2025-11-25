package v1beta1

import (
	"github.com/openstack-k8s-operators/lib-common/modules/common/util"
	goClient "sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// webhookClient is a client that will be initialized from internal/webhook SetupWebhookWithManager
// function and used for webhook functions (validation, defaulting that need to access resources
// to any particular webhook)
var webhookClient goClient.Client

// log is for logging in this package.
var testDefaultslog = logf.Log.WithName("test-defaults")

// TestDefaults -
type TestDefaults struct {
	TempestContainerImageURL     string
	TobikoContainerImageURL      string
	AnsibleTestContainerImageURL string
	HorizonTestContainerImageURL string
}

var testDefaults TestDefaults

// SetupOctaviaDefaults - initialize Octavia spec defaults for use with either internal or external webhooks
func SetupTestDefaults(defaults TestDefaults) {
	testDefaults = defaults
	testDefaultslog.Info("Test defaults initialized", "defaults", defaults)
}

// SetupWebhookClient sets the webhook client for API webhook functions.
// This allows internal webhooks to initialize the client before calling validation/default functions.
func SetupWebhookClient(client goClient.Client) {
	webhookClient = client
}

const (
	// ErrPrivilegedModeRequired
	ErrPrivilegedModeRequired = "%s.Spec.Privileged is required in order to successfully " +
		"execute tests with the provided configuration."

	// ErrDebug
	ErrDebug = "%s.Spec.Workflow parameter must be empty to run debug mode"

	// ErrNameTooLong
	ErrNameTooLong = "The combined length of %s pod name exceeds the maximum of %d " +
		"characters. Shorten the CR name or workflow step name to proceed."
)

const (
	// WarnPrivilegedModeOn
	WarnPrivilegedModeOn = "%s.Spec.Privileged is set to true. This means that test pods " +
		"are spawned with allowPrivilegedEscalation: true, readOnlyRootFilesystem: false, " +
		"runAsNonRoot: false, automountServiceAccountToken: true and default " +
		"capabilities on top of those required by the test operator (NET_ADMIN, NET_RAW)."

	// WarnPrivilegedModeOff
	WarnPrivilegedModeOff = "%[1]s.Spec.Privileged is set to false. Note, that a certain " +
		"set of tests might fail, as this configuration may be " +
		"required for the tests to run successfully. Before enabling" +
		"this parameter, consult documentation of the %[1]s CR."

	// WarnSELinuxLevel
	WarnSELinuxLevel = "%[1]s.Spec.Workflow is used and %[1]s.Spec.Privileged is " +
		"set to true. Please, consider setting %[1]s.Spec.SELinuxLevel. This " +
		"ensures that the copying of the logs to the PV is completed without any " +
		"complications."
)

const (
	DefaultTempestContainerImageURL     = "quay.io/podified-antelope-centos9/openstack-tempest-all:current-podified"
	DefaultTobikoContainerImageURL      = "quay.io/podified-antelope-centos9/openstack-tobiko:current-podified"
	DefaultAnsibleTestContainerImageURL = "quay.io/podified-antelope-centos9/openstack-ansible-tests:current-podified"
	DefaultHorizonTestContainerImageURL = "quay.io/podified-antelope-centos9/openstack-horizontest:current-podified"
)

// SetupDefaults - initializes any CRD field defaults based on environment variables (the defaulting mechanism itself is implemented via webhooks)
func SetupDefaults() {
	// Acquire environmental defaults and initialize Octavia defaults with them
	testDefaults := TestDefaults{
		TempestContainerImageURL:     util.GetEnvVar("RELATED_IMAGE_TEST_TEMPEST_IMAGE_URL_DEFAULT", DefaultTempestContainerImageURL),
		TobikoContainerImageURL:      util.GetEnvVar("RELATED_IMAGE_TEST_TOBIKO_IMAGE_URL_DEFAULT", DefaultTobikoContainerImageURL),
		AnsibleTestContainerImageURL: util.GetEnvVar("RELATED_IMAGE_TEST_ANSIBLETEST_IMAGE_URL_DEFAULT", DefaultAnsibleTestContainerImageURL),
		HorizonTestContainerImageURL: util.GetEnvVar("RELATED_IMAGE_HORIZONTEST_IMAGE_URL_DEFAULT", DefaultHorizonTestContainerImageURL),
	}

	SetupTestDefaults(testDefaults)
}
