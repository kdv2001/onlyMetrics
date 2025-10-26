package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"time"
)

// URL тип для адреса
type URL struct {
	url.URL
}

// FromString преобразует строку в адресный тип
func (u *URL) FromString(s string) error {
	if u == nil {
		return errors.New("nil URL")
	}

	serverAddr := url.URL{
		Host: s,
	}

	u.URL = serverAddr

	return nil
}

// IsZero возвращает признак пустого значения
func (u *URL) IsZero() bool {
	if u == nil {
		return true
	}

	return u.String() == ""
}

// UnmarshalJSON десериализует во временой тип
func (u *URL) UnmarshalJSON(b []byte) error {
	if u == nil {
		return fmt.Errorf("duration cannot be nil")
	}

	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch value := v.(type) {
	case string:
		return u.FromString(value)
	default:
		return errors.New("invalid Duration")
	}
}

// AsURL преобразует в адресный тип
func (u *URL) AsURL() url.URL {
	if u == nil {
		return url.URL{}
	}
	return u.URL
}

// Duration временной тип
type Duration struct {
	time.Duration
}

// FromString парсит временной тип из строки
func (d *Duration) FromString(s string) error {
	tmp, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return err
	}

	if tmp <= 0 {
		return fmt.Errorf("invalid duration: %d", tmp)
	}

	d.Duration = time.Duration(tmp) * time.Second
	return nil
}

// IsZero возвращает признак пустого значения
func (d *Duration) IsZero() bool {
	if d == nil {
		return true
	}

	return d.Duration == 0
}

// UnmarshalJSON десериализует во временой тип
func (d *Duration) UnmarshalJSON(b []byte) error {
	if d == nil {
		return fmt.Errorf("duration cannot be nil")
	}

	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch value := v.(type) {
	case float64:
		d.Duration = time.Duration(value)
		return nil
	case string:
		tmp, err := time.ParseDuration(value)
		if err != nil {
			return err
		}
		d.Duration = tmp
		return nil
	default:
		return errors.New("invalid Duration")
	}
}

// AsTimeDuration возвращает временной тип из стандартной библиотеки
func (d *Duration) AsTimeDuration() time.Duration {
	if d == nil {
		return time.Duration(0)
	}

	return d.Duration
}
