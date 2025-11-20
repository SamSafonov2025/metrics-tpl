package main

import (
	"context"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/SamSafonov2025/metrics-tpl/internal/postgres"
	"github.com/SamSafonov2025/metrics-tpl/internal/service"

	"go.uber.org/zap"

	"github.com/SamSafonov2025/metrics-tpl/internal/audit"
	"github.com/SamSafonov2025/metrics-tpl/internal/config"
	"github.com/SamSafonov2025/metrics-tpl/internal/logger"
	"github.com/SamSafonov2025/metrics-tpl/internal/router"
	"github.com/SamSafonov2025/metrics-tpl/internal/storage"
)

func main() {
	cfg := config.ParseServerFlags()

	if err := logger.Init(); err != nil {
		panic(err)
	}

	// логируем все поля конфига
	logger.GetLogger().Info("Server config loaded !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!",
		zap.String("address", cfg.ServerAddress),
		zap.Duration("store_interval", cfg.StoreInterval),
		zap.String("file_storage_path", cfg.FileStoragePath),
		zap.Bool("restore", cfg.Restore),
		zap.String("database_dsn", cfg.Database),
		zap.String("crypto_key", cfg.CryptoKey),
	)

	s := storage.NewStorage(cfg) // репозиторий (interfaces.Store)
	svc := service.NewMetricsService(s, cfg.StoreInterval,
		func(ctx context.Context) error { return postgres.Pool.Ping(ctx) })

	// Инициализируем систему аудита
	auditPublisher := audit.NewAuditPublisher()
	defer auditPublisher.Close()

	// Регистрируем наблюдателей на основе конфигурации
	if cfg.AuditFile != "" {
		fileObserver, err := audit.NewFileAuditObserver(cfg.AuditFile)
		if err != nil {
			logger.GetLogger().Fatal("Failed to create file audit observer", zap.Error(err))
		}
		auditPublisher.Register(fileObserver)
		logger.GetLogger().Info("File audit observer registered", zap.String("file", cfg.AuditFile))
	}

	if cfg.AuditURL != "" {
		urlObserver := audit.NewURLAuditObserver(cfg.AuditURL)
		auditPublisher.Register(urlObserver)
		logger.GetLogger().Info("URL audit observer registered", zap.String("url", cfg.AuditURL))
	}

	r := router.New(svc, cfg.CryptoKey, auditPublisher)

	logger.GetLogger().Info("Server started",
		zap.String("address", cfg.ServerAddress),
		zap.Duration("store_interval", cfg.StoreInterval),
		zap.String("file_storage_path", cfg.FileStoragePath),
		zap.Bool("restore", cfg.Restore),
	)

	server := &http.Server{Addr: cfg.ServerAddress, Handler: r}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.GetLogger().Fatal("Server failed to start", zap.Error(err))
		}
	}()

	<-ctx.Done()

	logger.GetLogger().Info("Shutting down server...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.GetLogger().Fatal("Server forced to shutdown", zap.Error(err))
	}
}
