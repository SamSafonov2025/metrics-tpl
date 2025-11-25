// Package handlers содержит HTTP-обработчики для сервера метрик.
//
// Пакет предоставляет набор хендлеров для работы с метриками через REST API:
// обновление метрик, получение значений, массовые операции и проверку состояния БД.
package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/SamSafonov2025/metrics-tpl/internal/audit"
	"github.com/SamSafonov2025/metrics-tpl/internal/consts"
	"github.com/SamSafonov2025/metrics-tpl/internal/dto"
	"github.com/SamSafonov2025/metrics-tpl/internal/logger"
	"github.com/SamSafonov2025/metrics-tpl/internal/service"
)

var (
	// bufferPool переиспользует буферы для JSON encoding/decoding
	bufferPool = sync.Pool{
		New: func() interface{} {
			return new(bytes.Buffer)
		},
	}

	// stringSlicePool переиспользует слайсы строк для метрик
	stringSlicePool = sync.Pool{
		New: func() interface{} {
			s := make([]string, 0, 100)
			return &s
		},
	}
)

// Handler содержит зависимости для обработки HTTP-запросов метрик.
// Включает сервисный слой для работы с метриками и издателя событий аудита.
type Handler struct {
	Svc            service.MetricsService
	AuditPublisher *audit.AuditPublisher
}

// NewHandler создает новый экземпляр Handler с заданным сервисом и издателем аудита.
// Параметр auditPublisher может быть nil, если аудит не требуется.
func NewHandler(svc service.MetricsService, auditPublisher *audit.AuditPublisher) *Handler {
	return &Handler{Svc: svc, AuditPublisher: auditPublisher}
}

// Ping проверяет доступность базы данных.
// Возвращает HTTP 200 при успешном подключении или HTTP 500 при ошибке.
//
// Endpoint: GET /ping
func (h *Handler) Ping(rw http.ResponseWriter, r *http.Request) {
	if err := h.Svc.Ping(r.Context()); err != nil {
		logger.GetLogger().Warn("Ping failed", zapError(err))
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
	rw.WriteHeader(http.StatusOK)
}

// HomeHandler возвращает HTML-страницу со списком всех метрик.
// Отображает gauge и counter метрики в виде форматированного списка.
//
// Endpoint: GET /
func (h *Handler) HomeHandler(rw http.ResponseWriter, r *http.Request) {
	gauges, counters, err := h.Svc.List(r.Context())
	if err != nil {
		logger.GetLogger().Error("List metrics failed", zapError(err))
		http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	var sb strings.Builder
	// Предаллокируем память для улучшения производительности
	sb.Grow(len(gauges)*50 + len(counters)*50 + 50)

	sb.WriteString("<h4>Gauges</h4>")
	for name, v := range gauges {
		sb.WriteString(name)
		sb.WriteString(": ")
		sb.WriteString(strconv.FormatFloat(v, 'f', -1, 64))
		sb.WriteString("</br>")
	}
	sb.WriteString("<h4>Counters</h4>")
	for name, v := range counters {
		sb.WriteString(name)
		sb.WriteString(": ")
		sb.WriteString(strconv.FormatInt(v, 10))
		sb.WriteString("</br>")
	}
	rw.Header().Set("Content-Type", "text/html; charset=utf-8")
	rw.WriteHeader(http.StatusOK)
	_, _ = rw.Write([]byte(sb.String()))
}

// UpdateHandler обновляет метрику через URL-параметры (устаревший формат).
// Принимает тип метрики, имя и значение из URL.
// Поддерживает типы "gauge" (float64) и "counter" (int64).
//
// Endpoint: POST /update/{metricType}/{metricName}/{metricValue}
//
// Возвращает:
//   - HTTP 200 при успешном обновлении
//   - HTTP 400 при некорректных параметрах
//   - HTTP 500 при внутренней ошибке
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
			logger.GetLogger().Warn("UpdateHandler bad gauge value", zapString("val", val), zapError(err))
			http.Error(rw, "Bad request", http.StatusBadRequest)
			return
		}
	case consts.MetricTypeCounter:
		if d, err := strconv.ParseInt(val, 10, 64); err == nil {
			m.Delta = &d
		} else {
			logger.GetLogger().Warn("UpdateHandler bad counter delta", zapString("val", val), zapError(err))
			http.Error(rw, "Bad request", http.StatusBadRequest)
			return
		}
	default:
		logger.GetLogger().Warn("UpdateHandler invalid metric type", zapString("type", m.MType))
		http.Error(rw, "Invalid metric type", http.StatusBadRequest)
		return
	}
	if _, err := h.Svc.Update(r.Context(), m); err != nil {
		logger.GetLogger().Error("UpdateHandler Update failed", zapError(err))
		http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	// Отправляем событие аудита после успешной обработки
	h.sendAuditEvent(r, []string{m.ID})
	rw.WriteHeader(http.StatusOK)
}

// GetHandler возвращает значение метрики в текстовом формате.
// Принимает тип метрики и имя из URL-параметров.
//
// Endpoint: GET /value/{metricType}/{metricName}
//
// Возвращает:
//   - HTTP 200 и значение метрики в теле ответа
//   - HTTP 400 при некорректном типе метрики
//   - HTTP 404 если метрика не найдена
//   - HTTP 500 при внутренней ошибке
func (h *Handler) GetHandler(rw http.ResponseWriter, r *http.Request) {
	typ := chi.URLParam(r, "metricType")
	id := chi.URLParam(r, "metricName")

	m, err := h.Svc.Get(r.Context(), typ, id)
	if err == service.ErrInvalidType {
		logger.GetLogger().Warn("GetHandler invalid type", zapString("type", typ))
		http.Error(rw, "Invalid metric type", http.StatusBadRequest)
		return
	}
	if err == service.ErrNotFound {
		logger.GetLogger().Warn("GetHandler metric not found", zapString("type", typ), zapString("id", id))
		http.Error(rw, "Metric not found", http.StatusNotFound)
		return
	}
	if err != nil {
		logger.GetLogger().Error("GetHandler Get failed", zapError(err))
		http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
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

// UpdateHandlerJSON обновляет метрику в формате JSON.
// Принимает метрику в теле запроса и возвращает обновленные данные.
//
// Endpoint: POST /update/
//
// Формат тела запроса:
//
//	{"id":"metricName","type":"gauge","value":123.45}
//	{"id":"metricName","type":"counter","delta":10}
//
// Возвращает:
//   - HTTP 200 и обновленную метрику в JSON
//   - HTTP 400 при некорректных данных
//   - HTTP 500 при внутренней ошибке
func (h *Handler) UpdateHandlerJSON(rw http.ResponseWriter, r *http.Request) {
	var m dto.Metrics
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		logger.GetLogger().Warn("UpdateHandlerJSON decode error", zapError(err))
		http.Error(rw, "Bad request", http.StatusBadRequest)
		return
	}
	m, err := h.Svc.Update(r.Context(), m)
	if err == service.ErrInvalidType || err == service.ErrBadValue {
		logger.GetLogger().Warn("UpdateHandlerJSON bad metric", zapError(err))
		http.Error(rw, "Bad request", http.StatusBadRequest)
		return
	}
	if err != nil {
		logger.GetLogger().Error("UpdateHandlerJSON Update failed", zapError(err))
		http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// Отправляем событие аудита после успешной обработки
	h.sendAuditEvent(r, []string{m.ID})

	// Используем буфер из пула для JSON encoding
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufferPool.Put(buf)

	if err := json.NewEncoder(buf).Encode(m); err != nil {
		logger.GetLogger().Error("UpdateHandlerJSON encode error", zapError(err))
		http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusOK)
	_, _ = rw.Write(buf.Bytes())
}

// ValueHandlerJSON возвращает значение метрики в формате JSON.
// Принимает запрос с типом и именем метрики в теле запроса.
//
// Endpoint: POST /value/
//
// Формат тела запроса:
//
//	{"id":"metricName","type":"gauge"}
//	{"id":"metricName","type":"counter"}
//
// Возвращает:
//   - HTTP 200 и метрику в JSON с её текущим значением
//   - HTTP 400 при некорректном типе метрики
//   - HTTP 404 если метрика не найдена
//   - HTTP 500 при внутренней ошибке
func (h *Handler) ValueHandlerJSON(rw http.ResponseWriter, r *http.Request) {
	var req dto.Metrics
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.GetLogger().Warn("ValueHandlerJSON decode error", zapError(err))
		http.Error(rw, "Bad request", http.StatusBadRequest)
		return
	}
	m, err := h.Svc.Get(r.Context(), req.MType, req.ID)
	if err == service.ErrInvalidType {
		logger.GetLogger().Warn("ValueHandlerJSON invalid type", zapString("type", req.MType))
		http.Error(rw, "Invalid metric type", http.StatusBadRequest)
		return
	}
	if err == service.ErrNotFound {
		logger.GetLogger().Warn("ValueHandlerJSON metric not found", zapString("type", req.MType), zapString("id", req.ID))
		http.Error(rw, "Metric not found", http.StatusNotFound)
		return
	}
	if err != nil {
		logger.GetLogger().Error("ValueHandlerJSON Get failed", zapError(err))
		http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// Используем буфер из пула для JSON encoding
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufferPool.Put(buf)

	if err := json.NewEncoder(buf).Encode(m); err != nil {
		logger.GetLogger().Error("ValueHandlerJSON encode error", zapError(err))
		http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusOK)
	_, _ = rw.Write(buf.Bytes())
}

// UpdateMetrics обновляет несколько метрик одним запросом.
// Принимает массив метрик в теле запроса и обрабатывает их атомарно.
//
// Endpoint: POST /updates/
//
// Формат тела запроса:
//
//	[
//	  {"id":"metric1","type":"gauge","value":123.45},
//	  {"id":"metric2","type":"counter","delta":10}
//	]
//
// Возвращает:
//   - HTTP 200 при успешном обновлении всех метрик
//   - HTTP 400 при некорректных данных в любой из метрик
//   - HTTP 500 при внутренней ошибке
func (h *Handler) UpdateMetrics(rw http.ResponseWriter, r *http.Request) {
	var body []dto.Metrics
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		logger.GetLogger().Warn("UpdateMetrics decode error", zapError(err))
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}
	if err := h.Svc.UpdateBatch(r.Context(), body); err != nil {
		if err == service.ErrInvalidType || err == service.ErrBadValue {
			logger.GetLogger().Warn("UpdateMetrics bad metric in batch", zapError(err))
			http.Error(rw, "Bad request", http.StatusBadRequest)
			return
		}
		logger.GetLogger().Error("UpdateMetrics UpdateBatch failed", zapError(err))
		http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// Отправляем событие аудита после успешной обработки
	// Используем пул для слайса имен метрик
	metricNamesPtr := stringSlicePool.Get().(*[]string)
	metricNames := (*metricNamesPtr)[:0] // Обнуляем длину, но сохраняем capacity
	defer stringSlicePool.Put(metricNamesPtr)

	// Расширяем слайс если нужно
	if cap(metricNames) < len(body) {
		metricNames = make([]string, 0, len(body))
		*metricNamesPtr = metricNames
	}

	for _, m := range body {
		metricNames = append(metricNames, m.ID)
	}
	h.sendAuditEvent(r, metricNames)

	rw.WriteHeader(http.StatusOK)
}

// маленькие помощники, чтобы не тащить zap в каждое место
func zapError(err error) zap.Field    { return zap.Error(err) }
func zapString(k, v string) zap.Field { return zap.String(k, v) }

// getClientIP извлекает IP адрес клиента из запроса
func getClientIP(r *http.Request) string {
	// Пробуем получить IP из заголовков прокси
	ip := r.Header.Get("X-Real-IP")
	if ip == "" {
		ip = r.Header.Get("X-Forwarded-For")
		if ip != "" {
			// X-Forwarded-For может содержать список IP, берем первый
			if idx := strings.Index(ip, ","); idx != -1 {
				ip = ip[:idx]
			}
		}
	}
	// Если заголовков нет, берем из RemoteAddr
	if ip == "" {
		ip = r.RemoteAddr
		// RemoteAddr может содержать порт, убираем его
		if idx := strings.LastIndex(ip, ":"); idx != -1 {
			ip = ip[:idx]
		}
	}
	return strings.TrimSpace(ip)
}

// sendAuditEvent отправляет событие аудита
func (h *Handler) sendAuditEvent(r *http.Request, metricNames []string) {
	if h.AuditPublisher == nil {
		return
	}
	event := audit.AuditEvent{
		Timestamp: time.Now().Unix(),
		Metrics:   metricNames,
		IPAddress: getClientIP(r),
	}
	h.AuditPublisher.NotifyAll(event)
}
