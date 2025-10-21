// Package http предоставляет http обработчики для сбора метрик и их последующего предоставления клиенту.
package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/kdv2001/onlyMetrics/internal/domain"
	"github.com/kdv2001/onlyMetrics/pkg/logger"
)

type useCases interface {
	UpdateMetric(ctx context.Context, value domain.MetricValue) error
	GetMetric(ctx context.Context, value domain.MetricType,
		name string) (domain.MetricValue, error)
	GetAllMetrics(ctx context.Context) ([]domain.MetricValue, error)
	Ping(ctx context.Context) error
	UpdateMetrics(ctx context.Context, metrics []domain.MetricValue) error
}

//	@Title			onlyMetric API
//	@Description	Сервис для сбора значений метрик с агентов.
//	@Version		1.0

//	@Contact.email	support@onlyMetrics.io

//	@BasePath	/
//	@Host		localhost:8080

// Handlers http обработчики для сбора метрик и их последующего предоставления клиенту.
type Handlers struct {
	metricUseCases useCases
}

// NewHandlers создает объект http обработчиков для сбора метрик и их последующего предоставления клиенту.
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

// CollectMetric обработчик сбора метрик из URL параметров.
//
//	@Summary		collect metric
//	@Description	collect metric from query
//	@Tags			metric
//	@Accept			json
//	@Produce		plain
//	@Param			metricType	query		string	true	"metricType"
//	@Param			value		query		float64	true	"value"
//	@Param			metricName	query		string	true	"metricName"
//	@Success		200			{object}	string
//	@Failure		400			{object}	string
//	@Failure		404			{object}	string
//	@Failure		500			{object}	string
//	@Router			/update/{metricType}/{metricName}/{value} [post]
func (h *Handlers) CollectMetric(w http.ResponseWriter, r *http.Request) {
	t, err := domain.NewMetricTypeFromString(chi.URLParam(r, MetricTypePathKey))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var v domain.MetricValue
	switch t {
	case domain.GaugeMetricType:
		mValue, iErr := strconv.ParseFloat(chi.URLParam(r, ValuePathKey), 64)
		if iErr != nil {
			http.Error(w, iErr.Error(), http.StatusBadRequest)
			return
		}
		v = domain.MetricValue{
			Type:       t,
			Name:       chi.URLParam(r, MetricNamePathKey),
			GaugeValue: mValue,
		}
	case domain.CounterMetricType:
		mValue, iErr := strconv.ParseInt(chi.URLParam(r, ValuePathKey), 10, 64)
		if iErr != nil {
			http.Error(w, iErr.Error(), http.StatusBadRequest)
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
		switch {
		case errors.Is(err, domain.ErrResourceIsLocked):
			w.WriteHeader(http.StatusLocked)
			return
		}

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

type metric struct {
	Delta *int64   `json:"delta,omitempty"` // Значение метрики в случае передачи counter
	Value *float64 `json:"value,omitempty"` // Значение метрики в случае передачи gauge
	ID    string   `json:"id"`              // Имя метрики
	MType string   `json:"type"`            // параметр, принимающий значение gauge или counter
}

// CollectBodyMetric обработчик сбора метрик из тела запроса.
//
//	@Summary		collect metric
//	@Description	collect metric from body
//	@Tags			metric
//	@Accept			json
//	@Produce		plain
//	@Param			metric	body		http.metric	true	"metric"
//	@Success		200		{object}	string
//	@Failure		400		{object}	string
//	@Failure		423		{object}	string
//	@Failure		500		{object}	string
//	@Router			/update [post]
func (h *Handlers) CollectBodyMetric(w http.ResponseWriter, r *http.Request) {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error reading body: %v", err), http.StatusBadRequest)
		return
	}

	logger.Infof(r.Context(), "CollectBodyMetric: %s", string(bodyBytes))
	var parsedMetric metric
	if err = json.Unmarshal(bodyBytes, &parsedMetric); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	v, err := metricToDomain(parsedMetric)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = h.metricUseCases.UpdateMetric(r.Context(), v)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrResourceIsLocked):
			w.WriteHeader(http.StatusLocked)
			return
		}

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// GetMetric обработчик для получения метрики.
//
//	@Summary		get metric
//	@Description	get metric
//	@Tags			metric
//	@Accept			json
//	@Produce		plain
//	@Param			metricType	query		string	true	"metricType"
//	@Param			metricName	query		string	true	"metricName"
//	@Success		200			{object}	string
//	@Failure		400			{object}	string
//	@Failure		423			{object}	string
//	@Failure		500			{object}	string
//	@Router			/value/{metricType}/{metricName}  [get]
func (h *Handlers) GetMetric(w http.ResponseWriter, r *http.Request) {
	t, err := domain.NewMetricTypeFromString(chi.URLParam(r, MetricTypePathKey))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	val, err := h.metricUseCases.GetMetric(r.Context(), t, chi.URLParam(r, MetricNamePathKey))
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrResourceIsLocked):
			w.WriteHeader(http.StatusLocked)
			return
		case errors.Is(err, domain.ErrNotFound):
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

// GetAllMetric обработчик для получения всех метрик.
//
//	@Summary		get  all metric
//	@Description	get all metric
//	@Tags			metric
//	@Accept			plain
//	@Produce		plain
//	@Success		200	{string}	string
//	@Failure		400	{object}	string
//	@Failure		423	{object}	string
//	@Failure		500	{object}	string
//	@Router			/  [get]
func (h *Handlers) GetAllMetric(w http.ResponseWriter, r *http.Request) {
	values, err := h.metricUseCases.GetAllMetrics(r.Context())
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrResourceIsLocked):
			w.WriteHeader(http.StatusLocked)
			return
		case errors.Is(err, domain.ErrNotFound):
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		http.Error(w, fmt.Sprintf("Error GetAllMetrics: %v", err), http.StatusBadRequest)
		return
	}

	resStrs := make([]string, 0, len(values))
	for _, v := range values {
		switch v.Type {
		case domain.GaugeMetricType:
			resStrs = append(resStrs,
				fmt.Sprintf("<br>%s %f</br>", v.Name, v.GaugeValue),
			)
		default:
			resStrs = append(resStrs,
				fmt.Sprintf("<br>%s %d</br>", v.Name, v.CounterValue),
			)
		}
	}

	w.Header().Set(ContentType, TextHTML)
	strings.Join(resStrs, "\n")
	resSTR := "<html><body>" + strings.Join(resStrs, "") + "</body></html>"
	_, err = w.Write([]byte(resSTR))
	if err != nil {
		http.Error(w, fmt.Sprintf("Error write response: %v", err), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// GetBodyMetric обработчик для получения метрик в теле ответа.
//
//	@Summary		get body metrics
//	@Description	get body metrics
//	@Tags			metric
//	@Accept			json
//	@Produce		json
//	@Param			metric	body		http.metric	true	"metric"
//	@Success		200		{object}	http.metric
//	@Failure		400		{object}	string
//	@Failure		423		{object}	string
//	@Failure		500		{object}	string
//	@Router			/value  [post]
func (h *Handlers) GetBodyMetric(w http.ResponseWriter, r *http.Request) {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error reading body: %v", err), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	logger.Infof(r.Context(), "Get %v", r.Header)
	logger.Infof(r.Context(), "Get %v, %d", string(bodyBytes), len(bodyBytes))
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
		switch {
		case errors.Is(err, domain.ErrResourceIsLocked):
			w.WriteHeader(http.StatusLocked)
			return
		case errors.Is(err, domain.ErrNotFound):
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

// GetPing обработчик для проверки работоспособности сервиса.
//
//	@Summary		get ping
//	@Description	get ping
//	@Tags			ping
//	@Success		200	{object}	string
//	@Failure		500	{object}	string
//	@Router			/ping  [get]
func (h *Handlers) GetPing(w http.ResponseWriter, r *http.Request) {
	err := h.metricUseCases.Ping(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// UpdateMetrics обработчик для обновления метрик.
//
//	@Summary		update metrics
//	@Description	update metrics
//	@Tags			metric
//	@Accept			json
//	@Produce		plain
//	@Param			metric	body		[]http.metric	true	"metric"
//	@Success		200		{object}	string
//	@Failure		400		{object}	string
//	@Failure		423		{object}	string
//	@Failure		500		{object}	string
//	@Router			/updates [post]
func (h *Handlers) UpdateMetrics(w http.ResponseWriter, r *http.Request) {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error reading body: %v", err), http.StatusBadRequest)
		return
	}

	logger.Infof(r.Context(), "UpdateMetrics: %s", string(bodyBytes))
	var parsedMetrics []metric
	if err = json.Unmarshal(bodyBytes, &parsedMetrics); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var res []domain.MetricValue
	for _, parsedMetric := range parsedMetrics {
		v, iErr := metricToDomain(parsedMetric)
		if iErr != nil {
			http.Error(w, iErr.Error(), http.StatusBadRequest)
			return
		}

		res = append(res, v)
	}

	err = h.metricUseCases.UpdateMetrics(r.Context(), res)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrResourceIsLocked):
			w.WriteHeader(http.StatusLocked)
			return
		}

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func metricToDomain(parsedMetric metric) (domain.MetricValue, error) {
	mType, err := domain.NewMetricTypeFromString(parsedMetric.MType)
	if err != nil {
		return domain.MetricValue{}, err
	}

	var v domain.MetricValue
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
			return domain.MetricValue{}, errors.New("counter value is empty")
		}

		v = domain.MetricValue{
			Type:         mType,
			Name:         parsedMetric.ID,
			CounterValue: resValue,
		}
	}

	return v, nil
}
