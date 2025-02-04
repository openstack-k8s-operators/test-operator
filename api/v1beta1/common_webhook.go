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

func isEmpty(value interface{}) bool {
	if v, ok := value.(reflect.Value); ok {
		switch v.Kind() {
		case reflect.String, reflect.Map:
			return v.Len() == 0
		case reflect.Ptr, reflect.Interface, reflect.Slice:
			return v.IsNil()
		}
	}
	return false
}

// merge non-workflow section into workflow
func mergeSectionIntoWorkflow(main interface{}, workflow interface{}) {
	mReflect  := reflect.ValueOf(main)
	wReflect := reflect.ValueOf(workflow).Elem()

	for i := 0; i < mReflect.NumField(); i++ {
		name := mReflect.Type().Field(i).Name
		mValue := mReflect.Field(i)
		wValue := wReflect.FieldByName(name)

		fmt.Println("Name: ", name)
		//fmt.Println("M Value: ", mValue)
		//fmt.Println("W Value: ", wValue)
		//fmt.Println("M Kind: ", mValue.Kind())
		//fmt.Println("W Kind: ", wValue.Kind())
		//fmt.Println("M Empty: ", isEmpty(mValue))
		//fmt.Println("W Empty: ", isEmpty(wValue))

		if mValue.Kind() == reflect.Struct {
			switch name {
			case "CommonOptions":
				wValue := wReflect.FieldByName("WorkflowCommonParameters")
				mergeSectionIntoWorkflow(mValue.Interface(), wValue.Addr().Interface())
			case "TempestRun", "TempestconfRun":
				mergeSectionIntoWorkflow(mValue.Interface(), wValue.Addr().Interface())
			case "Resources":
				mergeSectionIntoWorkflow(mValue.Interface(), wValue.Interface())

			}
			continue
		}

		if !wValue.IsValid() {
			continue
		}

		if isEmpty(wValue) && !isEmpty(mValue) {
			if mValue.Kind() == reflect.Map {
				mapCopy := reflect.MakeMap(mValue.Type())
				for _, key := range mValue.MapKeys() {
					value := mValue.MapIndex(key)
					mapCopy.SetMapIndex(key, value)
				}
				wValue = reflect.New(wValue.Type().Elem()).Elem()
				wValue.Set(mapCopy)
			} else if mValue.Kind() == reflect.Slice {
				sliceCopy := reflect.MakeSlice(mValue.Type(), mValue.Len(), mValue.Cap())
				reflect.Copy(sliceCopy, mValue)
				wValue = reflect.New(wValue.Type().Elem()).Elem()
				wValue.Set(sliceCopy)
			} else if wValue.Kind() == reflect.Ptr {
				mPtr := reflect.New(mValue.Type())
				mPtr.Elem().Set(mValue)
				wValue.Set(mPtr)
			} else {
				wValue.Set(mValue)
			}
		}
	}
}
