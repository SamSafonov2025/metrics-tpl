package storage

import (
	"github.com/SamSafonov2025/metrics-tpl/cmd/server/metrics"
)

type MemStorage struct {
	counters map[string]metrics.Counter
	gauges   map[string]metrics.Gauge
}

func NewStorage() *MemStorage {
	return &MemStorage{
		counters: make(map[string]metrics.Counter),
		gauges:   make(map[string]metrics.Gauge),
	}
}

func (s *MemStorage) IncrementCounter(name string, value int64) {
	s.counters[name] += metrics.Counter(value)
}

func (s *MemStorage) SetGauge(name string, value float64) {
	s.gauges[name] = metrics.Gauge(value)
}

func (s *MemStorage) GetCounter(name string) (int64, bool) {
	val, exists := s.counters[name]
	return int64(val), exists
}

func (s *MemStorage) GetGauge(name string) (float64, bool) {
	val, exists := s.gauges[name]
	return float64(val), exists
}

func (s *MemStorage) GetAllCounters() map[string]int64 {
	result := make(map[string]int64)
	for k, v := range s.counters {
		result[k] = int64(v)
	}
	return result
}

func (s *MemStorage) GetAllGauges() map[string]float64 {
	result := make(map[string]float64)
	for k, v := range s.gauges {
		result[k] = float64(v)
	}
	return result
}

func (s *MemStorage) UpdateCounter(name string, value metrics.Counter) error {
	s.counters[name] += value
	return nil
}

func (s *MemStorage) UpdateGuage(name string, value metrics.Gauge) error {
	s.gauges[name] = value
	return nil
}
