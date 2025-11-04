package metrics

import (
	"fmt"

	"github.com/kdv2001/onlyMetrics/internal/domain"
	pb "github.com/kdv2001/onlyMetrics/internal/gen/protogen/only_metrics/grpc"
)

// DomainToPB конвертирует доменную структуру в прото
func DomainToPB(value domain.MetricValue) *pb.Metric {
	switch value.Type {
	case domain.GaugeMetricType:
		return &pb.Metric{
			Type:       pb.MetricType_GAUGE_METRIC_TYPE,
			Name:       value.Name,
			GaugeValue: value.GaugeValue,
		}
	case domain.CounterMetricType:
		return &pb.Metric{
			Type:         pb.MetricType_COUNTER_METRIC_TYPE,
			Name:         value.Name,
			CounterValue: value.CounterValue,
		}
	}

	return nil
}

// PBToDomain конвертирует прото структуру в доменную
func PBToDomain(value *pb.Metric) (domain.MetricValue, error) {
	switch value.Type {
	case pb.MetricType_GAUGE_METRIC_TYPE:
		return domain.MetricValue{
			Type:       domain.GaugeMetricType,
			Name:       value.Name,
			GaugeValue: value.GaugeValue,
		}, nil
	case pb.MetricType_COUNTER_METRIC_TYPE:
		return domain.MetricValue{
			Type:         domain.CounterMetricType,
			Name:         value.Name,
			CounterValue: value.CounterValue,
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
