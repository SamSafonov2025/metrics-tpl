package main

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/SamSafonov2025/metrics-tpl/internal/config"
	"github.com/SamSafonov2025/metrics-tpl/internal/storage"
)

// TestServerGracefulShutdown проверяет базовое поведение graceful shutdown сервера
func TestServerGracefulShutdown(t *testing.T) {
	tmpDir := t.TempDir()
	storageFile := filepath.Join(tmpDir, "metrics.json")

	cfg := &config.ServerConfig{
		ServerAddress:   "localhost:18080",
		StoreInterval:   10 * time.Second,
		FileStoragePath: storageFile,
		Restore:         false,
		Database:        "",
	}

	store := storage.NewStorage(cfg)
	defer storage.TestReset()

	// Добавляем тестовые метрики
	ctx := context.Background()
	store.SetGauge(ctx, "test_gauge", 42.5)
	store.IncrementCounter(ctx, "test_counter", 10)

	// Запускаем простой HTTP сервер
	mux := http.NewServeMux()
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	server := &http.Server{
		Addr:    cfg.ServerAddress,
		Handler: mux,
	}

	// Запуск сервера
	go func() {
		server.ListenAndServe()
	}()

	// Даём серверу время запуститься
	time.Sleep(100 * time.Millisecond)

	// Проверяем что сервер работает
	resp, err := http.Get("http://" + cfg.ServerAddress + "/ping")
	if err != nil {
		t.Fatalf("Server not responding: %v", err)
	}
	resp.Body.Close()

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		t.Errorf("Server shutdown error: %v", err)
	}

	// Сохраняем данные (как в настоящем shutdown)
	storage.Close()

	// Проверяем что файл создан
	if _, err := os.Stat(storageFile); os.IsNotExist(err) {
		t.Error("Storage file was not created during shutdown")
	} else {
		t.Log("Storage file created successfully")
	}
}

// TestServerShutdownWithRequest проверяет, что активный запрос завершается
func TestServerShutdownWithRequest(t *testing.T) {
	mux := http.NewServeMux()

	// Медленный обработчик
	mux.HandleFunc("/slow", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	server := &http.Server{
		Addr:    "localhost:18081",
		Handler: mux,
	}

	go server.ListenAndServe()
	time.Sleep(100 * time.Millisecond)

	// Запускаем медленный запрос
	requestDone := make(chan bool)
	go func() {
		resp, err := http.Get("http://localhost:18081/slow")
		if err != nil {
			requestDone <- false
			return
		}
		defer resp.Body.Close()
		requestDone <- (resp.StatusCode == http.StatusOK)
	}()

	// Даём запросу время начаться
	time.Sleep(100 * time.Millisecond)

	// Запускаем shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go func() {
		server.Shutdown(shutdownCtx)
	}()

	// Проверяем что запрос завершился успешно
	select {
	case success := <-requestDone:
		if !success {
			t.Error("Request failed during shutdown")
		} else {
			t.Log("Request completed successfully during shutdown")
		}
	case <-time.After(3 * time.Second):
		t.Error("Request didn't complete")
	}
}

// TestDataSaveOnShutdown проверяет сохранение данных
func TestDataSaveOnShutdown(t *testing.T) {
	tmpDir := t.TempDir()
	storageFile := filepath.Join(tmpDir, "save_test.json")

	cfg := &config.ServerConfig{
		ServerAddress:   "localhost:18082",
		StoreInterval:   0, // Синхронное сохранение
		FileStoragePath: storageFile,
		Restore:         false,
		Database:        "",
	}

	store := storage.NewStorage(cfg)
	defer storage.TestReset()

	ctx := context.Background()
	store.SetGauge(ctx, "cpu", 75.5)
	store.SetGauge(ctx, "memory", 80.0)

	// Симулируем shutdown
	storage.Close()

	// Проверяем файл
	if _, err := os.Stat(storageFile); os.IsNotExist(err) {
		t.Fatal("Storage file not created")
	}

	// Читаем и проверяем содержимое
	data, err := os.ReadFile(storageFile)
	if err != nil {
		t.Fatalf("Cannot read file: %v", err)
	}

	if len(data) == 0 {
		t.Error("Storage file is empty")
	} else {
		t.Logf("Storage file contains %d bytes", len(data))
	}
}
