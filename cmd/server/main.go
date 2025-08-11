package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/SamSafonov2025/metrics-tpl/cmd/server/handlers"
	"github.com/SamSafonov2025/metrics-tpl/cmd/server/storage"
	"github.com/SamSafonov2025/metrics-tpl/internal/config"
	"github.com/SamSafonov2025/metrics-tpl/internal/logger"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

func main() {
	cfg := config.ParseFlags()

	if err := logger.Init(); err != nil {
		panic(err)
	}

	storage := storage.NewStorage()

	if cfg.Restore {
		if err := storage.LoadFromFile(cfg.FileStoragePath); err != nil {
			logger.GetLogger().Warn("Failed to restore metrics from file",
				zap.String("path", cfg.FileStoragePath),
				zap.Error(err),
			)
		} else {
			logger.GetLogger().Info("Metrics restored from file",
				zap.String("path", cfg.FileStoragePath),
			)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if cfg.StoreInterval > 0 {
		go func() {
			ticker := time.NewTicker(cfg.StoreInterval)
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
					if err := storage.SaveToFile(cfg.FileStoragePath); err != nil {
						logger.GetLogger().Error("Failed to save metrics", zap.Error(err))
					} else {
						logger.GetLogger().Info("Metrics saved to file",
							zap.String("path", cfg.FileStoragePath),
						)
					}
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-shutdown
		logger.GetLogger().Info("Shutting down server...")
		if err := storage.SaveToFile(cfg.FileStoragePath); err != nil {
			logger.GetLogger().Error("Failed to save metrics on shutdown", zap.Error(err))
		} else {
			logger.GetLogger().Info("Metrics saved on shutdown",
				zap.String("path", cfg.FileStoragePath),
			)
		}
		cancel()
	}()

	router := chi.NewRouter()
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

	server := &http.Server{
		Addr:    cfg.ServerAddress,
		Handler: router,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.GetLogger().Fatal("Server failed to start", zap.Error(err))
		}
	}()

	<-ctx.Done()
	if err := server.Shutdown(context.Background()); err != nil {
		logger.GetLogger().Error("Server shutdown error", zap.Error(err))
	}
}
