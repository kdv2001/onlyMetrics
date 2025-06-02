package agent

import (
	"context"
	"errors"
	"log"
	"math/rand"
	"reflect"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kdv2001/onlyMetrics/internal/domain"
)

type sendClient interface {
	SendGauge(ctx context.Context, value domain.MetricValue) error
	SendCounter(ctx context.Context, value domain.MetricValue) error
}

type metricsClient interface {
	GetMetrics(ctx context.Context) []domain.MetricValue
}
type UseCase struct {
	sendClient    sendClient
	metricsClient metricsClient
	sendInterval  time.Duration
}

func NewUseCase(sendClient sendClient, metricsClient metricsClient, sendInterval time.Duration) *UseCase {
	return &UseCase{
		sendClient:    sendClient,
		metricsClient: metricsClient,
		sendInterval:  sendInterval,
	}
}

func (u *UseCase) SendMetrics(ctx context.Context) error {
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		u.sendMetrics(ctx)
	}()

	wg.Wait()

	return nil
}

func (u *UseCase) sendMetrics(ctx context.Context) {
	t := time.NewTicker(u.sendInterval)
	defer t.Stop()

	for range t.C {
		metrics := u.metricsClient.GetMetrics(ctx)
		errs := make([]error, 0, len(metrics))
		for _, metric := range metrics {
			switch metric.Type {
			case domain.CounterMetricType:
				err := u.sendClient.SendCounter(ctx, metric)
				if err != nil {
					errs = append(errs, err)
				}
			case domain.GaugeMetricType:
				err := u.sendClient.SendGauge(ctx, metric)
				if err != nil {
					errs = append(errs, err)
				}
			}
		}

		if len(errs) > 0 {
			log.Printf("error send metric: %v", errors.Join(errs...))
		}
	}
}

type MetricsUpdater struct {
	mu          sync.RWMutex
	stats       *runtime.MemStats
	pollCount   atomic.Int64
	randomValue atomic.Int64
}

func NewMetricsUpdater(metricInterval time.Duration) *MetricsUpdater {
	m := &MetricsUpdater{
		mu:          sync.RWMutex{},
		stats:       &runtime.MemStats{},
		pollCount:   atomic.Int64{},
		randomValue: atomic.Int64{},
	}

	go func() {
		t := time.NewTicker(metricInterval)
		defer t.Stop()

		for range t.C {
			m.updateMetrics()
		}
	}()

	return m
}

func (m *MetricsUpdater) GetMetrics(_ context.Context) []domain.MetricValue {
	m.mu.RLock()
	v := reflect.ValueOf(m.stats).Elem()
	metrics := recursiveGetMetrics(v)
	metrics = append(metrics, domain.MetricValue{
		Name:         "PollCount",
		CounterValue: m.pollCount.Load(),
		Type:         domain.CounterMetricType,
	})
	metrics = append(metrics, domain.MetricValue{
		Name:         "RandomValue",
		CounterValue: m.randomValue.Load(),
		Type:         domain.CounterMetricType,
	})
	m.mu.RUnlock()

	return metrics
}

func (m *MetricsUpdater) updateMetrics() {
	m.mu.Lock()
	runtime.ReadMemStats(m.stats)
	m.pollCount.Add(1)
	m.randomValue.Swap(rand.Int63())
	m.mu.Unlock()
}

func recursiveGetMetrics(v reflect.Value) []domain.MetricValue {
	res := make([]domain.MetricValue, 0, 1)
	for i := 0; i < v.NumField(); i++ {
		switch {
		case v.Type().Field(i).Type.Kind() == reflect.Uint64:
			res = append(res, domain.MetricValue{
				Name:       v.Type().Field(i).Name,
				GaugeValue: float64(v.Field(i).Uint()),
				Type:       domain.GaugeMetricType,
			})
		case v.Type().Field(i).Type.Kind() == reflect.Float64:
			res = append(res, domain.MetricValue{
				Name:       v.Type().Field(i).Name,
				GaugeValue: v.Field(i).Float(),
				Type:       domain.GaugeMetricType,
			})
		case v.Type().Field(i).Type.Kind() == reflect.Struct:
			res = append(res, recursiveGetMetrics(v.Field(i))...)
		}
	}
	return res
}
