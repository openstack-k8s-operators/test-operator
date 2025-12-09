package v1beta1

import (
	"fmt"
	"reflect"

	"github.com/openstack-k8s-operators/lib-common/modules/common/util"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
	goClient "sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
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

// ValidatePodName checks if CR name exceeds DNS label length limit
func ValidatePodName(allErrs field.ErrorList, name, kind string) field.ErrorList {
	if len(name) >= validation.DNS1123LabelMaxLength {
		allErrs = append(allErrs, &field.Error{
			Type:     field.ErrorTypeInvalid,
			BadValue: len(name),
			Detail:   fmt.Sprintf(ErrNameTooLong, kind, validation.DNS1123LabelMaxLength),
		})
	}
	return allErrs
}

// ValidateWorkflowPodNames checks if workflow step pod names exceed DNS label length limit
func ValidateWorkflowPodNames(allErrs field.ErrorList, name, kind string, workflow interface{}) field.ErrorList {
	v := reflect.ValueOf(workflow)

	for i := 0; i < v.Len(); i++ {
		stepName := v.Index(i).FieldByName("StepName").String()
		podNameLength := len(name) + len(stepName) + len("-sXX-")
		if podNameLength >= validation.DNS1123LabelMaxLength {
			allErrs = append(allErrs, &field.Error{
				Type:     field.ErrorTypeInvalid,
				BadValue: podNameLength,
				Detail:   fmt.Sprintf(ErrNameTooLong, kind, validation.DNS1123LabelMaxLength),
			})
		}
	}
	return allErrs
}

// CheckExtraConfigmapsDeprecation returns warning if ExtraConfigmapsMounts is used
func CheckExtraConfigmapsDeprecation(allWarn admission.Warnings, extraConfigmaps interface{}) admission.Warnings {
	if v := reflect.ValueOf(extraConfigmaps); v.Len() > 0 {
		allWarn = append(allWarn, "The ExtraConfigmapsMounts parameter will be"+
			" deprecated! Please use ExtraMounts parameter instead!")
	}
	return allWarn
}

// CheckWorkflowExtraConfigmapsDeprecation checks for deprecated field in workflow steps
func CheckWorkflowExtraConfigmapsDeprecation(allWarn admission.Warnings, workflow interface{}) admission.Warnings {
	v := reflect.ValueOf(workflow)

	for i := 0; i < v.Len(); i++ {
		if field := v.Index(i).FieldByName("ExtraConfigmapsMounts"); field.IsValid() && !field.IsNil() {
			allWarn = append(allWarn, "The ExtraConfigmapsMounts parameter will be"+
				" deprecated! Please use ExtraMounts parameter instead!")
		}
	}
	return allWarn
}

// CheckPrivilegedWarning returns warning if privileged mode is enabled
func CheckPrivilegedWarning(allWarn admission.Warnings, privileged bool, kind string) admission.Warnings {
	if privileged {
		allWarn = append(allWarn, fmt.Sprintf(WarnPrivilegedModeOn, kind))
	}
	return allWarn
}

// CheckSELinuxWarning returns warning if privileged mode + workflow but no SELinux level
func CheckSELinuxWarning(allWarn admission.Warnings, privileged bool, seLinuxLevel, kind string) admission.Warnings {
	if privileged && len(seLinuxLevel) == 0 {
		allWarn = append(allWarn, fmt.Sprintf(WarnSELinuxLevel, kind))
	}
	return allWarn
}

// ValidateDebugWorkflow validates that debug mode and workflow are not both set
func ValidateDebugWorkflow(allErrs field.ErrorList, debug bool, kind string) field.ErrorList {
	if debug {
		allErrs = append(allErrs, &field.Error{
			Type:     field.ErrorTypeForbidden,
			BadValue: debug,
			Detail:   fmt.Sprintf(ErrDebug, kind),
		})
	}
	return allErrs
}

// BuildValidationError constructs an Invalid error from field errors
func BuildValidationError(kind, name string, errs field.ErrorList) error {
	// red error prefix
	for i := range errs {
		errs[i].Detail = fmt.Sprintf("\033[31mError:\033[0m %s", errs[i].Detail)
	}

	if len(errs) > 0 {
		return apierrors.NewInvalid(
			schema.GroupKind{
				Group: GroupVersion.WithKind(kind).Group,
				Kind:  GroupVersion.WithKind(kind).Kind,
			}, name, errs)
	}
	return nil
}
