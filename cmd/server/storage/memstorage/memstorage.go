package memstorage

import (
	"sync"

	"github.com/SamSafonov2025/metrics-tpl/cmd/server/metrics"
)

type MemStorage struct {
	mu       sync.RWMutex
	Counters map[string]metrics.Counter
	Gauges   map[string]metrics.Gauge

	// Раньше тут были FilePath/StoreSync/saveToFile — теперь файловой логики нет.
}

func (s *MemStorage) IncrementCounter(name string, value int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Counters[name] += metrics.Counter(value)
}

func (s *MemStorage) SetGauge(name string, value float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Gauges[name] = metrics.Gauge(value)
}

func (s *MemStorage) GetCounter(name string) (int64, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, exists := s.Counters[name]
	return int64(val), exists
}

func (s *MemStorage) GetGauge(name string) (float64, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, exists := s.Gauges[name]
	return float64(val), exists
}

func (s *MemStorage) GetAllCounters() map[string]int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[string]int64, len(s.Counters))
	for k, v := range s.Counters {
		result[k] = int64(v)
	}
	return result
}

func (s *MemStorage) GetAllGauges() map[string]float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[string]float64, len(s.Gauges))
	for k, v := range s.Gauges {
		result[k] = float64(v)
	}
	return result
}

// Доп. методы, если они вам нужны в другом коде:
func (s *MemStorage) UpdateCounter(name string, value metrics.Counter) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Counters[name] += value
	return nil
}

func (s *MemStorage) UpdateGuage(name string, value metrics.Gauge) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Gauges[name] = value
	return nil
}
