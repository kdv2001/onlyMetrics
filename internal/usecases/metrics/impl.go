package metrics

import (
	"context"
	"errors"
	"fmt"

	"github.com/kdv2001/onlyMetrics/internal/domain"
)

type metricStorage interface {
	UpdateGauge(_ context.Context, value domain.MetricValue) error
	UpdateCounter(_ context.Context, value domain.MetricValue) error
	GetGaugeValue(_ context.Context, name string) (float64, error)
	GetCounterValue(_ context.Context, name string) (int64, error)
	GetAllValues(_ context.Context) ([]domain.MetricValue, error)
}
type UseCases struct {
	metricStorage metricStorage
}

func NewUseCases(metricStorage metricStorage) *UseCases {
	return &UseCases{
		metricStorage: metricStorage,
	}
}

func (uc *UseCases) UpdateMetric(ctx context.Context, value domain.MetricValue) error {
	switch value.Type {
	case domain.GaugeMetricType:
		err := uc.metricStorage.UpdateGauge(ctx, value)
		if err != nil {
			return fmt.Errorf("error UpdateGauge: %w", err)
		}
	case domain.CounterMetricType:
		err := uc.metricStorage.UpdateCounter(ctx, value)
		if err != nil {
			return fmt.Errorf("error UpdateGauge: %w", err)
		}
	}

	return nil
}

func (uc *UseCases) GetAllMetrics(ctx context.Context) ([]domain.MetricValue, error) {
	return uc.metricStorage.GetAllValues(ctx)
}

func (uc *UseCases) GetMetric(ctx context.Context, value domain.MetricType,
	name string) (domain.MetricValue, error) {
	switch value {
	case domain.GaugeMetricType:
		val, err := uc.metricStorage.GetGaugeValue(ctx, name)
		if err != nil {
			return domain.MetricValue{}, fmt.Errorf("error GetGaugeValue: %w", err)
		}

		return domain.MetricValue{
			Type:       domain.GaugeMetricType,
			Name:       name,
			GaugeValue: val,
		}, nil
	case domain.CounterMetricType:
		val, err := uc.metricStorage.GetCounterValue(ctx, name)
		if err != nil {
			return domain.MetricValue{}, fmt.Errorf("error GetCounterValue: %w", err)
		}
		return domain.MetricValue{
			Type:         domain.CounterMetricType,
			Name:         name,
			CounterValue: val,
		}, nil
	}

	return domain.MetricValue{}, errors.New("unknown metric type")
}
