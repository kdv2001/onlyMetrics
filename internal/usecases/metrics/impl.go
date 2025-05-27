package metrics

import (
	"context"
	"fmt"

	"github.com/kdv2001/onlyMetrics/internal/domain"
)

type metricStorage interface {
	UpdateGauge(_ context.Context, value domain.MetricValue) error
	UpdateCounter(_ context.Context, value domain.MetricValue) error
}

type UseCases struct {
	metricStorage metricStorage
}

func NewUseCases(metricStorage metricStorage) *UseCases {
	return &UseCases{
		metricStorage: metricStorage,
	}
}

func (uc *UseCases) UpdateMetric(_ context.Context, value domain.MetricValue) error {
	switch value.Type {
	case domain.GaugeMetricType:
		err := uc.metricStorage.UpdateGauge(context.Background(), value)
		if err != nil {
			return fmt.Errorf("error UpdateGauge: %w", err)
		}
	case domain.CounterMetricType:
		err := uc.metricStorage.UpdateCounter(context.Background(), value)
		if err != nil {
			return fmt.Errorf("error UpdateGauge: %w", err)
		}
	}

	return nil
}
