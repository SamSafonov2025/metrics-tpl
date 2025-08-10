package main

import (
	"net/http"

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
	router := chi.NewRouter()

	h := handlers.NewHandler(storage)

	router.Get("/", logger.HandlerLog(h.HomeHandler))
	router.Post("/update/{metricType}/{metricName}/{metricValue}", logger.HandlerLog(h.UpdateHandler))
	router.Get("/value/{metricType}/{metricName}", logger.HandlerLog(h.GetHandler))

	logger.GetLogger().Info("Server started",
		zap.String("address", cfg.ServerAddress),
	)

	err := http.ListenAndServe(cfg.ServerAddress, router)
	if err != nil {
		logger.GetLogger().Fatal("Server failed to start",
			zap.Error(err),
		)
	}
}
