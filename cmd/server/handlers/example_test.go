package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/SamSafonov2025/metrics-tpl/cmd/server/handlers"
	"github.com/SamSafonov2025/metrics-tpl/internal/config"
	"github.com/SamSafonov2025/metrics-tpl/internal/dto"
	"github.com/SamSafonov2025/metrics-tpl/internal/service"
	"github.com/SamSafonov2025/metrics-tpl/internal/storage"
)

// Example демонстрирует базовую работу с метриками через HTTP API.
// Показывает обновление gauge и counter метрик, а также получение их значений.
func Example() {
	// Сбрасываем синглтон стораджа для чистого примера
	storage.TestReset()
	defer storage.Close()

	// Создаем хранилище и сервис
	cfg := &config.ServerConfig{
		StoreInterval:   5 * time.Second,
		FileStoragePath: "",
		Restore:         false,
		Database:        "", // используем in-memory хранилище
	}

	repo := storage.NewStorage(cfg)
	svc := service.NewMetricsService(repo, 5*time.Second, nil)
	h := handlers.NewHandler(svc, nil)

	// Настраиваем роутер
	router := chi.NewRouter()
	router.Post("/update", h.UpdateHandlerJSON)
	router.Post("/value", h.ValueHandlerJSON)

	// Пример 1: Обновление gauge метрики
	gaugeValue := 23.5
	gaugeMetric := dto.Metrics{
		ID:    "temperature",
		MType: "gauge",
		Value: &gaugeValue,
	}
	body, _ := json.Marshal(gaugeMetric)
	req := httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	fmt.Printf("Обновление gauge: статус %d\n", rr.Code)

	// Пример 2: Обновление counter метрики
	counterDelta := int64(10)
	counterMetric := dto.Metrics{
		ID:    "requests",
		MType: "counter",
		Delta: &counterDelta,
	}
	body, _ = json.Marshal(counterMetric)
	req = httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	fmt.Printf("Обновление counter: статус %d\n", rr.Code)

	// Пример 3: Получение значения gauge метрики
	valueReq := dto.Metrics{ID: "temperature", MType: "gauge"}
	body, _ = json.Marshal(valueReq)
	req = httptest.NewRequest(http.MethodPost, "/value", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	var response dto.Metrics
	json.Unmarshal(rr.Body.Bytes(), &response)
	fmt.Printf("Значение temperature: %.1f\n", *response.Value)

	// Output:
	// Обновление gauge: статус 200
	// Обновление counter: статус 200
	// Значение temperature: 23.5
}

// ExampleHandler_UpdateHandlerJSON демонстрирует обновление метрики в формате JSON.
func ExampleHandler_UpdateHandlerJSON() {
	storage.TestReset()
	defer storage.Close()

	cfg := &config.ServerConfig{
		StoreInterval:   5 * time.Second,
		FileStoragePath: "",
		Restore:         false,
		Database:        "",
	}

	repo := storage.NewStorage(cfg)
	svc := service.NewMetricsService(repo, 5*time.Second, nil)
	h := handlers.NewHandler(svc, nil)

	router := chi.NewRouter()
	router.Post("/update", h.UpdateHandlerJSON)

	// Обновляем gauge метрику
	value := 42.7
	metric := dto.Metrics{
		ID:    "cpu_usage",
		MType: "gauge",
		Value: &value,
	}

	body, _ := json.Marshal(metric)
	req := httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	fmt.Printf("Статус: %d\n", rr.Code)

	// Проверяем сохраненное значение
	val, exists := repo.GetGauge(context.Background(), "cpu_usage")
	if exists {
		fmt.Printf("Сохраненное значение: %.1f\n", val)
	}

	// Output:
	// Статус: 200
	// Сохраненное значение: 42.7
}

// ExampleHandler_ValueHandlerJSON демонстрирует получение значения метрики в формате JSON.
func ExampleHandler_ValueHandlerJSON() {
	storage.TestReset()
	defer storage.Close()

	cfg := &config.ServerConfig{
		StoreInterval:   5 * time.Second,
		FileStoragePath: "",
		Restore:         false,
		Database:        "",
	}

	repo := storage.NewStorage(cfg)
	svc := service.NewMetricsService(repo, 5*time.Second, nil)
	h := handlers.NewHandler(svc, nil)

	// Предварительно сохраняем метрику
	repo.SetGauge(context.Background(), "memory_usage", 75.3)

	router := chi.NewRouter()
	router.Post("/value", h.ValueHandlerJSON)

	// Запрашиваем значение метрики
	request := dto.Metrics{
		ID:    "memory_usage",
		MType: "gauge",
	}

	body, _ := json.Marshal(request)
	req := httptest.NewRequest(http.MethodPost, "/value", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	var response dto.Metrics
	json.Unmarshal(rr.Body.Bytes(), &response)

	fmt.Printf("Статус: %d\n", rr.Code)
	fmt.Printf("Метрика: %s\n", response.ID)
	fmt.Printf("Значение: %.1f\n", *response.Value)

	// Output:
	// Статус: 200
	// Метрика: memory_usage
	// Значение: 75.3
}

// ExampleHandler_UpdateMetrics демонстрирует массовое обновление метрик.
func ExampleHandler_UpdateMetrics() {
	storage.TestReset()
	defer storage.Close()

	cfg := &config.ServerConfig{
		StoreInterval:   5 * time.Second,
		FileStoragePath: "",
		Restore:         false,
		Database:        "",
	}

	repo := storage.NewStorage(cfg)
	svc := service.NewMetricsService(repo, 5*time.Second, nil)
	h := handlers.NewHandler(svc, nil)

	router := chi.NewRouter()
	router.Post("/updates", h.UpdateMetrics)

	// Подготавливаем несколько метрик для обновления
	gaugeValue1 := 23.5
	gaugeValue2 := 56.8
	counterDelta := int64(100)

	metrics := []dto.Metrics{
		{ID: "temperature", MType: "gauge", Value: &gaugeValue1},
		{ID: "humidity", MType: "gauge", Value: &gaugeValue2},
		{ID: "total_requests", MType: "counter", Delta: &counterDelta},
	}

	body, _ := json.Marshal(metrics)
	req := httptest.NewRequest(http.MethodPost, "/updates", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	fmt.Printf("Статус: %d\n", rr.Code)

	// Проверяем сохраненные значения
	temp, _ := repo.GetGauge(context.Background(), "temperature")
	hum, _ := repo.GetGauge(context.Background(), "humidity")
	req_count, _ := repo.GetCounter(context.Background(), "total_requests")

	fmt.Printf("temperature: %.1f\n", temp)
	fmt.Printf("humidity: %.1f\n", hum)
	fmt.Printf("total_requests: %d\n", req_count)

	// Output:
	// Статус: 200
	// temperature: 23.5
	// humidity: 56.8
	// total_requests: 100
}

// ExampleHandler_UpdateHandler демонстрирует обновление метрики через URL параметры.
func ExampleHandler_UpdateHandler() {
	storage.TestReset()
	defer storage.Close()

	cfg := &config.ServerConfig{
		StoreInterval:   5 * time.Second,
		FileStoragePath: "",
		Restore:         false,
		Database:        "",
	}

	repo := storage.NewStorage(cfg)
	svc := service.NewMetricsService(repo, 5*time.Second, nil)
	h := handlers.NewHandler(svc, nil)

	router := chi.NewRouter()
	router.Post("/update/{metricType}/{metricName}/{metricValue}", h.UpdateHandler)

	// Обновляем gauge метрику через URL
	req := httptest.NewRequest(http.MethodPost, "/update/gauge/cpu_temp/78.5", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	fmt.Printf("Статус: %d\n", rr.Code)

	// Проверяем значение
	val, exists := repo.GetGauge(context.Background(), "cpu_temp")
	if exists {
		fmt.Printf("cpu_temp: %.1f\n", val)
	}

	// Output:
	// Статус: 200
	// cpu_temp: 78.5
}

// ExampleHandler_GetHandler демонстрирует получение значения метрики через URL параметры.
func ExampleHandler_GetHandler() {
	storage.TestReset()
	defer storage.Close()

	cfg := &config.ServerConfig{
		StoreInterval:   5 * time.Second,
		FileStoragePath: "",
		Restore:         false,
		Database:        "",
	}

	repo := storage.NewStorage(cfg)
	svc := service.NewMetricsService(repo, 5*time.Second, nil)
	h := handlers.NewHandler(svc, nil)

	// Предварительно сохраняем метрику
	repo.IncrementCounter(context.Background(), "requests", 42)

	router := chi.NewRouter()
	router.Get("/value/{metricType}/{metricName}", h.GetHandler)

	// Получаем значение counter метрики
	req := httptest.NewRequest(http.MethodGet, "/value/counter/requests", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	fmt.Printf("Статус: %d\n", rr.Code)
	fmt.Printf("Значение: %s\n", rr.Body.String())

	// Output:
	// Статус: 200
	// Значение: 42
}
