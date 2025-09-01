package router

import (
	"github.com/go-chi/chi/v5"

	"github.com/SamSafonov2025/metrics-tpl/cmd/server/handlers"
	"github.com/SamSafonov2025/metrics-tpl/internal/compressor"
	"github.com/SamSafonov2025/metrics-tpl/internal/interfaces"
	"github.com/SamSafonov2025/metrics-tpl/internal/logger"
)

// New строит chi.Router и регистрирует все маршруты приложения.
func New(s interfaces.Store) *chi.Mux {
	r := chi.NewRouter()
	r.Use(compressor.GzipMiddleware)

	h := handlers.NewHandler(s)

	r.Get("/", logger.HandlerLog(h.HomeHandler))
	r.Post("/update/{metricType}/{metricName}/{metricValue}", logger.HandlerLog(h.UpdateHandler))
	r.Get("/value/{metricType}/{metricName}", logger.HandlerLog(h.GetHandler))

	r.Get("/ping", h.Ping)

	// JSON-роуты: поддерживаем и со слэшем, и без
	r.Post("/update", logger.HandlerLog(h.UpdateHandlerJSON))
	r.Post("/update/", logger.HandlerLog(h.UpdateHandlerJSON))
	r.Post("/value", logger.HandlerLog(h.ValueHandlerJSON))
	r.Post("/value/", logger.HandlerLog(h.ValueHandlerJSON))

	r.Post("/updates/", h.UpdateMetrics)

	return r
}
