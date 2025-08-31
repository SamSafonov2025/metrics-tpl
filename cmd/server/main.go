package main

import (
	"context"
	"github.com/SamSafonov2025/metrics-tpl/internal/router"
	"github.com/SamSafonov2025/metrics-tpl/internal/storage"
	"net/http"
	"os/signal"
	"syscall"

	"github.com/SamSafonov2025/metrics-tpl/internal/config"
	"github.com/SamSafonov2025/metrics-tpl/internal/logger"
	"go.uber.org/zap"
)

func main() {
	cfg := config.ParseServerFlags()

	if err := logger.Init(); err != nil {
		panic(err)
	}

	s := storage.NewStorage(cfg)
	defer storage.Close()

	r := router.New(s)

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
