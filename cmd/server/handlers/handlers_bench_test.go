package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SamSafonov2025/metrics-tpl/internal/dto"
	"github.com/SamSafonov2025/metrics-tpl/internal/service"
	"github.com/SamSafonov2025/metrics-tpl/internal/storage/memstorage"
	"github.com/go-chi/chi/v5"
)

// BenchmarkHandler_UpdateHandlerJSON измеряет производительность обновления метрики через JSON
func BenchmarkHandler_UpdateHandlerJSON(b *testing.B) {
	storage := memstorage.New()
	svc := service.NewMetricsService(storage, 0, nil)
	h := NewHandler(svc, nil)

	value := 123.456
	metric := dto.Metrics{
		ID:    "test_metric",
		MType: "gauge",
		Value: &value,
	}

	body, _ := json.Marshal(metric)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		h.UpdateHandlerJSON(w, req)
	}
}

// BenchmarkHandler_ValueHandlerJSON измеряет производительность получения метрики через JSON
func BenchmarkHandler_ValueHandlerJSON(b *testing.B) {
	storage := memstorage.New()
	svc := service.NewMetricsService(storage, 0, nil)
	h := NewHandler(svc, nil)

	// Подготовка данных
	value := 123.456
	_ = storage.SetGauge(context.Background(), "test_metric", value)

	metric := dto.Metrics{
		ID:    "test_metric",
		MType: "gauge",
	}

	body, _ := json.Marshal(metric)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/value/", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		h.ValueHandlerJSON(w, req)
	}
}

// BenchmarkHandler_UpdateMetrics измеряет производительность батч-обновления
func BenchmarkHandler_UpdateMetrics(b *testing.B) {
	storage := memstorage.New()
	svc := service.NewMetricsService(storage, 0, nil)
	h := NewHandler(svc, nil)

	// Подготовка батча разного размера
	sizes := []int{10, 50, 100}

	for _, size := range sizes {
		b.Run(formatBenchName(size), func(b *testing.B) {
			metrics := make([]dto.Metrics, size)
			for i := 0; i < size; i++ {
				if i%2 == 0 {
					delta := int64(i)
					metrics[i] = dto.Metrics{
						ID:    formatMetricName("counter", i),
						MType: "counter",
						Delta: &delta,
					}
				} else {
					value := float64(i)
					metrics[i] = dto.Metrics{
						ID:    formatMetricName("gauge", i),
						MType: "gauge",
						Value: &value,
					}
				}
			}

			body, _ := json.Marshal(metrics)

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				req := httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()

				h.UpdateMetrics(w, req)
			}
		})
	}
}

// BenchmarkHandler_HomeHandler измеряет производительность отображения всех метрик
func BenchmarkHandler_HomeHandler(b *testing.B) {
	storage := memstorage.New()
	svc := service.NewMetricsService(storage, 0, nil)
	h := NewHandler(svc, nil)

	ctx := context.Background()

	// Подготовка данных
	for i := 0; i < 100; i++ {
		_ = storage.IncrementCounter(ctx, formatMetricName("counter", i), int64(i))
		_ = storage.SetGauge(ctx, formatMetricName("gauge", i), float64(i))
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		h.HomeHandler(w, req)
	}
}

// BenchmarkHandler_UpdateHandler измеряет производительность обновления через URL параметры
func BenchmarkHandler_UpdateHandler(b *testing.B) {
	storage := memstorage.New()
	svc := service.NewMetricsService(storage, 0, nil)
	h := NewHandler(svc, nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/update/gauge/test/123.456", nil)
		w := httptest.NewRecorder()

		// Эмулируем chi роутер
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("metricType", "gauge")
		rctx.URLParams.Add("metricName", "test")
		rctx.URLParams.Add("metricValue", "123.456")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		h.UpdateHandler(w, req)
	}
}

// Вспомогательные функции
func formatBenchName(size int) string {
	return "batch_size_" + string(rune(size/10+48)) + string(rune(size%10+48))
}

func formatMetricName(prefix string, i int) string {
	return prefix + "_" + string(rune(i/100+48)) + string(rune((i/10)%10+48)) + string(rune(i%10+48))
}
