package handlers

import (
	"encoding/json"
	"github.com/SamSafonov2025/metrics-tpl/internal/consts"
	"github.com/SamSafonov2025/metrics-tpl/internal/dto"
	"github.com/SamSafonov2025/metrics-tpl/internal/service"
	"github.com/go-chi/chi/v5"
	"net/http"
	"strconv"
	"strings"
)

type Handler struct {
	Svc service.MetricsService
}

func NewHandler(svc service.MetricsService) *Handler { return &Handler{Svc: svc} }

func (h *Handler) Ping(rw http.ResponseWriter, r *http.Request) {
	if err := h.Svc.Ping(r.Context()); err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
	rw.WriteHeader(http.StatusOK)
}

func (h *Handler) HomeHandler(rw http.ResponseWriter, r *http.Request) {
	gauges, counters, err := h.Svc.List(r.Context())
	if err != nil {
		http.Error(rw, "internal error", http.StatusInternalServerError)
		return
	}

	var sb strings.Builder
	sb.WriteString("<h4>Gauges</h4>")
	for name, v := range gauges {
		sb.WriteString(name + ": " + strconv.FormatFloat(v, 'f', -1, 64) + "</br>")
	}
	sb.WriteString("<h4>Counters</h4>")
	for name, v := range counters {
		sb.WriteString(name + ": " + strconv.FormatInt(v, 10) + "</br>")
	}
	rw.Header().Set("Content-Type", "text/html; charset=utf-8")
	rw.WriteHeader(http.StatusOK)
	_, _ = rw.Write([]byte(sb.String()))
}

func (h *Handler) UpdateHandler(rw http.ResponseWriter, r *http.Request) {
	m := dto.Metrics{
		ID:    chi.URLParam(r, "metricName"),
		MType: chi.URLParam(r, "metricType"),
	}
	val := chi.URLParam(r, "metricValue")
	switch m.MType {
	case consts.MetricTypeGauge:
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			m.Value = &f
		} else {
			http.Error(rw, "Bad request", http.StatusBadRequest)
			return
		}
	case consts.MetricTypeCounter:
		if d, err := strconv.ParseInt(val, 10, 64); err == nil {
			m.Delta = &d
		} else {
			http.Error(rw, "Bad request", http.StatusBadRequest)
			return
		}
	default:
		http.Error(rw, "Invalid metric type", http.StatusBadRequest)
		return
	}
	if _, err := h.Svc.Update(r.Context(), m); err != nil {
		http.Error(rw, "internal error", http.StatusInternalServerError)
		return
	}
	rw.WriteHeader(http.StatusOK)
}

func (h *Handler) GetHandler(rw http.ResponseWriter, r *http.Request) {
	typ := chi.URLParam(r, "metricType")
	id := chi.URLParam(r, "metricName")

	m, err := h.Svc.Get(r.Context(), typ, id)
	if err == service.ErrInvalidType {
		http.Error(rw, "Invalid metric type", http.StatusBadRequest)
		return
	}
	if err == service.ErrNotFound {
		http.Error(rw, "Metric not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(rw, "internal error", http.StatusInternalServerError)
		return
	}

	rw.Header().Set("Content-type", "text/plain")
	if m.MType == consts.MetricTypeGauge {
		_, _ = rw.Write([]byte(strconv.FormatFloat(*m.Value, 'f', -1, 64)))
	}
	if m.MType == consts.MetricTypeCounter {
		_, _ = rw.Write([]byte(strconv.FormatInt(*m.Delta, 10)))
	}
}

func (h *Handler) UpdateHandlerJSON(rw http.ResponseWriter, r *http.Request) {
	var m dto.Metrics
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		http.Error(rw, "Bad request", http.StatusBadRequest)
		return
	}
	m, err := h.Svc.Update(r.Context(), m)
	if err == service.ErrInvalidType || err == service.ErrBadValue {
		http.Error(rw, "Bad request", http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(rw, "internal error", http.StatusInternalServerError)
		return
	}

	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(rw).Encode(m)
}

func (h *Handler) ValueHandlerJSON(rw http.ResponseWriter, r *http.Request) {
	var req dto.Metrics
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(rw, "Bad request", http.StatusBadRequest)
		return
	}
	m, err := h.Svc.Get(r.Context(), req.MType, req.ID)
	if err == service.ErrInvalidType {
		http.Error(rw, "Invalid metric type", http.StatusBadRequest)
		return
	}
	if err == service.ErrNotFound {
		http.Error(rw, "Metric not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(rw, "internal error", http.StatusInternalServerError)
		return
	}

	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(rw).Encode(m)
}

func (h *Handler) UpdateMetrics(rw http.ResponseWriter, r *http.Request) {
	var body []dto.Metrics
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}
	if err := h.Svc.UpdateBatch(r.Context(), body); err != nil {
		if err == service.ErrInvalidType || err == service.ErrBadValue {
			http.Error(rw, "Bad request", http.StatusBadRequest)
			return
		}
		http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	rw.WriteHeader(http.StatusOK)
}
