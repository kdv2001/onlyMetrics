package http

import (
	"context"

	"github.com/kdv2001/onlyMetrics/internal/domain"
)

type metricUseCaseMock struct {
	value domain.MetricValue
	err   error
}

func (m *metricUseCaseMock) UpdateMetric(ctx context.Context, value domain.MetricValue) error {
	return m.err
}

func (m *metricUseCaseMock) GetMetric(_ context.Context, _ domain.MetricType,
	_ string) (domain.MetricValue, error) {
	return m.value, m.err
}

func (m *metricUseCaseMock) GetAllMetrics(ctx context.Context) ([]domain.MetricValue, error) {
	return nil, nil
}
