package metrics

import (
	"context"

	"github.com/kdv2001/onlyMetrics/internal/domain"
)

type mockMetric struct {
	err          error
	counterValue int64
	gaugeValue   float64
}

func (m *mockMetric) UpdateGauge(_ context.Context, value domain.MetricValue) error {
	return m.err
}

func (m *mockMetric) UpdateCounter(_ context.Context, value domain.MetricValue) error {
	return m.err
}

func (m *mockMetric) GetGaugeValue(_ context.Context, name string) (float64, error) {
	return m.gaugeValue, m.err
}

func (m *mockMetric) GetCounterValue(_ context.Context, name string) (int64, error) {
	return m.counterValue, m.err
}

func (m *mockMetric) GetAllValues(_ context.Context) ([]domain.MetricValue, error) {
	return nil, m.err
}

func (m *mockMetric) UpdateMetrics(ctx context.Context, metrics []domain.MetricValue) error {
	return m.err
}

func (m *mockMetric) Ping(_ context.Context) error {
	return nil
}
