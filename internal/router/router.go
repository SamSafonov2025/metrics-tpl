package router

import (
	"github.com/SamSafonov2025/metrics-tpl/internal/service"
	"github.com/go-chi/chi/v5"

	"github.com/SamSafonov2025/metrics-tpl/cmd/server/handlers"
	"github.com/SamSafonov2025/metrics-tpl/internal/compressor"
	"github.com/SamSafonov2025/metrics-tpl/internal/crypto"
	"github.com/SamSafonov2025/metrics-tpl/internal/logger"
)

// New строит chi.Router и регистрирует все маршруты приложения.
func New(svc service.MetricsService, key string) *chi.Mux {
	r := chi.NewRouter()
	r.Use(compressor.GzipMiddleware)

	h := handlers.NewHandler(svc)
	c := crypto.Crypto{Key: key}

	r.With(c.HashValidationMiddleware).Post("/update", logger.HandlerLog(h.UpdateHandlerJSON))
	r.With(c.HashValidationMiddleware).Post("/update/", logger.HandlerLog(h.UpdateHandlerJSON))
	r.With(c.HashValidationMiddleware).Post("/update/{metricType}/{metricName}/{metricValue}", logger.HandlerLog(h.UpdateHandler))
	r.With(c.HashValidationMiddleware).Post("/updates", logger.HandlerLog(h.UpdateMetrics))
	r.With(c.HashValidationMiddleware).Post("/updates/", logger.HandlerLog(h.UpdateMetrics))
	r.With(c.HashValidationMiddleware).Post("/value", logger.HandlerLog(h.ValueHandlerJSON))
	r.With(c.HashValidationMiddleware).Post("/value/", logger.HandlerLog(h.ValueHandlerJSON))

	r.Get("/", logger.HandlerLog(h.HomeHandler))
	r.Get("/value/{metricType}/{metricName}", logger.HandlerLog(h.GetHandler))
	r.Get("/ping", h.Ping)

	return r
}
