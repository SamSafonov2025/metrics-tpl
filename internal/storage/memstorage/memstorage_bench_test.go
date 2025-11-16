package memstorage

import (
	"context"
	"fmt"
	"testing"

	"github.com/SamSafonov2025/metrics-tpl/internal/dto"
)

// BenchmarkMemStorage_IncrementCounter измеряет производительность инкремента счетчика
func BenchmarkMemStorage_IncrementCounter(b *testing.B) {
	storage := New()
	ctx := context.Background()

	b.Run("single_thread", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = storage.IncrementCounter(ctx, "test_counter", 1)
		}
	})

	b.Run("parallel", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_ = storage.IncrementCounter(ctx, "test_counter", 1)
			}
		})
	})
}

// BenchmarkMemStorage_SetGauge измеряет производительность установки gauge
func BenchmarkMemStorage_SetGauge(b *testing.B) {
	storage := New()
	ctx := context.Background()

	b.Run("single_thread", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = storage.SetGauge(ctx, "test_gauge", 123.456)
		}
	})

	b.Run("parallel", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_ = storage.SetGauge(ctx, "test_gauge", 123.456)
			}
		})
	})
}

// BenchmarkMemStorage_GetCounter измеряет производительность чтения счетчика
func BenchmarkMemStorage_GetCounter(b *testing.B) {
	storage := New()
	ctx := context.Background()
	_ = storage.IncrementCounter(ctx, "test_counter", 100)

	b.Run("single_thread", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = storage.GetCounter(ctx, "test_counter")
		}
	})

	b.Run("parallel", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, _ = storage.GetCounter(ctx, "test_counter")
			}
		})
	})
}

// BenchmarkMemStorage_GetAllMetrics измеряет производительность получения всех метрик
func BenchmarkMemStorage_GetAllMetrics(b *testing.B) {
	storage := New()
	ctx := context.Background()

	// Подготовка данных
	for i := 0; i < 100; i++ {
		_ = storage.IncrementCounter(ctx, fmt.Sprintf("counter_%d", i), int64(i))
		_ = storage.SetGauge(ctx, fmt.Sprintf("gauge_%d", i), float64(i))
	}

	b.ResetTimer()

	b.Run("get_all_counters", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = storage.GetAllCounters(ctx)
		}
	})

	b.Run("get_all_gauges", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = storage.GetAllGauges(ctx)
		}
	})
}

// BenchmarkMemStorage_SetMetrics измеряет производительность батч-обновления
func BenchmarkMemStorage_SetMetrics(b *testing.B) {
	ctx := context.Background()

	sizes := []int{10, 50, 100, 500}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("batch_size_%d", size), func(b *testing.B) {
			storage := New()

			// Подготовка батча метрик
			metrics := make([]dto.Metrics, size)
			for i := 0; i < size; i++ {
				if i%2 == 0 {
					delta := int64(i)
					metrics[i] = dto.Metrics{
						ID:    fmt.Sprintf("counter_%d", i),
						MType: "counter",
						Delta: &delta,
					}
				} else {
					value := float64(i)
					metrics[i] = dto.Metrics{
						ID:    fmt.Sprintf("gauge_%d", i),
						MType: "gauge",
						Value: &value,
					}
				}
			}

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_ = storage.SetMetrics(ctx, metrics)
			}
		})
	}
}
