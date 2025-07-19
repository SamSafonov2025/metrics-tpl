package storage

import (
	"github.com/SamSafonov2025/metrics-tpl.git/cmd/server/metrics"
)

type MemStorage struct {
	counters map[string][]metrics.Counter
	gauges   map[string]metrics.Gauge
}

func NewStorage() *MemStorage {
	return &MemStorage{
		counters: make(map[string][]metrics.Counter),
		gauges:   make(map[string]metrics.Gauge),
	}
}

func (s *MemStorage) UpdateCounter(name string, value metrics.Counter) error {
	s.counters[name] = append(s.counters[name], value)
	return nil
}

func (s *MemStorage) UpdateGuage(name string, value metrics.Gauge) error {
	s.gauges[name] = value
	return nil
}
