package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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

// CollectMetric обработчик сбора метрик из URL параметров
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

type metric struct {
	ID    string   `json:"id"`              // Имя метрики
	MType string   `json:"type"`            // параметр, принимающий значение gauge или counter
	Delta *int64   `json:"delta,omitempty"` // Значение метрики в случае передачи counter
	Value *float64 `json:"value,omitempty"` // Значение метрики в случае передачи gauge
}

// CollectBodyMetric обработчик сбора метрик из тела запроса
func (h *Handlers) CollectBodyMetric(w http.ResponseWriter, r *http.Request) {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error reading body: %v", err), http.StatusBadRequest)
		return
	}

	var parsedMetric metric
	if err = json.Unmarshal(bodyBytes, &parsedMetric); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var v domain.MetricValue
	mType, err := domain.NewMetricTypeFromString(parsedMetric.MType)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	switch mType {
	case domain.GaugeMetricType:
		var resValue float64
		if parsedMetric.Value != nil {
			resValue = *parsedMetric.Value
		}

		v = domain.MetricValue{
			Type:       mType,
			Name:       parsedMetric.ID,
			GaugeValue: resValue,
		}
	case domain.CounterMetricType:
		var resValue int64
		if parsedMetric.Delta != nil {
			resValue = *parsedMetric.Delta
		}

		if parsedMetric.Delta == nil {
			http.Error(w, "counter value is empty", http.StatusBadRequest)
			return
		}

		v = domain.MetricValue{
			Type:         mType,
			Name:         parsedMetric.ID,
			CounterValue: resValue,
		}
	}

	err = h.metricUseCases.UpdateMetric(r.Context(), v)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// GetMetric обработчик для получения метрик
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

// GetBodyMetric обработчик для получения метрик из тела запроса
func (h *Handlers) GetBodyMetric(w http.ResponseWriter, r *http.Request) {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error reading body: %v", err), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var parsedMetric metric
	if err = json.Unmarshal(bodyBytes, &parsedMetric); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	mType, err := domain.NewMetricTypeFromString(parsedMetric.MType)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	val, err := h.metricUseCases.GetMetric(r.Context(), mType, parsedMetric.ID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var v metric
	switch val.Type {
	case domain.GaugeMetricType:
		v = metric{
			ID:    val.Name,
			MType: val.Type.String(),
			Value: &val.GaugeValue,
		}
	case domain.CounterMetricType:
		v = metric{
			ID:    val.Name,
			MType: val.Type.String(),
			Delta: &val.CounterValue,
		}
	}

	b, err := json.Marshal(v)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(b)

	w.WriteHeader(http.StatusOK)
}
