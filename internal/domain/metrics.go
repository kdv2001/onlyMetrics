package domain

import "fmt"

// MetricType тип метрик.
type MetricType string

const (
	// GaugeMetricType тип "градусник".
	GaugeMetricType MetricType = "gauge"
	// CounterMetricType тип "счетчик"
	CounterMetricType MetricType = "counter"
)

// NewMetricTypeFromString конструктор типа метрики.
func NewMetricTypeFromString(s string) (MetricType, error) {
	switch s {
	case string(GaugeMetricType):
		return GaugeMetricType, nil

	case string(CounterMetricType):
		return CounterMetricType, nil
	}

	return "", fmt.Errorf("unknown metric type")
}

// String возвращает строковое значение типа метрики.
func (mt MetricType) String() string {
	return string(mt)
}

// MetricValue значение метрики, в зависимости от типа будет заполнено только одно из полей.
type MetricValue struct {
	Type         MetricType
	Name         string
	CounterValue int64
	GaugeValue   float64
}
