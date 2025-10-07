// Package memory предоставляет методы для работы с in memory хранилищем метрик.
package memory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/kdv2001/onlyMetrics/internal/domain"
	"github.com/kdv2001/onlyMetrics/pkg/logger"
)

// Storage хранилище метрик.
type Storage struct {
	gaugeMu sync.RWMutex
	gauge   map[string]float64

	counterMu sync.RWMutex
	counter   map[string]int64

	filePath string
	period   time.Duration
}

// NewStorage создает объект хранилища.
func NewStorage(ctx context.Context, filePath string,
	period time.Duration, restoreData bool) *Storage {
	s := &Storage{
		gauge:    make(map[string]float64),
		counter:  make(map[string]int64),
		filePath: filePath,
		period:   period,
	}

	s.asyncFlushData(ctx)
	// неудачная загрузка метрик не крит, чтоб не запускать приложение
	if restoreData {
		if err := s.restoreMetrics(ctx); err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				logger.Errorf(ctx, "error restore metric: %v", err)
			}
		}
	}

	return s
}

// Close закрывает хранилище с сохранением метрик.
func (s *Storage) Close(ctx context.Context) {
	err := s.flushMetrics(ctx)
	if err != nil {
		logger.Errorf(ctx, "error flush metrics")
	}
}

func (s *Storage) asyncFlushData(ctx context.Context) {
	go func() {
		// для нулевого периода запись в файл должна быть синхронной
		if s.period <= 0 {
			return
		}

		// запускаем периодический процесс для обновления метрик в файле
		ticker := time.NewTicker(s.period)
		for {
			select {
			case <-ctx.Done():
				err := s.flushMetrics(ctx)
				if err != nil {
					logger.Errorf(ctx, "error flush metric")
				}
				return
			case <-ticker.C:
				err := s.flushMetrics(ctx)
				if err != nil {
					logger.Errorf(ctx, "error flush metric: %v", err)
				}
				logger.Infof(ctx, "store metrics")
			}
		}
	}()
}

func (s *Storage) flushMetrics(ctx context.Context) error {
	if s.filePath == "" {
		return nil
	}

	values, err := s.GetAllValues(ctx)
	if err != nil {
		return err
	}

	bytes, err := json.Marshal(values)
	if err != nil {
		return err
	}

	file, err := os.OpenFile(s.filePath, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(bytes)
	if err != nil {
		return err
	}

	return nil
}

func (s *Storage) restoreMetrics(ctx context.Context) error {
	if s.filePath == "" {
		return nil
	}

	file, err := os.OpenFile(s.filePath, os.O_RDONLY, 0666)
	if err != nil {
		return err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return err
	}

	var values []domain.MetricValue
	if err = json.Unmarshal(data, &values); err != nil {
		return err
	}

	s.gaugeMu.Lock()
	defer s.gaugeMu.Unlock()
	s.counterMu.Lock()
	defer s.counterMu.Unlock()

	for _, v := range values {
		switch v.Type {
		case domain.CounterMetricType:
			s.counter[v.Name] = v.CounterValue
		case domain.GaugeMetricType:
			s.gauge[v.Name] = v.GaugeValue
		}
	}

	logger.Infof(ctx, "resotre metrics")
	return nil
}

// UpdateGauge обновить или добавить, если не существует, метрику типа "градусник".
func (s *Storage) UpdateGauge(ctx context.Context, value domain.MetricValue) error {
	s.gaugeMu.Lock()
	s.gauge[value.Name] = value.GaugeValue
	s.gaugeMu.Unlock()

	if err := s.flushMetrics(ctx); err != nil {
		return err
	}

	return nil
}

// UpdateCounter обновить или добавить, если не существует, метрику типа "счетчик".
func (s *Storage) UpdateCounter(ctx context.Context, value domain.MetricValue) error {
	s.counterMu.Lock()
	v := s.counter[value.Name]
	s.counter[value.Name] = v + value.CounterValue
	s.counterMu.Unlock()

	if err := s.flushMetrics(ctx); err != nil {
		return err
	}

	return nil
}

// GetGaugeValue получить метрику типа "градусник".
func (s *Storage) GetGaugeValue(_ context.Context, name string) (float64, error) {
	s.gaugeMu.RLock()
	defer s.gaugeMu.RUnlock()
	val, exist := s.gauge[name]
	if !exist {
		return 0, fmt.Errorf("err get gauge: %w", domain.ErrNotFound)
	}

	return val, nil
}

// GetCounterValue получить метрику типа "счетчик".
func (s *Storage) GetCounterValue(_ context.Context, name string) (int64, error) {
	s.counterMu.RLock()
	defer s.counterMu.RUnlock()
	val, exist := s.counter[name]
	if !exist {
		return 0, fmt.Errorf("err get counter: %w", domain.ErrNotFound)
	}

	return val, nil
}

// GetAllValues вернуть все значения метрик.
func (s *Storage) GetAllValues(_ context.Context) ([]domain.MetricValue, error) {
	values := make([]domain.MetricValue, 0, len(s.gauge)+len(s.counter))
	s.gaugeMu.RLock()
	defer s.gaugeMu.RUnlock()
	for n, v := range s.gauge {
		values = append(values, domain.MetricValue{
			Type:       domain.GaugeMetricType,
			Name:       n,
			GaugeValue: v,
		})
	}

	s.counterMu.RLock()
	defer s.counterMu.RUnlock()
	for n, v := range s.counter {
		values = append(values, domain.MetricValue{
			Type:         domain.CounterMetricType,
			Name:         n,
			CounterValue: v,
		})
	}

	return values, nil
}

// Ping необходим только для удовлетворения общему интерфейсу.
func (s *Storage) Ping(_ context.Context) error {
	return nil
}

// UpdateMetrics обновляет значения метрик.
func (s *Storage) UpdateMetrics(ctx context.Context, metrics []domain.MetricValue) error {
	errs := make([]error, 0)
	for _, m := range metrics {
		switch m.Type {
		case domain.GaugeMetricType:
			err := s.UpdateGauge(ctx, m)
			if err != nil {
				errs = append(errs, err)
			}
		case domain.CounterMetricType:
			err := s.UpdateCounter(ctx, m)
			if err != nil {
				errs = append(errs, err)
			}
		}
	}

	return errors.Join(errs...)
}
