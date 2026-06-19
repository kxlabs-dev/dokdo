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

		if field.IsAnonSlice {
			if goFieldType.Kind() != reflect.Slice {
				return &TypeMismatchError{Field: field.Name, Expected: "[]struct", Got: goFieldType.String()}
			}
			elemType := goFieldType.Elem()
			if elemType.Kind() != reflect.Struct {
				return &TypeMismatchError{Field: field.Name, Expected: "[]struct", Got: "[]" + elemType.String()}
			}
			for _, anonField := range field.AnonFields {
				for i := 0; i < elemType.NumField(); i++ {
					ef := elemType.Field(i)
					if anonField.Name == ef.Name {
						if ef.Type.String() != anonField.TypeStr {
							return &TypeMismatchError{
								Field:    field.Name + "." + anonField.Name,
								Expected: anonField.TypeStr,
								Got:      ef.Type.String(),
							}
						}
						break
					}
				}
			}
			continue
		}

		if goFieldType.String() != field.TypeStr {
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
