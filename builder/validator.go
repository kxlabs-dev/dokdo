package builder

import (
	"fmt"
	"reflect"
	"strings"
)

func ValidateParams(params interface{}, info *TypeInfo) error {
	if params == nil {
		return nil
	}

	rv := reflect.ValueOf(params)
	if rv.Kind() == reflect.Map {
		return fmt.Errorf("invalid params: map type is not allowed, use struct")
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
				return fmt.Errorf("type mismatch: field %s expected []struct got %s",
					field.Name, goFieldType.String())
			}
			elemType := goFieldType.Elem()
			if elemType.Kind() != reflect.Struct {
				return fmt.Errorf("type mismatch: field %s expected []struct got []%s",
					field.Name, elemType.String())
			}
			for _, anonField := range field.AnonFields {
				for i := 0; i < elemType.NumField(); i++ {
					ef := elemType.Field(i)
					if anonField.Name == ef.Name {
						if ef.Type.String() != anonField.TypeStr {
							return fmt.Errorf("type mismatch: field %s.%s expected %s got %s",
								field.Name, anonField.Name, anonField.TypeStr, ef.Type.String())
						}
						break
					}
				}
			}
			continue
		}

		if goFieldType.String() != field.TypeStr {
			return fmt.Errorf("type mismatch: field %s expected %s got %s",
				field.Name, field.TypeStr, goFieldType.String())
		}

		if !field.Nullable {
			kind := goFieldVal.Kind()
			if kind == reflect.Ptr || kind == reflect.Interface {
				if goFieldVal.IsNil() {
					return fmt.Errorf("required field missing: %s", field.Name)
				}
			}
		}
	}

	return nil
}
