package v1beta1

import (
	"fmt"
	"reflect"
)

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
		"are spawned with allowPrivilegedEscalation: true and default " +
		"capabilities on top of those required by the test operator " +
		"(NET_ADMIN, NET_RAW)."

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

// merge non-workflow section into workflow
func mergeSectionIntoWorkflow(instance interface{}, workflowStepNum int) {
        spec, ok := instance.(*TempestSpec)
        if !ok {
        fmt.Println("Error, instance is not of type *TempestSpec")
                return
        }

        tRun := spec.TempestRun
        wtRun := &spec.Workflow[workflowStepNum].TempestRun

        tRunReflect := reflect.ValueOf(tRun)
        wtRunReflect := reflect.ValueOf(wtRun).Elem()

	setNonZeroValues(tRunReflect, wtRunReflect, false)
}

func setNonZeroValues(src reflect.Value, dest reflect.Value, is_struct bool) {
        for i := 0; i < src.NumField(); i++ {
                tRunName := src.Type().Field(i).Name
                tRunValue := src.Field(i)
                wtRunValue := dest.FieldByName(tRunName)

                if wtRunValue.IsZero() && !tRunValue.IsZero() {
			if tRunValue.Kind() == reflect.Struct {
				setNonZeroValues(tRunValue, wtRunValue, true)
			} else {
				if is_struct {
					wtRunValue.Set(tRunValue)
				} else {
					tRunPtr := reflect.New(tRunValue.Type())
					tRunPtr.Elem().Set(tRunValue)
					wtRunValue.Set(tRunPtr)
				}
			}
                }
        }
}
