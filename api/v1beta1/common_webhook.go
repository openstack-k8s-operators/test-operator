package v1beta1

import (
	"fmt"
	"reflect"
)

const (
	// ErrPrivilegedModeRequired
	ErrPrivilegedModeRequired = "%s.Spec.Privileged is requied in order to successfully " +
		"execute tests with the provided configuration."
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
)

// merge non-workflow section into workflow
func mergeSectionIntoWorkflow(instance interface{}, workflowStepNum int) {
        spec, ok := instance.(*TempestSpec)
        if !ok {
        fmt.Println("Temporary error, instance is not of type *TempestSpec")
                return
        }

        tRun := spec.TempestRun
        wtRun := &spec.Workflow[workflowStepNum].TempestRun

        tRunReflect := reflect.ValueOf(tRun)
        wtRunReflect := reflect.ValueOf(wtRun).Elem()

        for i := 0; i < tRunReflect.NumField(); i++ {
                tRunName := tRunReflect.Type().Field(i).Name
                tRunValue := tRunReflect.Field(i)

                wRunValue := wtRunReflect.FieldByName(tRunName)
                if wRunValue.IsZero() {
                        tRunPtr := reflect.New(tRunValue.Type())
                        tRunPtr.Elem().Set(tRunValue)
                        wRunValue.Set(tRunPtr)
                }
        }
}
