package config

import (
	"fmt"
	"os"
	"reflect"
)

const envTag = "env"

// UnmarshalEnv парсит переменные окружения в структуру
// Имя переменой задается через тег env
// Кастомный тип должен реализовывать interface
// FromString(s string) error
func UnmarshalEnv(res any) error {
	vo := reflect.ValueOf(res)
	to := reflect.TypeOf(res)

	for i := 0; i < vo.Elem().NumField(); i++ {
		field := vo.Elem().Field(i)
		fieldType := to.Elem().Field(i)

		keyName := fieldType.Tag.Get(envTag)
		envValue, exist := os.LookupEnv(keyName)
		if !exist {
			continue
		}

		if envValue == "" {
			return fmt.Errorf("environment variable %s is empty", keyName)
		}

		if err := set(field, envValue); err != nil {
			return fmt.Errorf("environment variable %s is invalid: %s", keyName, err)
		}
	}

	return nil
}
