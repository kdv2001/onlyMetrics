package http

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/kdv2001/onlyMetrics/internal/domain"
)

type useCases interface {
	UpdateMetric(ctx context.Context, value domain.MetricValue) error
	GetMetric(ctx context.Context, value domain.MetricType,
		name string) (domain.MetricValue, error)
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
	t, err := domain.NewMetricTypeFromString(chi.URLParam(r, MetricTypePathKey))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var v domain.MetricValue
	switch t {
	case domain.GaugeMetricType:
		mValue, err := strconv.ParseFloat(chi.URLParam(r, ValuePathKey), 64)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		v = domain.MetricValue{
			Type:       t,
			Name:       chi.URLParam(r, MetricNamePathKey),
			GaugeValue: mValue,
		}
	case domain.CounterMetricType:
		mValue, err := strconv.ParseInt(chi.URLParam(r, ValuePathKey), 10, 64)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		v = domain.MetricValue{
			Type:         t,
			Name:         chi.URLParam(r, MetricNamePathKey),
			CounterValue: mValue,
		}
	}

	err = h.metricUseCases.UpdateMetric(r.Context(), v)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handlers) GetMetric(w http.ResponseWriter, r *http.Request) {
	t, err := domain.NewMetricTypeFromString(chi.URLParam(r, MetricTypePathKey))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	val, err := h.metricUseCases.GetMetric(r.Context(), t, chi.URLParam(r, MetricNamePathKey))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	switch val.Type {
	case domain.GaugeMetricType:
		_, _ = w.Write([]byte(fmt.Sprint(val.GaugeValue)))
	case domain.CounterMetricType:
		_, _ = w.Write([]byte(fmt.Sprint(val.CounterValue)))
	}
}
