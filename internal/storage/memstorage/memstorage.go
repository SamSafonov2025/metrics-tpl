package memstorage

import (
	"log"

	"github.com/SamSafonov2025/metrics-tpl/internal/dto"
	metrics2 "github.com/SamSafonov2025/metrics-tpl/internal/metrics"
)

type MemStorage struct {
	Counters map[string]metrics2.Counter
	Gauges   map[string]metrics2.Gauge
}

// Конструктор на всякий случай (если ещё не было)
func New() *MemStorage {
	return &MemStorage{
		Counters: make(map[string]metrics2.Counter),
		Gauges:   make(map[string]metrics2.Gauge),
	}
}

func (s *MemStorage) IncrementCounter(name string, value int64) {
	s.Counters[name] += metrics2.Counter(value)
}

func (s *MemStorage) SetGauge(name string, value float64) {
	s.Gauges[name] = metrics2.Gauge(value)
}

func (s *MemStorage) GetCounter(name string) (int64, bool) {
	val, exists := s.Counters[name]
	return int64(val), exists
}

func (s *MemStorage) GetGauge(name string) (float64, bool) {
	val, exists := s.Gauges[name]
	return float64(val), exists
}

func (s *MemStorage) GetAllCounters() map[string]int64 {
	result := make(map[string]int64, len(s.Counters))
	for k, v := range s.Counters {
		result[k] = int64(v)
	}
	return result
}

func (s *MemStorage) GetAllGauges() map[string]float64 {
	result := make(map[string]float64, len(s.Gauges))
	for k, v := range s.Gauges {
		result[k] = float64(v)
	}
	return result
}

func (s *MemStorage) UpdateCounter(name string, value metrics2.Counter) error {
	s.Counters[name] += value
	return nil
}

// Правильное имя
func (s *MemStorage) UpdateGauge(name string, value metrics2.Gauge) error {
	s.Gauges[name] = value
	return nil
}

// Совместимость со старым кодом (опечатка). Можно удалить позже.
// Deprecated: use UpdateGauge.
func (s *MemStorage) UpdateGuage(name string, value metrics2.Gauge) error {
	return s.UpdateGauge(name, value)
}

func (s *MemStorage) SetMetrics(metrics []dto.Metrics) {
	for _, metric := range metrics {
		switch metric.MType {
		case dto.MetricTypeCounter:
			if metric.Delta == nil {
				log.Printf("counter %q has nil delta — skipped", metric.ID)
				continue
			}
			s.IncrementCounter(metric.ID, *metric.Delta)

		case dto.MetricTypeGauge:
			if metric.Value == nil {
				log.Printf("gauge %q has nil value — skipped", metric.ID)
				continue
			}
			s.SetGauge(metric.ID, *metric.Value)

		default:
			log.Printf("Unknown metric type: %s (id=%s)", metric.MType, metric.ID)
		}
	}
}

func (s *MemStorage) StorageType() string {
	return "ms"
}
