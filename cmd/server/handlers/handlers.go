package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/SamSafonov2025/metrics-tpl/internal/dto"
	"github.com/SamSafonov2025/metrics-tpl/internal/interfaces"
	"github.com/SamSafonov2025/metrics-tpl/internal/postgres"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	Storage interfaces.Store
}

func NewHandler(storage interfaces.Store) *Handler { return &Handler{Storage: storage} }

func (h *Handler) Ping(rw http.ResponseWriter, r *http.Request) {
	if err := postgres.Pool.Ping(r.Context()); err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
	rw.WriteHeader(http.StatusOK)
}

func (h *Handler) HomeHandler(rw http.ResponseWriter, r *http.Request) {
	var sb strings.Builder
	sb.WriteString("<h4>Gauges</h4>")

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	for name, v := range h.Storage.GetAllGauges(ctx) {
		sb.WriteString(name + ": " + strconv.FormatFloat(v, 'f', -1, 64) + "</br>")
	}

	sb.WriteString("<h4>Counters</h4>")
	for name, v := range h.Storage.GetAllCounters(ctx) {
		sb.WriteString(name + ": " + strconv.FormatInt(v, 10) + "</br>")
	}

	rw.Header().Set("Content-Type", "text/html; charset=utf-8")
	rw.WriteHeader(http.StatusOK)
	_, _ = rw.Write([]byte(sb.String()))
}

func (h *Handler) UpdateHandler(rw http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "metricType")
	metricName := chi.URLParam(r, "metricName")
	metricValue := chi.URLParam(r, "metricValue")

	switch metricType {
	case "counter":
		val, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			http.Error(rw, "Bad request", http.StatusBadRequest)
			return
		}
		if err := h.Storage.IncrementCounter(r.Context(), metricName, val); err != nil {
			http.Error(rw, "internal error", http.StatusInternalServerError)
			return
		}

	case "gauge":
		val, err := strconv.ParseFloat(metricValue, 64)
		if err != nil {
			http.Error(rw, "Bad request", http.StatusBadRequest)
			return
		}
		if err := h.Storage.SetGauge(r.Context(), metricName, val); err != nil {
			http.Error(rw, "internal error", http.StatusInternalServerError)
			return
		}

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
		if value, ok := h.Storage.GetGauge(r.Context(), metricName); ok {
			rw.Header().Set("Content-type", "text/plain")
			_, _ = rw.Write([]byte(strconv.FormatFloat(value, 'f', -1, 64)))
		} else {
			http.Error(rw, "Metric not found", http.StatusNotFound)
		}
	case "counter":
		if value, ok := h.Storage.GetCounter(r.Context(), metricName); ok {
			rw.Header().Set("Content-type", "text/plain")
			_, _ = rw.Write([]byte(strconv.FormatInt(value, 10)))
		} else {
			http.Error(rw, "Metric not found", http.StatusNotFound)
		}
	}
}

func (h *Handler) UpdateHandlerJSON(rw http.ResponseWriter, r *http.Request) {
	var metric dto.Metrics
	if err := json.NewDecoder(r.Body).Decode(&metric); err != nil {
		http.Error(rw, "Bad request", http.StatusBadRequest)
		return
	}

	switch metric.MType {
	case "gauge":
		if metric.Value == nil {
			http.Error(rw, "Missing value for gauge", http.StatusBadRequest)
			return
		}
		if err := h.Storage.SetGauge(r.Context(), metric.ID, *metric.Value); err != nil {
			http.Error(rw, "internal error", http.StatusInternalServerError)
			return
		}
		if v, _ := h.Storage.GetGauge(r.Context(), metric.ID); true {
			metric.Value = &v
		}
	case "counter":
		if metric.Delta == nil {
			http.Error(rw, "Missing delta for counter", http.StatusBadRequest)
			return
		}
		if err := h.Storage.IncrementCounter(r.Context(), metric.ID, *metric.Delta); err != nil {
			http.Error(rw, "internal error", http.StatusInternalServerError)
			return
		}
		if v, _ := h.Storage.GetCounter(r.Context(), metric.ID); true {
			metric.Delta = &v
		}
	default:
		http.Error(rw, "Invalid metric type", http.StatusBadRequest)
		return
	}

	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(rw).Encode(metric)
}

func (h *Handler) ValueHandlerJSON(rw http.ResponseWriter, r *http.Request) {
	var metric dto.Metrics
	if err := json.NewDecoder(r.Body).Decode(&metric); err != nil {
		http.Error(rw, "Bad request", http.StatusBadRequest)
		return
	}

	switch metric.MType {
	case "gauge":
		value, exists := h.Storage.GetGauge(r.Context(), metric.ID)
		if !exists {
			http.Error(rw, "Metric not found", http.StatusNotFound)
			return
		}
		metric.Value = &value
	case "counter":
		value, exists := h.Storage.GetCounter(r.Context(), metric.ID)
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

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}
	// ограничим серверную обработку по времени
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := h.Storage.SetMetrics(ctx, body); err != nil {
		http.Error(rw, "internal error", http.StatusInternalServerError)
		return
	}
	rw.WriteHeader(http.StatusOK)
}
