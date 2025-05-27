package http

import (
	"context"
	"net/http"
	"strconv"

	"github.com/kdv2001/onlyMetrics/internal/domain"
)

type useCases interface {
	UpdateMetric(ctx context.Context, value domain.MetricValue) error
}

type Handlers struct {
	metricUseCases useCases
}

func NewHandlers(useCases useCases) *Handlers {
	return &Handlers{
		metricUseCases: useCases,
	}
}

const (
	MetricTypePathKey = "metricType"
	MetricNamePathKey = "metricName"
	ValuePathKey      = "value"
)

func (h *Handlers) CollectMetric(w http.ResponseWriter, r *http.Request) {
	t, err := domain.NewMetricTypeFromString(r.PathValue(MetricTypePathKey))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	mValue, err := strconv.ParseFloat(r.PathValue(ValuePathKey), 64)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	v := domain.MetricValue{
		Type:  t,
		Name:  r.PathValue(MetricNamePathKey),
		Value: mValue,
	}

	err = h.metricUseCases.UpdateMetric(r.Context(), v)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
