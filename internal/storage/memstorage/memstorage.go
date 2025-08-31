package memstorage

import (
	metrics2 "github.com/SamSafonov2025/metrics-tpl/internal/metrics"
	"sync"
)

type MemStorage struct {
	mu       sync.RWMutex
	Counters map[string]metrics2.Counter
	Gauges   map[string]metrics2.Gauge

	// Раньше тут были FilePath/StoreSync/saveToFile — теперь файловой логики нет.
}

func (s *MemStorage) IncrementCounter(name string, value int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Counters[name] += metrics2.Counter(value)
}

func (s *MemStorage) SetGauge(name string, value float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Gauges[name] = metrics2.Gauge(value)
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
func (s *MemStorage) UpdateCounter(name string, value metrics2.Counter) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Counters[name] += value
	return nil
}

func (s *MemStorage) UpdateGuage(name string, value metrics2.Gauge) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Gauges[name] = value
	return nil
}
