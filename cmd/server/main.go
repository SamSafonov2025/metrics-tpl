package main

import (
	"context"
	"net/http"
	"os/signal"
	"syscall"

	"github.com/SamSafonov2025/metrics-tpl/cmd/server/handlers"
	"github.com/SamSafonov2025/metrics-tpl/cmd/server/storage"
	"github.com/SamSafonov2025/metrics-tpl/internal/compressor"
	"github.com/SamSafonov2025/metrics-tpl/internal/config"
	"github.com/SamSafonov2025/metrics-tpl/internal/logger"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

func main() {
	cfg := config.ParseServerFlags()

	if err := logger.Init(); err != nil {
		panic(err)
	}

	storage := storage.NewStorage(cfg.FileStoragePath, cfg.StoreInterval, cfg.Restore)

	defer storage.Close()

	if cfg.StoreInterval > 0 {
		go storage.RunBackup(cfg.StoreInterval)
	}

	router := chi.NewRouter()
	router.Use(compressor.GzipMiddleware)

	h := handlers.NewHandler(storage)

	router.Get("/", logger.HandlerLog(h.HomeHandler))
	router.Post("/update/{metricType}/{metricName}/{metricValue}", logger.HandlerLog(h.UpdateHandler))
	router.Get("/value/{metricType}/{metricName}", logger.HandlerLog(h.GetHandler))
	// JSON-роуты: поддерживаем и со слэшем, и без
	router.Post("/update", logger.HandlerLog(h.UpdateHandlerJSON))
	router.Post("/update/", logger.HandlerLog(h.UpdateHandlerJSON))
	router.Post("/value", logger.HandlerLog(h.ValueHandlerJSON))
	router.Post("/value/", logger.HandlerLog(h.ValueHandlerJSON))

	logger.GetLogger().Info("Server started",
		zap.String("address", cfg.ServerAddress),
		zap.Duration("store_interval", cfg.StoreInterval),
		zap.String("file_storage_path", cfg.FileStoragePath),
		zap.Bool("restore", cfg.Restore),
	)

	server := &http.Server{Addr: cfg.ServerAddress, Handler: router}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.GetLogger().Fatal("Server failed to start",
				zap.Error(err),
			)
		}
	}()

	<-ctx.Done()

	logger.GetLogger().Info("Shutting down server...")
	if err := server.Shutdown(context.Background()); err != nil {
		logger.GetLogger().Fatal("Server forced to shutdown",
			zap.Error(err),
		)
	}
}
