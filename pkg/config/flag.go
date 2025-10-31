package config

import (
	"flag"
	"fmt"
	"reflect"
	"strings"
)

const flagTag = "flag"

// UnmarshalFlags парсит флаг в структуру по тегам
// тег может быть сформирован следующим образом:
// flag:[ключа флага];[стандартное значение];[описание флага]
// Кастомный тип должен реализовывать interface
// FromString(s string) error
func UnmarshalFlags(res any) error {
	m, err := parseFlagsValues(res)
	if err != nil {
		return err
	}

	err = assignValues(res, m)
	if err != nil {
		return err
	}

	return nil
}

// parseFlagsValues парсит значения флагов
func parseFlagsValues(res any) (map[string]*string, error) {
	vo := reflect.ValueOf(res)
	to := reflect.TypeOf(res)

	tagNameToValue := make(map[string]*string, vo.Elem().Kind())

	for i := 0; i < vo.Elem().NumField(); i++ {
		fieldType := to.Elem().Field(i)

		keyName := strings.Split(fieldType.Tag.Get(flagTag), ";")
		if len(keyName) == 0 {
			continue
		}

		flagName := ""
		if len(keyName) > 0 {
			flagName = keyName[0]
		}

		defaultValue := ""
		if len(keyName) > 1 {
			defaultValue = keyName[1]
		}

		description := ""
		if len(keyName) > 2 {
			description = keyName[2]
		}

		f := flag.String(flagName, defaultValue, description)
		if f == nil {
			continue
		}
		tagNameToValue[flagName] = f
	}

	flag.Parse()

	return tagNameToValue, nil
}

// assignValues присваивает значения флагов
func assignValues(res any, flagNameToValue map[string]*string) error {
	vo := reflect.ValueOf(res)
	to := reflect.TypeOf(res)

	for i := 0; i < vo.Elem().NumField(); i++ {
		field := vo.Elem().Field(i)
		fieldType := to.Elem().Field(i)

		keyName := strings.Split(fieldType.Tag.Get(flagTag), ";")
		if len(keyName) == 0 {
			continue
		}

		flagName := ""
		if len(keyName) > 0 {
			flagName = keyName[0]
		}

		value := flagNameToValue[flagName]
		if value == nil {
			return fmt.Errorf("flag %s is nil", flagName)
		}

		if *value == "" {
			continue
		}

		if err := set(field, *value); err != nil {
			return fmt.Errorf("error parse flag %s, %w", flagName, err)
		}
	}

	return nil
}
