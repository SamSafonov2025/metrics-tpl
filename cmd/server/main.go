package main

import (
	"context"
	"crypto/rsa"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/SamSafonov2025/metrics-tpl/internal/postgres"
	"github.com/SamSafonov2025/metrics-tpl/internal/rsacrypto"
	"github.com/SamSafonov2025/metrics-tpl/internal/service"

	"go.uber.org/zap"

	"github.com/SamSafonov2025/metrics-tpl/internal/audit"
	"github.com/SamSafonov2025/metrics-tpl/internal/config"
	"github.com/SamSafonov2025/metrics-tpl/internal/logger"
	"github.com/SamSafonov2025/metrics-tpl/internal/router"
	"github.com/SamSafonov2025/metrics-tpl/internal/storage"
)

var (
	buildVersion string = "N/A"
	buildDate    string = "N/A"
	buildCommit  string = "N/A"
)

func main() {
	// Выводим информацию о сборке
	fmt.Printf("Build version: %s\n", buildVersion)
	fmt.Printf("Build date: %s\n", buildDate)
	fmt.Printf("Build commit: %s\n", buildCommit)

	cfg := config.ParseServerFlags()

	if err := logger.Init(); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	// логируем все поля конфига
	logger.GetLogger().Info("Server config loaded !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!",
		zap.String("address", cfg.ServerAddress),
		zap.Duration("store_interval", cfg.StoreInterval),
		zap.String("file_storage_path", cfg.FileStoragePath),
		zap.Bool("restore", cfg.Restore),
		zap.String("database_dsn", cfg.Database),
		zap.String("crypto_key", cfg.CryptoKey),
		zap.String("crypto_key_path", cfg.CryptoKeyPath),
	)

	// Load RSA private key if path is provided
	var privateKey *rsa.PrivateKey
	if cfg.CryptoKeyPath != "" {
		var err error
		privateKey, err = rsacrypto.LoadPrivateKey(cfg.CryptoKeyPath)
		if err != nil {
			logger.GetLogger().Fatal("Failed to load private key",
				zap.String("path", cfg.CryptoKeyPath),
				zap.Error(err))
		}
		logger.GetLogger().Info("Loaded RSA private key", zap.String("path", cfg.CryptoKeyPath))
	}

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

	r := router.New(svc, cfg.CryptoKey, privateKey, auditPublisher)

	logger.GetLogger().Info("Server started",
		zap.String("address", cfg.ServerAddress),
		zap.Duration("store_interval", cfg.StoreInterval),
		zap.String("file_storage_path", cfg.FileStoragePath),
		zap.Bool("restore", cfg.Restore),
	)

	server := &http.Server{Addr: cfg.ServerAddress, Handler: r}

	// Graceful shutdown: перехватываем сигналы SIGINT, SIGTERM, SIGQUIT
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.GetLogger().Fatal("Server failed to start", zap.Error(err))
		}
	}()

	// Ожидаем сигнал завершения
	<-ctx.Done()
	logger.GetLogger().Info("Received shutdown signal, gracefully shutting down server...")

	// Останавливаем HTTP сервер с тайм-аутом
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.GetLogger().Error("Server shutdown error", zap.Error(err))
	} else {
		logger.GetLogger().Info("HTTP server stopped successfully")
	}

	// Сохраняем все несохранённые данные перед завершением
	logger.GetLogger().Info("Saving unsaved data...")
	storage.Close()
	logger.GetLogger().Info("Server shutdown completed successfully")
}
