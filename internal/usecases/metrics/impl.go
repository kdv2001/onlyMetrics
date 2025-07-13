package metrics

import (
	"context"
	"errors"
	"fmt"

	"github.com/kdv2001/onlyMetrics/internal/domain"
)

type MetricStorage interface {
	UpdateGauge(ctx context.Context, value domain.MetricValue) error
	UpdateCounter(ctx context.Context, value domain.MetricValue) error
	GetGaugeValue(ctx context.Context, name string) (float64, error)
	GetCounterValue(ctx context.Context, name string) (int64, error)
	GetAllValues(ctx context.Context) ([]domain.MetricValue, error)
	Ping(ctx context.Context) error
}
type UseCases struct {
	metricStorage MetricStorage
}

func NewUseCases(metricStorage MetricStorage) *UseCases {
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

// Ping ...
func (uc *UseCases) Ping(ctx context.Context) error {
	return uc.metricStorage.Ping(ctx)
}
