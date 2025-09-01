package interfaces

import (
	"context"
	"github.com/SamSafonov2025/metrics-tpl/internal/dto"
)

type Store interface {
	StorageType() string

	SetMetrics(ctx context.Context, dto []dto.Metrics) error
	SetGauge(ctx context.Context, metricName string, value float64) error
	IncrementCounter(ctx context.Context, metricName string, value int64) error

	GetGauge(ctx context.Context, metricName string) (float64, bool)
	GetCounter(ctx context.Context, metricName string) (int64, bool)
	GetAllGauges(ctx context.Context) map[string]float64
	GetAllCounters(ctx context.Context) map[string]int64
}
