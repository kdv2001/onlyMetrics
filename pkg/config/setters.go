package config

import (
	"fmt"
	"reflect"
	"strconv"
)

func set(field reflect.Value, value string) error {
	switch field.Type().Kind() {
	case reflect.Pointer:
		v, ok := field.Interface().(interface {
			FromString(s string) error
		})
		if !ok {
			return fmt.Errorf(`field "%s" is not a pointer`, field.String())
		}

		err := v.FromString(value)
		if err != nil {
			return fmt.Errorf("error parse %s, %w", field.String(), err)
		}

		field.Set(reflect.ValueOf(v))
		return nil
	case reflect.String:
		field.SetString(value)
	case reflect.Bool:
		if err := setBool(field, value); err != nil {
			return fmt.Errorf("error parse %s, %w", field.String(), err)
		}
		return nil
	case reflect.Int64, reflect.Int:
		if value == "" {
			return fmt.Errorf(`field "%s" is empty`, field.String())
		}

		intEnvValue, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("failed to parse %s: %w", value, err)
		}

		field.SetInt(intEnvValue)
		return nil
	default:
		return fmt.Errorf("unprocessed type %s", field.Type().Kind())
	}

	return nil
}

func setBool(value reflect.Value, val string) error {
	if val == "" {
		return fmt.Errorf(`field "%s" is empty`, value.String())
	}

	boolValue, err := strconv.ParseBool(val)
	if err != nil {
		return fmt.Errorf("failed to parse %s: %w", val, err)
	}
	value.SetBool(boolValue)

	return nil
}
