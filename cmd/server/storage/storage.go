package storage

import (
	"encoding/json"
	"os"
	"sync"
	"time"

	"github.com/SamSafonov2025/metrics-tpl/cmd/server/metrics"
)

type MemStorage struct {
	mu        sync.RWMutex
	counters  map[string]metrics.Counter
	gauges    map[string]metrics.Gauge
	filePath  string
	storeSync bool
	done      chan struct{}
}

func NewStorage(filePath string, storeInterval time.Duration, restore bool) *MemStorage {
	s := &MemStorage{
		counters:  make(map[string]metrics.Counter),
		gauges:    make(map[string]metrics.Gauge),
		filePath:  filePath,
		storeSync: storeInterval == 0,
		done:      make(chan struct{}),
	}

	if restore {
		s.loadFromFile()
	}

	return s
}

func (s *MemStorage) RunBackup(interval time.Duration) {
	if interval <= 0 {
		return
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.SaveToFile()
		case <-s.done:
			return
		}
	}
}

func (s *MemStorage) Close() {
	close(s.done)
	s.SaveToFile()
}

func (s *MemStorage) loadFromFile() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	file, err := os.Open(s.filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	var metricsList []Metrics
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&metricsList); err != nil {
		return err
	}

	newCounters := make(map[string]metrics.Counter)
	newGauges := make(map[string]metrics.Gauge)

	for _, m := range metricsList {
		switch m.MType {
		case "counter":
			if m.Delta == nil {
				continue
			}
			newCounters[m.ID] = metrics.Counter(*m.Delta)
		case "gauge":
			if m.Value == nil {
				continue
			}
			newGauges[m.ID] = metrics.Gauge(*m.Value)
		}
	}

	s.counters = newCounters
	s.gauges = newGauges

	return nil
}

func (s *MemStorage) SaveToFile() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	file, err := os.Create(s.filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	var metricsList []Metrics

	for k, v := range s.counters {
		delta := int64(v)
		metricsList = append(metricsList, Metrics{ID: k, MType: "counter", Delta: &delta})
	}

	for k, v := range s.gauges {
		value := float64(v)
		metricsList = append(metricsList, Metrics{ID: k, MType: "gauge", Value: &value})
	}

	encoder := json.NewEncoder(file)
	return encoder.Encode(metricsList)
}

func (s *MemStorage) IncrementCounter(name string, value int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counters[name] += metrics.Counter(value)
	if s.storeSync {
		s.saveToFile()
	}
}

func (s *MemStorage) SetGauge(name string, value float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gauges[name] = metrics.Gauge(value)
	if s.storeSync {
		s.saveToFile()
	}
}

func (s *MemStorage) saveToFile() {
	file, err := os.Create(s.filePath)
	if err != nil {
		return
	}
	defer file.Close()

	var metricsList []Metrics

	for k, v := range s.counters {
		delta := int64(v)
		metricsList = append(metricsList, Metrics{ID: k, MType: "counter", Delta: &delta})
	}

	for k, v := range s.gauges {
		value := float64(v)
		metricsList = append(metricsList, Metrics{ID: k, MType: "gauge", Value: &value})
	}

	encoder := json.NewEncoder(file)
	_ = encoder.Encode(metricsList)
}

func (s *MemStorage) GetCounter(name string) (int64, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, exists := s.counters[name]
	return int64(val), exists
}

func (s *MemStorage) GetGauge(name string) (float64, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, exists := s.gauges[name]
	return float64(val), exists
}

func (s *MemStorage) GetAllCounters() map[string]int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[string]int64)
	for k, v := range s.counters {
		result[k] = int64(v)
	}
	return result
}

func (s *MemStorage) GetAllGauges() map[string]float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[string]float64)
	for k, v := range s.gauges {
		result[k] = float64(v)
	}
	return result
}

func (s *MemStorage) UpdateCounter(name string, value metrics.Counter) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counters[name] += value
	if s.storeSync {
		s.saveToFile()
	}
	return nil
}

func (s *MemStorage) UpdateGuage(name string, value metrics.Gauge) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gauges[name] = value
	if s.storeSync {
		s.saveToFile()
	}
	return nil
}

type Metrics struct {
	ID    string   `json:"id"`
	MType string   `json:"type"`
	Delta *int64   `json:"delta,omitempty"`
	Value *float64 `json:"value,omitempty"`
}
