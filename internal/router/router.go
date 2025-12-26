package router

import (
	"crypto/rsa"

	"github.com/go-chi/chi/v5"

	"github.com/SamSafonov2025/metrics-tpl/internal/service"

	"github.com/SamSafonov2025/metrics-tpl/cmd/server/handlers"
	"github.com/SamSafonov2025/metrics-tpl/internal/audit"
	"github.com/SamSafonov2025/metrics-tpl/internal/compressor"
	"github.com/SamSafonov2025/metrics-tpl/internal/crypto"
	"github.com/SamSafonov2025/metrics-tpl/internal/logger"
	"github.com/SamSafonov2025/metrics-tpl/internal/rsadecryptor"
)

// New строит chi.Router и регистрирует все маршруты приложения.
func New(svc service.MetricsService, key string, privateKey *rsa.PrivateKey, auditPublisher *audit.AuditPublisher) *chi.Mux {
	r := chi.NewRouter()

	// порядок важен:
	// 0) расшифровка RSA (если есть)
	if privateKey != nil {
		r.Use(rsadecryptor.RSADecryptMiddleware(privateKey))
	}
	// 1) распаковка gzip (если есть)
	gzipMW := compressor.NewGzipMiddleware()
	r.Use(gzipMW.Handler)
	// 2) Глобальный логгер — увидит и 400 от HashValidationMiddleware
	r.Use(logger.Middleware)

	h := handlers.NewHandler(svc, auditPublisher)
	c := crypto.Crypto{Key: key}

	// Можно убрать HandlerLog(...) здесь, чтобы не было дублей.
	// Я оставлю чистые хендлеры; если хотите оставить старые — просто верните logger.HandlerLog(...)
	r.With(c.HashValidationMiddleware).Post("/update", h.UpdateHandlerJSON)
	r.With(c.HashValidationMiddleware).Post("/update/", h.UpdateHandlerJSON)
	r.With(c.HashValidationMiddleware).Post("/update/{metricType}/{metricName}/{metricValue}", h.UpdateHandler)
	r.With(c.HashValidationMiddleware).Post("/updates", h.UpdateMetrics)
	r.With(c.HashValidationMiddleware).Post("/updates/", h.UpdateMetrics)
	r.With(c.HashValidationMiddleware).Post("/value", h.ValueHandlerJSON)
	r.With(c.HashValidationMiddleware).Post("/value/", h.ValueHandlerJSON)

	r.Get("/", h.HomeHandler)
	r.Get("/value/{metricType}/{metricName}", h.GetHandler)
	r.Get("/ping", h.Ping)

	return r
}
