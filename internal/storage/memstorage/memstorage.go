package memstorage

import (
	"context"
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

func (s *MemStorage) IncrementCounter(_ context.Context, name string, value int64) error {
	s.Counters[name] += metrics2.Counter(value)
	return nil
}

func (s *MemStorage) SetGauge(_ context.Context, name string, value float64) error {
	s.Gauges[name] = metrics2.Gauge(value)
	return nil
}

func (s *MemStorage) GetCounter(_ context.Context, name string) (int64, bool) {
	val, exists := s.Counters[name]
	return int64(val), exists
}

func (s *MemStorage) GetGauge(_ context.Context, name string) (float64, bool) {
	val, exists := s.Gauges[name]
	return float64(val), exists
}

func (s *MemStorage) GetAllCounters(_ context.Context) map[string]int64 {
	result := make(map[string]int64, len(s.Counters))
	for k, v := range s.Counters {
		result[k] = int64(v)
	}
	return result
}

func (s *MemStorage) GetAllGauges(_ context.Context) map[string]float64 {
	result := make(map[string]float64, len(s.Gauges))
	for k, v := range s.Gauges {
		result[k] = float64(v)
	}
	return result
}

func (s *MemStorage) UpdateCounter(_ context.Context, name string, value metrics2.Counter) error {
	s.Counters[name] += value
	return nil
}

// Правильное имя
func (s *MemStorage) UpdateGauge(_ context.Context, name string, value metrics2.Gauge) error {
	s.Gauges[name] = value
	return nil
}

func (s *MemStorage) SetMetrics(ctx context.Context, metrics []dto.Metrics) error {
	for _, metric := range metrics {
		switch metric.MType {
		case dto.MetricTypeCounter:
			if metric.Delta == nil {
				log.Printf("counter %q has nil delta — skipped", metric.ID)
				continue
			}
			s.IncrementCounter(ctx, metric.ID, *metric.Delta)

		case dto.MetricTypeGauge:
			if metric.Value == nil {
				log.Printf("gauge %q has nil value — skipped", metric.ID)
				continue
			}
			s.SetGauge(ctx, metric.ID, *metric.Value)

		default:
			log.Printf("Unknown metric type: %s (id=%s)", metric.MType, metric.ID)
		}
	}
	return nil
}

func (s *MemStorage) StorageType() string {
	return "ms"
}
