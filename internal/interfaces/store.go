// Package interfaces содержит интерфейсы для абстракции зависимостей.
package interfaces

import (
	"context"

	"github.com/SamSafonov2025/metrics-tpl/internal/dto"
)

// Store определяет интерфейс хранилища метрик.
// Реализации могут использовать различные backend'ы: память, файловую систему, базу данных.
//
// Интерфейс поддерживает два типа метрик:
//   - Gauge: вещественные значения, которые могут увеличиваться и уменьшаться
//   - Counter: целочисленные счетчики, которые только увеличиваются
type Store interface {
	// StorageType возвращает строковое описание типа хранилища
	// (например, "memory", "file", "postgres")
	StorageType() string

	// SetMetrics атомарно сохраняет множество метрик.
	// Используется для массовых операций и должен обеспечивать транзакционность.
	SetMetrics(ctx context.Context, dto []dto.Metrics) error

	// SetGauge устанавливает значение gauge метрики.
	// Перезаписывает предыдущее значение, если метрика существует.
	SetGauge(ctx context.Context, metricName string, value float64) error

	// IncrementCounter увеличивает значение counter метрики на заданную величину.
	// Если метрика не существует, создает её с начальным значением равным value.
	IncrementCounter(ctx context.Context, metricName string, value int64) error

	// GetGauge возвращает значение gauge метрики.
	// Второй параметр (bool) указывает, существует ли метрика.
	GetGauge(ctx context.Context, metricName string) (float64, bool)

	// GetCounter возвращает значение counter метрики.
	// Второй параметр (bool) указывает, существует ли метрика.
	GetCounter(ctx context.Context, metricName string) (int64, bool)

	// GetAllGauges возвращает все gauge метрики в виде map[имя]значение.
	GetAllGauges(ctx context.Context) map[string]float64

	// GetAllCounters возвращает все counter метрики в виде map[имя]значение.
	GetAllCounters(ctx context.Context) map[string]int64
}
