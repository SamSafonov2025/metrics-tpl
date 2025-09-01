package handlers

import (
	"context"
	"encoding/json"
	"github.com/SamSafonov2025/metrics-tpl/internal/dto"
	"github.com/SamSafonov2025/metrics-tpl/internal/postgres"

	"github.com/SamSafonov2025/metrics-tpl/internal/interfaces"

	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

type Metrics struct {
	ID    string   `json:"id"`
	MType string   `json:"type"`
	Delta *int64   `json:"delta,omitempty"`
	Value *float64 `json:"value,omitempty"`
}

type Handler struct {
	Storage interfaces.Store
}

func NewHandler(storage interfaces.Store) *Handler {
	return &Handler{Storage: storage}
}

func (h *Handler) Ping(rw http.ResponseWriter, _ *http.Request) {
	err := postgres.Pool.Ping(context.Background())
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
	}
	rw.WriteHeader(http.StatusOK)
}

func (h *Handler) HomeHandler(rw http.ResponseWriter, r *http.Request) {
	body := "<h4>Gauges</h4>"
	for gaugeName, value := range h.Storage.GetAllGauges() {
		body += gaugeName + ": " + strconv.FormatFloat(value, 'f', -1, 64) + "</br>"
	}
	body += "<h4>Counters</h4>"

	for counterName, value := range h.Storage.GetAllCounters() {
		body += counterName + ": " + strconv.FormatInt(value, 10) + "</br>"
	}
	rw.Header().Set("Content-Type", "text/html; charset=utf-8")
	rw.WriteHeader(http.StatusOK)
	_, _ = rw.Write([]byte(body))
}

func (h *Handler) UpdateHandler(rw http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "metricType")
	metricName := chi.URLParam(r, "metricName")
	metricValue := chi.URLParam(r, "metricValue")

	switch metricType {
	case "counter":
		value, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			http.Error(rw, "Bad request", http.StatusBadRequest)
			return
		}
		h.Storage.IncrementCounter(metricName, value)
	case "gauge":
		value, err := strconv.ParseFloat(metricValue, 64)
		if err != nil {
			http.Error(rw, "Bad request", http.StatusBadRequest)
			return
		}
		h.Storage.SetGauge(metricName, value)
	default:
		http.Error(rw, "Bad request", http.StatusBadRequest)
		return
	}
	rw.WriteHeader(http.StatusOK)
}

func (h *Handler) GetHandler(rw http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "metricType")
	metricName := chi.URLParam(r, "metricName")

	if metricType != "gauge" && metricType != "counter" {
		http.Error(rw, "Invalid metric type", http.StatusBadRequest)
		return
	}

	switch metricType {
	case "gauge":
		value, exists := h.Storage.GetGauge(metricName)
		if !exists {
			http.Error(rw, "Metric not found", http.StatusNotFound)
			return
		}
		rw.Header().Set("Content-type", "text/plain")
		rw.Write([]byte(strconv.FormatFloat(value, 'f', -1, 64)))
	case "counter":
		value, exists := h.Storage.GetCounter(metricName)
		if !exists {
			http.Error(rw, "Metric not found", http.StatusNotFound)
			return
		}
		rw.Header().Set("Content-type", "text/plain")
		rw.Write([]byte(strconv.FormatInt(value, 10)))
	default:
		http.Error(rw, "Invalid metric type", http.StatusBadRequest)
		return
	}
}

func (h *Handler) UpdateHandlerJSON(rw http.ResponseWriter, r *http.Request) {
	var metric Metrics
	err := json.NewDecoder(r.Body).Decode(&metric)
	if err != nil {
		http.Error(rw, "Bad request", http.StatusBadRequest)
		return
	}

	switch metric.MType {
	case "gauge":
		if metric.Value == nil {
			http.Error(rw, "Missing value for gauge", http.StatusBadRequest)
			return
		}
		h.Storage.SetGauge(metric.ID, *metric.Value)
		value, _ := h.Storage.GetGauge(metric.ID)
		metric.Value = &value
	case "counter":
		if metric.Delta == nil {
			http.Error(rw, "Missing delta for counter", http.StatusBadRequest)
			return
		}
		h.Storage.IncrementCounter(metric.ID, *metric.Delta)
		value, _ := h.Storage.GetCounter(metric.ID)
		metric.Delta = &value
	default:
		http.Error(rw, "Invalid metric type", http.StatusBadRequest)
		return
	}

	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(rw).Encode(metric)
}

func (h *Handler) ValueHandlerJSON(rw http.ResponseWriter, r *http.Request) {
	var metric Metrics
	err := json.NewDecoder(r.Body).Decode(&metric)
	if err != nil {
		http.Error(rw, "Bad request", http.StatusBadRequest)
		return
	}

	switch metric.MType {
	case "gauge":
		value, exists := h.Storage.GetGauge(metric.ID)
		if !exists {
			http.Error(rw, "Metric not found", http.StatusNotFound)
			return
		}
		metric.Value = &value
	case "counter":
		value, exists := h.Storage.GetCounter(metric.ID)
		if !exists {
			http.Error(rw, "Metric not found", http.StatusNotFound)
			return
		}
		metric.Delta = &value
	default:
		http.Error(rw, "Invalid metric type", http.StatusBadRequest)
		return
	}

	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(rw).Encode(metric)
}

func (h *Handler) UpdateMetrics(rw http.ResponseWriter, r *http.Request) {
	var body []dto.Metrics
	rw.Header().Set("Content-Type", "application/json")
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}
	h.Storage.SetMetrics(body)
	rw.WriteHeader(http.StatusOK)
}
