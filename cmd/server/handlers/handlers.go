package handlers

import (
	"net/http"
	"strconv"

	"github.com/SamSafonov2025/metrics-tpl/cmd/server/storage"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	storage *storage.MemStorage
}

func NewHandler(storage *storage.MemStorage) *Handler {
	return &Handler{storage: storage}
}

func (h *Handler) HomeHandler(rw http.ResponseWriter, r *http.Request) {
	body := "<h4>Gauges</h4>"
	for gaugeName, value := range h.storage.GetAllGauges() {
		body += gaugeName + ": " + strconv.FormatFloat(value, 'f', -1, 64) + "</br>"
	}
	body += "<h4>Counters</h4>"

	for counterName, value := range h.storage.GetAllCounters() {
		body += counterName + ": " + strconv.FormatInt(value, 10) + "</br>"
	}
	rw.Header().Set("Content-Type", "text/html; charset=utf-8")
	rw.Write([]byte(body))
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
		h.storage.IncrementCounter(metricName, value)
	case "gauge":
		value, err := strconv.ParseFloat(metricValue, 64)
		if err != nil {
			http.Error(rw, "Bad request", http.StatusBadRequest)
			return
		}
		h.storage.SetGauge(metricName, value)
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
		value, exists := h.storage.GetGauge(metricName)
		if !exists {
			http.Error(rw, "Metric not found", http.StatusNotFound)
			return
		}
		rw.Header().Set("Content-type", "text/plain")
		rw.Write([]byte(strconv.FormatFloat(value, 'f', -1, 64)))
	case "counter":
		value, exists := h.storage.GetCounter(metricName)
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
