package memstorage

import (
	"context"
	"github.com/SamSafonov2025/metrics-tpl/internal/consts"
	"log"
	"sync"

	"github.com/SamSafonov2025/metrics-tpl/internal/dto"
	metrics2 "github.com/SamSafonov2025/metrics-tpl/internal/metrics"
)

type MemStorage struct {
	mu       sync.RWMutex
	Counters map[string]metrics2.Counter
	Gauges   map[string]metrics2.Gauge
}

func New() *MemStorage {
	return &MemStorage{
		Counters: make(map[string]metrics2.Counter),
		Gauges:   make(map[string]metrics2.Gauge),
	}
}

func (s *MemStorage) IncrementCounter(_ context.Context, name string, value int64) error {
	s.mu.Lock()
	s.Counters[name] += metrics2.Counter(value)
	s.mu.Unlock()
	return nil
}

func (s *MemStorage) SetGauge(_ context.Context, name string, value float64) error {
	s.mu.Lock()
	s.Gauges[name] = metrics2.Gauge(value)
	s.mu.Unlock()
	return nil
}

func (s *MemStorage) GetCounter(_ context.Context, name string) (int64, bool) {
	s.mu.RLock()
	val, exists := s.Counters[name]
	s.mu.RUnlock()
	return int64(val), exists
}

func (s *MemStorage) GetGauge(_ context.Context, name string) (float64, bool) {
	s.mu.RLock()
	val, exists := s.Gauges[name]
	s.mu.RUnlock()
	return float64(val), exists
}

func (s *MemStorage) GetAllCounters(_ context.Context) map[string]int64 {
	s.mu.RLock()
	result := make(map[string]int64, len(s.Counters))
	for k, v := range s.Counters {
		result[k] = int64(v)
	}
	s.mu.RUnlock()
	return result
}

func (s *MemStorage) GetAllGauges(_ context.Context) map[string]float64 {
	s.mu.RLock()
	result := make(map[string]float64, len(s.Gauges))
	for k, v := range s.Gauges {
		result[k] = float64(v)
	}
	s.mu.RUnlock()
	return result
}

// Если эти методы нужны интерфейсом — оставляем и делаем потокобезопасными
func (s *MemStorage) UpdateCounter(_ context.Context, name string, value metrics2.Counter) error {
	s.mu.Lock()
	s.Counters[name] += value
	s.mu.Unlock()
	return nil
}

func (s *MemStorage) UpdateGauge(_ context.Context, name string, value metrics2.Gauge) error {
	s.mu.Lock()
	s.Gauges[name] = value
	s.mu.Unlock()
	return nil
}

// Батч-обновление: держим lock на время всего прохода (атоминее и быстрее)
func (s *MemStorage) SetMetrics(_ context.Context, metrics []dto.Metrics) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, metric := range metrics {
		switch metric.MType {
		case consts.MetricTypeCounter:
			if metric.Delta == nil {
				log.Printf("counter %q has nil delta — skipped", metric.ID)
				continue
			}
			s.Counters[metric.ID] += metrics2.Counter(*metric.Delta)

		case consts.MetricTypeGauge:
			if metric.Value == nil {
				log.Printf("gauge %q has nil value — skipped", metric.ID)
				continue
			}
			s.Gauges[metric.ID] = metrics2.Gauge(*metric.Value)

		default:
			log.Printf("Unknown metric type: %s (id=%s)", metric.MType, metric.ID)
		}
	}
	return nil
}

func (s *MemStorage) StorageType() string {
	return "ms"
}
