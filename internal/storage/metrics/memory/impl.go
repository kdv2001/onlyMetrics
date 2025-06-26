package memory

import (
	"context"
	"fmt"
	"sync"

	"github.com/kdv2001/onlyMetrics/internal/domain"
)

type Storage struct {
	gaugeMu sync.RWMutex
	gauge   map[string]float64

	counterMu sync.RWMutex
	counter   map[string]int64
}

func NewStorage() *Storage {
	return &Storage{
		gauge:   make(map[string]float64),
		counter: make(map[string]int64),
	}
}

func (s *Storage) UpdateGauge(_ context.Context, value domain.MetricValue) error {
	s.gaugeMu.Lock()
	s.gauge[value.Name] = value.GaugeValue
	s.gaugeMu.Unlock()

	return nil
}

func (s *Storage) UpdateCounter(_ context.Context, value domain.MetricValue) error {
	s.counterMu.Lock()
	v := s.counter[value.Name]
	s.counter[value.Name] = v + value.CounterValue
	s.counterMu.Unlock()

	return nil
}

func (s *Storage) GetGaugeValue(_ context.Context, name string) (float64, error) {
	s.gaugeMu.RLock()
	defer s.gaugeMu.RUnlock()
	val, exist := s.gauge[name]
	if !exist {
		return 0, fmt.Errorf("err get gauge: %w", domain.ErrNotFound)
	}

	return val, nil
}

func (s *Storage) GetCounterValue(_ context.Context, name string) (int64, error) {
	s.counterMu.RLock()
	defer s.counterMu.RUnlock()
	val, exist := s.counter[name]
	if !exist {
		return 0, fmt.Errorf("err get counter: %w", domain.ErrNotFound)
	}

	return val, nil
}

// GetAllValues ...
func (s *Storage) GetAllValues(_ context.Context) ([]domain.MetricValue, error) {
	values := make([]domain.MetricValue, 0, len(s.gauge)+len(s.counter))
	s.counterMu.RLock()
	defer s.counterMu.RUnlock()
	for n, v := range s.gauge {
		values = append(values, domain.MetricValue{
			Type:       domain.GaugeMetricType,
			Name:       n,
			GaugeValue: v,
		})
	}
	for n, v := range s.counter {
		values = append(values, domain.MetricValue{
			Type:         domain.CounterMetricType,
			Name:         n,
			CounterValue: v,
		})
	}

	return values, nil
}
