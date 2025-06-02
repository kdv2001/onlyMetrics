package metrics

import (
	"context"

	"github.com/kdv2001/onlyMetrics/internal/domain"
)

type mockMetric struct {
	err error
}

func (m *mockMetric) UpdateGauge(_ context.Context, value domain.MetricValue) error {
	return m.err
}

func (m *mockMetric) UpdateCounter(_ context.Context, value domain.MetricValue) error {
	return m.err
}
