package logger

import (
	"bytes"
	"io"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"
)

var (
	loggerInstance *zap.Logger
	once           sync.Once
	initErr        error
	nopLogger      = zap.NewNop()
)

type responseData struct {
	status int
	size   int
}

type loggingResponseWriter struct {
	http.ResponseWriter
	responseData *responseData
}

func (r *loggingResponseWriter) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b)
	r.responseData.size += size
	return size, err
}

func (r *loggingResponseWriter) WriteHeader(statusCode int) {
	r.ResponseWriter.WriteHeader(statusCode)
	r.responseData.status = statusCode
}

// Init пытается сконфигурировать production-логгер; при ошибке тихо падаем на no-op,
// чтобы не сносить тесты. Safe для многократных вызовов.
func Init() error {
	once.Do(func() {
		l, err := zap.NewProduction()
		if err != nil {
			// fallback, чтобы не было паники в тех местах, где GetLogger() уже используется
			l = nopLogger
		}
		loggerInstance = l
		initErr = err
	})
	return initErr
}

// GetLogger всегда возвращает валидный *zap.Logger.
// Если Init() не вызывали — вернёт no-op и тесты не упадут.
func GetLogger() *zap.Logger {
	if loggerInstance == nil {
		return nopLogger
	}
	return loggerInstance
}

// HandlerLog — HTTP middleware логгирования запросов/ответов.
func HandlerLog(h http.HandlerFunc) http.HandlerFunc {
	logFn := func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// безопасно читаем body (до 1 МБ), но не мешаем обработке при ошибке
		var requestBody []byte
		if r.Body != nil {
			const limit = 1 << 20 // 1 MiB
			bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, limit))
			if err != nil {
				GetLogger().Warn("error reading request body", zap.Error(err))
				// продолжаем без тела
			} else {
				requestBody = bodyBytes
				r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			}
		}

		responseData := &responseData{
			status: http.StatusOK, // по умолчанию 200, если WriteHeader не вызывали
			size:   0,
		}
		lw := loggingResponseWriter{
			ResponseWriter: w,
			responseData:   responseData,
		}
		h.ServeHTTP(&lw, r)

		duration := time.Since(start)
		contentType := r.Header.Get("Content-Type")

		GetLogger().Info("HTTP request",
			zap.String("SHA256", r.Header.Get("HashSHA256")),
			zap.String("method", r.Method),
			zap.String("url", r.URL.String()),
			zap.String("content_type", contentType),
			zap.Int("req_body_size", len(requestBody)),
			zap.Int("status", responseData.status),
			zap.Int("resp_size", responseData.size),
			zap.Duration("duration", duration),
		)
	}
	return logFn
}
