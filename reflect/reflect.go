// This package provides some punk-rock reflection which is not in the stdlib.
package reflect

import (
	r "reflect"
)

const (
	prefixSeperator = "_"
)

// StructToMap converts a struct to map. The object's default key string
// is the struct field name but can be specified in the struct field's
// tag value. The "cql" key in the struct field's tag value is the key
// name. Examples:
//
//   // Field appears in the resulting map as key "myName".
//   Field int `cql:"myName"`
//
//   // Field appears in the resulting as key "Field"
//   Field int
//
//   // Field appears in the resulting map as key "myName"
//   Field int "myName"
func StructToMap(val interface{}) (map[string]interface{}, bool) {
	// indirect so function works with both structs and pointers to them
	structVal := r.Indirect(r.ValueOf(val))
	kind := structVal.Kind()
	if kind != r.Struct {
		return nil, false
	}
	structFields := cachedTypeFields(structVal.Type())
	mapVal := make(map[string]interface{}, len(structFields))
	for _, info := range structFields {
		field := fieldByIndex(structVal, info.index)

		// If field is ptr then ensure not nil and get elem
		if field.Kind() == r.Ptr {
			if field.IsNil() {
				if field.CanSet() {
					field.Set(r.New(field.Type().Elem()))
				} else {
					continue
				}
			}
		}

		var isStruct bool
		if field.Kind() == r.Struct {
			isStruct = true
		} else if field.Kind() == r.Ptr && field.Elem().Kind() == r.Struct {
			isStruct = true
		}

		// If field is a struct then flatten
		if isStruct && info.flatten {
			childMap, ok := StructToMap(field.Interface())
			if ok {
				for ck, cv := range childMap {
					mapVal[info.name+prefixSeperator+ck] = cv
				}
			}
		} else {
			mapVal[info.name] = field.Interface()
		}
	}
	return mapVal, true
}

// MapToStruct converts a map to a struct. It is the inverse of the StructToMap
// function. For details see StructToMap.
func MapToStruct(m map[string]interface{}, struc interface{}) error {
	val := r.Indirect(r.ValueOf(struc))
	fieldsMap := getFieldsMap(val)

	for k, v := range m {
		if info, ok := fieldsMap[k]; ok {
			structField := fieldByIndex(val, info.index)
			if structField.Type().Name() == r.TypeOf(v).Name() {
				structField.Set(r.ValueOf(v))
			}
		}
	}
	return nil
}

func getFieldsMap(v r.Value) map[string]field {
	v = r.Indirect(v)
	fieldsMap := make(map[string]field)
	structFields := cachedTypeFields(v.Type())

	// Create fields map for faster lookup
	for _, field := range structFields {
		fieldValue := fieldByIndex(v, field.index)

		if fieldValue.Kind() == r.Ptr && field.flatten {
			if fieldValue.IsNil() {
				if fieldValue.CanSet() {
					fieldValue.Set(r.New(fieldValue.Type().Elem()))
				} else {
					continue
				}
			}
		}

		var isStruct bool
		if fieldValue.Kind() == r.Struct {
			isStruct = true
		} else if fieldValue.Kind() == r.Ptr && fieldValue.Elem().Kind() == r.Struct {
			isStruct = true
		}

		// If field is a struct then flatten
		if isStruct && field.flatten {
			if fieldValue.Kind() == r.Ptr {
				fieldValue = fieldValue.Elem()
			}

			for ck, cv := range getFieldsMap(fieldValue) {
				fieldsMap[field.name+prefixSeperator+ck] = cv
			}
		} else {
			fieldsMap[field.name] = field
		}
	}

	return fieldsMap
}

func fieldByIndex(v r.Value, index []int) r.Value {
	for _, i := range index {
		if v.Kind() == r.Ptr {
			if v.IsNil() {
				if v.CanSet() {
					v.Set(r.New(v.Type().Elem()))
				} else {
					return r.Value{}
				}
			}
			v = v.Elem()
		}
		v = v.Field(i)
	}

	return v
}
