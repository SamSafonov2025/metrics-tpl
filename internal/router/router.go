package router

import (
	"github.com/SamSafonov2025/metrics-tpl/internal/service"
	"github.com/go-chi/chi/v5"

	"github.com/SamSafonov2025/metrics-tpl/cmd/server/handlers"
	"github.com/SamSafonov2025/metrics-tpl/internal/compressor"
	"github.com/SamSafonov2025/metrics-tpl/internal/logger"
)

// New строит chi.Router и регистрирует все маршруты приложения.
func New(svc service.MetricsService) *chi.Mux {
	r := chi.NewRouter()
	r.Use(compressor.GzipMiddleware)

	h := handlers.NewHandler(svc)

	r.Get("/", logger.HandlerLog(h.HomeHandler))
	r.Post("/update/{metricType}/{metricName}/{metricValue}", logger.HandlerLog(h.UpdateHandler))
	r.Get("/value/{metricType}/{metricName}", logger.HandlerLog(h.GetHandler))

	r.Get("/ping", h.Ping)

	// JSON-роуты: поддерживаем и со слэшем, и без
	r.Post("/update", logger.HandlerLog(h.UpdateHandlerJSON))
	r.Post("/update/", logger.HandlerLog(h.UpdateHandlerJSON))
	r.Post("/value", logger.HandlerLog(h.ValueHandlerJSON))
	r.Post("/value/", logger.HandlerLog(h.ValueHandlerJSON))

	// БАТЧ: тоже со слэшем и без + логгер
	r.Post("/updates", logger.HandlerLog(h.UpdateMetrics))
	r.Post("/updates/", logger.HandlerLog(h.UpdateMetrics))

	return r
}
