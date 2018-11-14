package main

import (
	"reflect"
	"errors"
)

func setField(strctField reflect.Value, val reflect.Value) error {
	fieldKind := strctField.Kind()

	switch fieldKind {
	case reflect.String:
		if val.Elem().Kind() != reflect.String {
			return errors.New("Invalid field type, expected string")
		}
		strctField.SetString(val.Elem().String())
	case reflect.Int:
		valKind := val.Elem().Kind()
		if valKind != reflect.Float64 && valKind != reflect.Int {
			return errors.New("Invalid field type, expected int")
		}

		floatVal := val.Elem().Float()
		intVal := int64(floatVal)

		strctField.SetInt(intVal)
	case reflect.Bool:
		if val.Elem().Kind() != reflect.Bool {
			return errors.New("Invalid field type, expected bool")
		}
		strctField.SetBool(val.Elem().Bool())
	case reflect.Struct, reflect.Slice:
		if fieldKind == reflect.Struct && val.Elem().Kind() != reflect.Map {
			return errors.New("Invalid field type, expected struct")
		}
		if fieldKind == reflect.Slice && val.Elem().Kind() != reflect.Slice {
			return errors.New("Invalid field type, expected slice")
		}

		in := val.Elem().Interface()
		out := reflect.New(strctField.Type()).Interface()

		i2s(in, out)
		strctField.Set(reflect.ValueOf(out).Elem())
	}

	return nil
}

func i2s(data interface{}, out interface{}) error {
	if reflect.ValueOf(out).Kind() != reflect.Ptr {
		return errors.New("Expected pointer out argument")
	}

	dataVal := reflect.ValueOf(data)
	outVal := reflect.ValueOf(out).Elem()

	if dataVal.Kind() == reflect.Map {
		for _, key := range dataVal.MapKeys() {
			val := dataVal.MapIndex(key)
			strctField := outVal.FieldByName(key.String())

			err := setField(strctField, val)
			if err != nil {
				return err
			}
		}
	}

	if dataVal.Kind() == reflect.Slice {
		if outVal.Kind() != reflect.Slice {
			return errors.New("Invalid field type")
		}

		var inV interface{}
		var outV interface{}
		cap := dataVal.Len()
		sf := reflect.MakeSlice(outVal.Type(), cap, cap)

		for i := 0; i < cap; i++ {
			inV = dataVal.Index(i).Interface()
			outV = reflect.New(sf.Index(i).Type()).Interface()

			i2s(inV, outV)
			sf.Index(i).Set(reflect.ValueOf(outV).Elem())
		}
		outVal.Set(sf)
	}

	out = outVal
	return nil
}
