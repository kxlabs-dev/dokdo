package builder

import (
	"reflect"
	"strings"
)

func ValidateParams(params interface{}, info *TypeInfo) error {
	if params == nil {
		return nil
	}

	rv := reflect.ValueOf(params)
	// external/ 로 internal을 우회할 수 없으므로 이 체크는 모듈 내부 코드가
	// ValidateParams를 검증 없이 직접 호출하는 실수를 막기 위한 방어 로직이다.
	if rv.Kind() == reflect.Map {
		return &RuntimeError{Message: "map type is not allowed, use struct"}
	}

	for rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return nil
		}
		rv = rv.Elem()
	}

	if rv.Kind() != reflect.Struct {
		return nil
	}

	rt := rv.Type()

	for _, field := range info.Fields {
		var found bool
		var goFieldType reflect.Type
		var goFieldVal reflect.Value

		for i := 0; i < rt.NumField(); i++ {
			f := rt.Field(i)
			if strings.EqualFold(field.Name, f.Name) {
				found = true
				goFieldType = f.Type
				goFieldVal = rv.Field(i)
				break
			}
		}

		if !found {
			continue
		}

		if field.IsNamedSlice {
			if goFieldType.Kind() != reflect.Slice {
				return &TypeMismatchError{Field: field.Name, Expected: "[]" + field.SliceElemType, Got: goFieldType.String()}
			}
			elemType := goFieldType.Elem()
			if elemType.Kind() != reflect.Struct {
				return &TypeMismatchError{Field: field.Name, Expected: "[]" + field.SliceElemType, Got: "[]" + elemType.String()}
			}
			if err := validateStructFields(field.Name, elemType, field.SliceFields); err != nil {
				return err
			}
			continue
		}

		if field.IsNamedStruct {
			if goFieldType.Kind() != reflect.Struct {
				return &TypeMismatchError{Field: field.Name, Expected: field.StructElemType, Got: goFieldType.String()}
			}
			if err := validateStructFields(field.Name, goFieldType, field.StructFields); err != nil {
				return err
			}
			continue
		}

		cmpType := goFieldType
		if field.Nullable && cmpType.Kind() == reflect.Ptr {
			cmpType = cmpType.Elem()
		}
		if cmpType.String() != field.TypeStr {
			return &TypeMismatchError{Field: field.Name, Expected: field.TypeStr, Got: goFieldType.String()}
		}

		if !field.Nullable {
			kind := goFieldVal.Kind()
			if kind == reflect.Ptr || kind == reflect.Interface {
				if goFieldVal.IsNil() {
					return &RequiredFieldError{Field: field.Name}
				}
			}
		}
	}

	return nil
}

func validateStructFields(prefix string, rt reflect.Type, fields []FieldInfo) error {
	for _, sf := range fields {
		var found bool
		var sfType reflect.Type
		for i := 0; i < rt.NumField(); i++ {
			if strings.EqualFold(sf.Name, rt.Field(i).Name) {
				found = true
				sfType = rt.Field(i).Type
				break
			}
		}
		if !found {
			continue
		}
		if sf.IsNamedSlice {
			if sfType.Kind() != reflect.Slice {
				return &TypeMismatchError{Field: prefix + "." + sf.Name, Expected: "[]" + sf.SliceElemType, Got: sfType.String()}
			}
			elemType := sfType.Elem()
			if elemType.Kind() != reflect.Struct {
				return &TypeMismatchError{Field: prefix + "." + sf.Name, Expected: "[]" + sf.SliceElemType, Got: "[]" + elemType.String()}
			}
			if err := validateStructFields(prefix+"."+sf.Name, elemType, sf.SliceFields); err != nil {
				return err
			}
			continue
		}
		if sf.IsNamedStruct {
			if sfType.Kind() != reflect.Struct {
				return &TypeMismatchError{Field: prefix + "." + sf.Name, Expected: sf.StructElemType, Got: sfType.String()}
			}
			if err := validateStructFields(prefix+"."+sf.Name, sfType, sf.StructFields); err != nil {
				return err
			}
			continue
		}
		cmpType := sfType
		if sf.Nullable && cmpType.Kind() == reflect.Ptr {
			cmpType = cmpType.Elem()
		}
		if cmpType.String() != sf.TypeStr {
			return &TypeMismatchError{Field: prefix + "." + sf.Name, Expected: sf.TypeStr, Got: sfType.String()}
		}
	}
	return nil
}
