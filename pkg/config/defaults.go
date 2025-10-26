package config

import (
	"reflect"
)

const defaultTag = "default"

// SetDefaultValues устанавливает стандартные значения переменных если они не заданы
// Стандартное значение задается через тег default
// Кастомный тип должен реализовывать interface
// IsZero() bool
// FromString(s string) error
func SetDefaultValues(res any) error {
	vo := reflect.ValueOf(res)
	to := reflect.TypeOf(res)

	for i := 0; i < vo.Elem().NumField(); i++ {
		field := vo.Elem().Field(i)
		fieldType := to.Elem().Field(i)

		v, ok := field.Interface().(interface {
			IsZero() bool
		})

		if ok {
			if !v.IsZero() {
				continue
			}
		} else {
			if !field.IsZero() {
				continue
			}
		}

		value := fieldType.Tag.Get(defaultTag)
		if value == "" {
			continue
		}

		if err := set(field, value); err != nil {
			return err
		}
	}

	return nil
}
