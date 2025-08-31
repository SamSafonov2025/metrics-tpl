package handlers

import (
	"bytes"
	"encoding/json"
	"github.com/SamSafonov2025/metrics-tpl/cmd/server/storage"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

func TestUpdateHandlerGaugeSuccess(t *testing.T) {
	s := storage.NewStorage("", 0, false)
	h := NewHandler(s)
	router := chi.NewRouter()
	router.Post("/update/{metricType}/{metricName}/{metricValue}", h.UpdateHandler)

	req := httptest.NewRequest(http.MethodPost, "/update/gauge/temperature/23.5", nil)
	req.Header.Set("Content-Type", "text/plain")

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	value, exists := s.GetGauge("temperature")
	assert.True(t, exists)
	assert.Equal(t, 23.5, value)
}

func TestUpdateHandlerCounterSuccess(t *testing.T) {
	s := storage.NewStorage("", 0, false)
	h := NewHandler(s)
	router := chi.NewRouter()
	router.Post("/update/{metricType}/{metricName}/{metricValue}", h.UpdateHandler)

	req := httptest.NewRequest(http.MethodPost, "/update/counter/hits/10", nil)
	req.Header.Set("Content-Type", "text/plain")

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	value, exists := s.GetCounter("hits")
	assert.True(t, exists)
	assert.Equal(t, int64(10), value)
}

func TestUpdateHandlerInvalidMetricType(t *testing.T) {
	s := storage.NewStorage("", 0, false)
	h := NewHandler(s)
	router := chi.NewRouter()
	router.Post("/update/{metricType}/{metricName}/{metricValue}", h.UpdateHandler)

	req := httptest.NewRequest(http.MethodPost, "/update/unknown/temperature/23.5", nil)
	req.Header.Set("Content-Type", "text/plain")

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestGetHandlerGaugeSuccess(t *testing.T) {
	s := storage.NewStorage("", 0, false)
	s.SetGauge("temperature", 23.5)
	h := NewHandler(s)
	router := chi.NewRouter()
	router.Get("/value/{metricType}/{metricName}", h.GetHandler)

	req := httptest.NewRequest(http.MethodGet, "/value/gauge/temperature", nil)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "23.5", rr.Body.String())
}

func TestGetHandlerCounterSuccess(t *testing.T) {
	s := storage.NewStorage("", 0, false)
	s.IncrementCounter("hits", 10)
	h := NewHandler(s)
	router := chi.NewRouter()
	router.Get("/value/{metricType}/{metricName}", h.GetHandler)

	req := httptest.NewRequest(http.MethodGet, "/value/counter/hits", nil)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "10", rr.Body.String())
}

func TestGetHandlerMetricNotFound(t *testing.T) {
	s := storage.NewStorage("", 0, false)
	h := NewHandler(s)
	router := chi.NewRouter()
	router.Get("/value/{metricType}/{metricName}", h.GetHandler)

	req := httptest.NewRequest(http.MethodGet, "/value/gauge/unknown", nil)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestHomeHandle(t *testing.T) {
	s := storage.NewStorage("", 0, false)
	s.SetGauge("temperature", 23.5)
	s.IncrementCounter("hits", 10)
	h := NewHandler(s)
	router := chi.NewRouter()
	router.Get("/", h.HomeHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	body := rr.Body.String()

	assert.Contains(t, body, "Gauges")
	assert.Contains(t, body, "temperature: 23.5")
	assert.Contains(t, body, "Counters")
	assert.Contains(t, body, "hits: 10")
}

func TestUpdateHandlerJSON_GaugeSuccess(t *testing.T) {
	s := storage.NewStorage("", 0, false)
	h := NewHandler(s)
	router := chi.NewRouter()
	router.Post("/update", h.UpdateHandlerJSON)

	metric := Metrics{ID: "temperature", MType: "gauge", Value: new(float64)}
	*metric.Value = 23.5
	body, _ := json.Marshal(metric)

	req := httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var response Metrics
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "temperature", response.ID)
	assert.Equal(t, "gauge", response.MType)
	assert.Equal(t, 23.5, *response.Value)

	value, exists := s.GetGauge("temperature")
	assert.True(t, exists)
	assert.Equal(t, 23.5, value)
}

func TestUpdateHandlerJSON_CounterSuccess(t *testing.T) {
	s := storage.NewStorage("", 0, false)
	h := NewHandler(s)
	router := chi.NewRouter()
	router.Post("/update", h.UpdateHandlerJSON)

	metric := Metrics{ID: "hits", MType: "counter", Delta: new(int64)}
	*metric.Delta = 10
	body, _ := json.Marshal(metric)

	req := httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var response Metrics
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "hits", response.ID)
	assert.Equal(t, "counter", response.MType)
	assert.Equal(t, int64(10), *response.Delta)

	value, exists := s.GetCounter("hits")
	assert.True(t, exists)
	assert.Equal(t, int64(10), value)
}

func TestUpdateHandlerJSON_InvalidType(t *testing.T) {
	s := storage.NewStorage("", 0, false)
	h := NewHandler(s)
	router := chi.NewRouter()
	router.Post("/update", h.UpdateHandlerJSON)

	metric := Metrics{ID: "invalid", MType: "invalid", Value: new(float64)}
	*metric.Value = 1.0
	body, _ := json.Marshal(metric)

	req := httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestUpdateHandlerJSON_MissingValue(t *testing.T) {
	s := storage.NewStorage("", 0, false)
	h := NewHandler(s)
	router := chi.NewRouter()
	router.Post("/update", h.UpdateHandlerJSON)

	metric := Metrics{ID: "temperature", MType: "gauge"}
	body, _ := json.Marshal(metric)

	req := httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestUpdateHandlerJSON_MissingDelta(t *testing.T) {
	s := storage.NewStorage("", 0, false)
	h := NewHandler(s)
	router := chi.NewRouter()
	router.Post("/update", h.UpdateHandlerJSON)

	metric := Metrics{ID: "hits", MType: "counter"}
	body, _ := json.Marshal(metric)

	req := httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestValueHandlerJSON_GaugeSuccess(t *testing.T) {
	s := storage.NewStorage("", 0, false)
	s.SetGauge("temperature", 23.5)
	h := NewHandler(s)
	router := chi.NewRouter()
	router.Post("/value", h.ValueHandlerJSON)

	metric := Metrics{ID: "temperature", MType: "gauge"}
	body, _ := json.Marshal(metric)

	req := httptest.NewRequest(http.MethodPost, "/value", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var response Metrics
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "temperature", response.ID)
	assert.Equal(t, "gauge", response.MType)
	assert.Equal(t, 23.5, *response.Value)
}

func TestValueHandlerJSON_CounterSuccess(t *testing.T) {
	s := storage.NewStorage("", 0, false)
	s.IncrementCounter("hits", 10)
	h := NewHandler(s)
	router := chi.NewRouter()
	router.Post("/value", h.ValueHandlerJSON)

	metric := Metrics{ID: "hits", MType: "counter"}
	body, _ := json.Marshal(metric)

	req := httptest.NewRequest(http.MethodPost, "/value", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var response Metrics
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "hits", response.ID)
	assert.Equal(t, "counter", response.MType)
	assert.Equal(t, int64(10), *response.Delta)
}

func TestValueHandlerJSON_InvalidType(t *testing.T) {
	s := storage.NewStorage("", 0, false)
	h := NewHandler(s)
	router := chi.NewRouter()
	router.Post("/value", h.ValueHandlerJSON)

	metric := Metrics{ID: "invalid", MType: "invalid"}
	body, _ := json.Marshal(metric)

	req := httptest.NewRequest(http.MethodPost, "/value", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestValueHandlerJSON_MetricNotFound(t *testing.T) {
	s := storage.NewStorage("", 0, false)
	h := NewHandler(s)
	router := chi.NewRouter()
	router.Post("/value", h.ValueHandlerJSON)

	metric := Metrics{ID: "not_found", MType: "gauge"}
	body, _ := json.Marshal(metric)

	req := httptest.NewRequest(http.MethodPost, "/value", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}
