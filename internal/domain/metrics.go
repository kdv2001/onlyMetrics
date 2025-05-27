package domain

import "fmt"

type MetricType string

const (
	GaugeMetricType   MetricType = "gauge"
	CounterMetricType MetricType = "counter"
)

func NewMetricTypeFromString(s string) (MetricType, error) {
	switch s {
	case string(GaugeMetricType):
		return GaugeMetricType, nil

	case string(CounterMetricType):
		return CounterMetricType, nil
	}

	return "", fmt.Errorf("unknown metric type")
}

func (mt MetricType) String() string {
	return string(mt)
}

type MetricValue struct {
	Type  MetricType
	Name  string
	Value float64
}
