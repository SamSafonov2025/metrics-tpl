package interfaces

import "github.com/SamSafonov2025/metrics-tpl/internal/dto"

type Store interface {
	StorageType() string
	SetMetrics(dto []dto.Metrics)
	SetGauge(metricName string, value float64)
	IncrementCounter(metricName string, value int64)
	GetGauge(metricName string) (float64, bool)
	GetCounter(metricName string) (int64, bool)
	GetAllGauges() map[string]float64
	GetAllCounters() map[string]int64
}
