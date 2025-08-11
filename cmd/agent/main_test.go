package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMetricsSender_SendJSON(t *testing.T) {
	// Создаем тестовый сервер
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Проверяем путь
		assert.Equal(t, "/update", r.URL.Path, "Expected path /update")
		// Проверяем метод
		assert.Equal(t, http.MethodPost, r.Method, "Expected POST method")
		// Проверяем заголовок Content-Type
		contentType := r.Header.Get("Content-Type")
		assert.Equal(t, "application/json", contentType, "Expected Content-Type: application/json")

		// Декодируем тело
		var metric Metrics
		err := json.NewDecoder(r.Body).Decode(&metric)
		if err != nil {
			t.Fatalf("Failed to decode JSON: %v", err)
		}

		// Проверяем метрику
		assert.Equal(t, "testMetric", metric.ID, "Expected metric ID 'testMetric'")
		assert.Equal(t, "gauge", metric.MType, "Expected metric type 'gauge'")
		assert.NotNil(t, metric.Value, "Value should not be nil")
		assert.Equal(t, 42.5, *metric.Value, "Expected value 42.5")

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Создаем MetricsSender с адресом тестового сервера
	sender := NewMetricsSender(server.Listener.Addr().String())

	// Создаем тестовую метрику
	value := 42.5
	metric := Metrics{ID: "testMetric", MType: "gauge", Value: &value}

	// Отправляем метрику
	err := sender.SendJSON(metric)
	assert.NoError(t, err, "Sending should not produce error")
}
