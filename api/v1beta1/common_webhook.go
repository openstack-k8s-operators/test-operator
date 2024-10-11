package v1beta1

import (
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

func StringInSlice(stringValue string, stringList []string) bool {
	for _, v := range stringList {
		if v == stringValue {
			return true
		}
	}

	return false
}

func PrivilegedRequired(instance interface{}, requiredFields []string) bool {
	v := reflect.ValueOf(instance)
	workflowField := "Workflow"

	for i := 0; i < v.NumField(); i++ {
		fieldValue := v.Field(i)
		fieldType := v.Type().Field(i)

		tempestlog.Info(fieldType.Name)
		tempestlog.Info(fieldValue.Kind().String())

		// Recursively check whether there is a non empty value in a structure
		if fieldValue.Kind() == reflect.Struct {
			if PrivilegedRequired(fieldValue.Interface(), requiredFields) {
				return true
			}
		}

		// Check if workflow section contains field requiring privileged mode
		if fieldType.Name == workflowField {
			for i := 0; i < fieldValue.Len(); i++ {
				if PrivilegedRequired(fieldValue.Index(i).Interface(), requiredFields) {
					return true
				}
			}
		}

		// If we are iterating over a field that does not require a privileged mode
		if !StringInSlice(fieldType.Name, requiredFields) {
			continue
		}

		switch fieldValue.Kind() {
		case reflect.String, reflect.Array, reflect.Slice:
			if fieldValue.Len() > 0 {
				return true
			}
		case reflect.Pointer:
			if fieldValue.IsNil() {
				return true
			}
		}
	}

	return false
}
