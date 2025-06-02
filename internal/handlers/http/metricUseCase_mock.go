package http

import (
	"context"

	"github.com/kdv2001/onlyMetrics/internal/domain"
)

type metricUseCaseMock struct {
	err error
}

func (m *metricUseCaseMock) UpdateMetric(ctx context.Context, value domain.MetricValue) error {
	return m.err
}
