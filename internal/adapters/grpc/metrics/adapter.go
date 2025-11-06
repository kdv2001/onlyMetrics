package metrics

import (
	"fmt"

	"github.com/kdv2001/onlyMetrics/internal/domain"
	pb "github.com/kdv2001/onlyMetrics/internal/gen/protogen/only_metrics/grpc"
)

// DomainMetricToPB конвертирует доменную структуру в прото
func DomainMetricToPB(value domain.MetricValue) *pb.Metric {
	switch value.Type {
	case domain.GaugeMetricType:
		m := &pb.Metric{}
		m.SetType(pb.MetricType_GAUGE_METRIC_TYPE)
		m.SetName(value.Name)
		m.SetGaugeValue(value.GaugeValue)

		return m
	case domain.CounterMetricType:
		m := &pb.Metric{}
		m.SetType(pb.MetricType_COUNTER_METRIC_TYPE)
		m.SetName(value.Name)
		m.SetCounterValue(value.CounterValue)
		return m
	}

	return nil
}

// PBToDomain конвертирует прото структуру в доменную
func PBToDomain(value *pb.Metric) (domain.MetricValue, error) {
	switch value.GetType() {
	case pb.MetricType_GAUGE_METRIC_TYPE:
		return domain.MetricValue{
			Type:       domain.GaugeMetricType,
			Name:       value.GetName(),
			GaugeValue: value.GetGaugeValue(),
		}, nil
	case pb.MetricType_COUNTER_METRIC_TYPE:
		return domain.MetricValue{
			Type:         domain.CounterMetricType,
			Name:         value.GetName(),
			CounterValue: value.GetCounterValue(),
		}, nil
	}

	return domain.MetricValue{}, fmt.Errorf("unknown metric type: %w", domain.ErrInvalidArgument)
}

// PBToDomainType конвертирует прото тип метрики в доменный
func PBToDomainType(value pb.MetricType) (domain.MetricType, error) {
	switch value {
	case pb.MetricType_GAUGE_METRIC_TYPE:
		return domain.GaugeMetricType, nil
	case pb.MetricType_COUNTER_METRIC_TYPE:
		return domain.CounterMetricType, nil
	}

	return "", fmt.Errorf("unknown metric type: %w", domain.ErrInvalidArgument)
}
