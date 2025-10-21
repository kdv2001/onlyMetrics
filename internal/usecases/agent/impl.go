// Package agent предоставляет методы бизнес-логики для сбора метрик и последующей их обработки.
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
	"github.com/kdv2001/onlyMetrics/pkg/logger"
)

type sendClient interface {
	SendGauge(ctx context.Context, value domain.MetricValue) error
	SendCounter(ctx context.Context, value domain.MetricValue) error
	SendMetrics(ctx context.Context, values []domain.MetricValue) error
}

type metricsClient interface {
	GetMetrics(ctx context.Context) ([]domain.MetricValue, error)
}

// UseCase объект, содержащий бизнес-логику обработки метрик.
type UseCase struct {
	sendClient    sendClient
	metricsClient metricsClient
	sendInterval  time.Duration
	workerNums    int64
}

// NewUseCase создает объект бизнес логики.
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

// SendMetrics отправляет метрики потребителю.
func (u *UseCase) SendMetrics(ctx context.Context) error {
	wg := sync.WaitGroup{}
	jobChan := make(chan []domain.MetricValue, u.workerNums)
	// воркеры отправители
	for i := int64(0); i < u.workerNums; i++ {
		wg.Add(1)
		go u.sendWorker(jobChan, &wg)
	}

	wg.Add(1)
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
			close(job)
			return
		case <-t.C:
			metrics, err := u.metricsClient.GetMetrics(ctx)
			if err != nil {
				log.Printf("error GetMetrics: %v", err)
				continue
			}
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

func (u *UseCase) sendWorker(job <-chan []domain.MetricValue, wg *sync.WaitGroup) {
	defer wg.Done()
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
	pollCount   atomic.Int64
	randomValue Container[float64]
	stats       []domain.MetricValue
	mu          sync.RWMutex
}

// Container объект для обеспечения безопасного доступ к данным.
type Container[T comparable] struct {
	value T
	mu    sync.RWMutex
}

// GetValue возвращает значение контейнера.
func (c *Container[T]) GetValue() T {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.value
}

// SetValue устанавливает новое значение контейнера.
func (c *Container[T]) SetValue(n T) {
	c.mu.Lock()
	c.value = n
	c.mu.Unlock()
}

// NewMetricsUpdater создает объект автоматического сбора и обновления метрик.
func NewMetricsUpdater(ctx context.Context, metricInterval time.Duration) *MetricsUpdater {
	m := &MetricsUpdater{
		mu:          sync.RWMutex{},
		stats:       nil,
		pollCount:   atomic.Int64{},
		randomValue: Container[float64]{},
	}

	err := m.updateMetrics(ctx)
	if err != nil {
		logger.Errorf(ctx, "error updating metrics: %v", err)
	}

	go func() {
		t := time.NewTicker(metricInterval)
		defer t.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				err := m.updateMetrics(ctx)
				if err != nil {
					logger.Errorf(ctx, "error updating metrics: %v", err)
				}
			}
		}
	}()

	return m
}

// GetMetrics возвращает собранные значения метрик.
func (m *MetricsUpdater) GetMetrics(_ context.Context) ([]domain.MetricValue, error) {
	m.mu.RLock()
	metrics := m.stats
	m.mu.RUnlock()

	return metrics, nil
}

// updateMetrics обновляет значения метрик.
func (m *MetricsUpdater) updateMetrics(_ context.Context) error {
	vm, err := mem.VirtualMemory()
	if err != nil {
		return err
	}

	stats := new(runtime.MemStats)
	runtime.ReadMemStats(stats)
	v := reflect.ValueOf(stats).Elem()
	metrics := recursiveGetMetrics(v)

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

	pollCountNew := m.pollCount.Add(1)
	metrics = append(metrics, domain.MetricValue{
		Name:         "PollCount",
		CounterValue: pollCountNew,
		Type:         domain.CounterMetricType,
	})

	randomValueNew := rand.Float64()
	m.randomValue.SetValue(randomValueNew)
	metrics = append(metrics, domain.MetricValue{
		Name:       "RandomValue",
		GaugeValue: randomValueNew,
		Type:       domain.GaugeMetricType,
	})

	cc, _ := cpu.Counts(true)
	metrics = append(metrics, domain.MetricValue{
		Name:       "CPUutilization1",
		GaugeValue: float64(cc),
		Type:       domain.GaugeMetricType,
	})

	m.mu.Lock()
	m.stats = metrics
	m.mu.Unlock()

	return nil
}

// recursiveGetMetrics рекурсивно проходит по каждому полю структуры.
func recursiveGetMetrics(v reflect.Value) []domain.MetricValue {
	res := make([]domain.MetricValue, 0, v.NumField())
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
