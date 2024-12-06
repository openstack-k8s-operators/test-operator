package v1beta1

const (
	// ErrPrivilegedModeRequired
	ErrPrivilegedModeRequired = "%s.Spec.Privileged is requied in order to successfully " +
		"execute tests with the provided configuration."

	// ErrDebug
	ErrDebug = "%s.Spec.Workflow parameter must be empty to run debug mode"
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
