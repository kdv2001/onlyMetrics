package memory

import (
	"context"
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
	s.gaugeMu.Lock()
	v := s.counter[value.Name]
	s.counter[value.Name] = v + value.CounterValue
	s.gaugeMu.Unlock()

	return nil
}
