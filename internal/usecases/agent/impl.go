package agent

import (
	"context"
	"log"
	"math/rand"
	"reflect"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"

	"github.com/kdv2001/onlyMetrics/internal/domain"
)

type sendClient interface {
	SendGauge(ctx context.Context, value domain.MetricValue) error
	SendCounter(ctx context.Context, value domain.MetricValue) error
	SendMetrics(ctx context.Context, values []domain.MetricValue) error
}

type metricsClient interface {
	GetMetrics(ctx context.Context) []domain.MetricValue
}
type UseCase struct {
	sendClient    sendClient
	metricsClient metricsClient
	sendInterval  time.Duration
	workerNums    int64
}

func NewUseCase(sendClient sendClient, metricsClient metricsClient,
	sendInterval time.Duration, workerNums int64) *UseCase {
	if workerNums == 0 {
		workerNums = 1
	}

	return &UseCase{
		sendClient:    sendClient,
		metricsClient: metricsClient,
		sendInterval:  sendInterval,
		workerNums:    workerNums,
	}
}

func (u *UseCase) SendMetrics(ctx context.Context) error {
	wg := sync.WaitGroup{}
	wg.Add(1)
	jobChan := make(chan []domain.MetricValue)
	defer close(jobChan)

	// воркеры отправители
	for i := int64(0); i < u.workerNums; i++ {
		go u.sendWorker(jobChan)
	}

	// воркер сборщик
	go func() {
		defer wg.Done()
		u.sendMetrics(ctx, jobChan)
	}()

	wg.Wait()

	return nil
}

func (u *UseCase) sendMetrics(ctx context.Context, job chan<- []domain.MetricValue) {
	t := time.NewTicker(u.sendInterval)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			metrics := u.metricsClient.GetMetrics(ctx)
			part := int64(len(metrics)) / u.workerNums
			for i := int64(0); i < u.workerNums; i += part {
				bottom := i
				top := i + part
				if top > int64(len(metrics)) {
					top = int64(len(metrics))
				}

				job <- metrics[bottom:top]
			}

		}
	}
}

func (u *UseCase) sendWorker(job <-chan []domain.MetricValue) {
	for metrics := range job {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := u.sendClient.SendMetrics(ctx, metrics)
		if err != nil {
			log.Printf("error send metric: %v", err)
		}
		cancel()
	}
}

type MetricsUpdater struct {
	mu          sync.RWMutex
	stats       *runtime.MemStats
	pollCount   atomic.Int64
	randomValue Container[float64]
}

// Container ...
type Container[T comparable] struct {
	value T
	mu    sync.RWMutex
}

// GetValue возвращает значение контейнера
func (c *Container[T]) GetValue() T {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.value
}

// SetValue устанавливает новое значение контейнера
func (c *Container[T]) SetValue(n T) {
	c.mu.Lock()
	c.value = n
	c.mu.Unlock()
}

func NewMetricsUpdater(metricInterval time.Duration) *MetricsUpdater {
	m := &MetricsUpdater{
		mu:          sync.RWMutex{},
		stats:       &runtime.MemStats{},
		pollCount:   atomic.Int64{},
		randomValue: Container[float64]{},
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
		Name:       "RandomValue",
		GaugeValue: m.randomValue.GetValue(),
		Type:       domain.GaugeMetricType,
	})
	vm, _ := mem.VirtualMemory()
	metrics = append(metrics, domain.MetricValue{
		Name:       "TotalMemory",
		GaugeValue: float64(vm.Total),
		Type:       domain.GaugeMetricType,
	})
	metrics = append(metrics, domain.MetricValue{
		Name:       "FreeMemory",
		GaugeValue: float64(vm.Free),
		Type:       domain.GaugeMetricType,
	})
	cc, _ := cpu.Counts(true)
	metrics = append(metrics, domain.MetricValue{
		Name:       "CPUutilization1",
		GaugeValue: float64(cc),
		Type:       domain.GaugeMetricType,
	})
	m.mu.RUnlock()

	return metrics
}

func (m *MetricsUpdater) updateMetrics() {
	m.mu.Lock()
	runtime.ReadMemStats(m.stats)
	m.pollCount.Add(1)
	m.randomValue.SetValue(rand.Float64())
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
		case v.Type().Field(i).Type.Kind() == reflect.Uint32:
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
