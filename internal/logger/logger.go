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

func Init() error {
	var err error
	once.Do(func() {
		loggerInstance, err = zap.NewProduction()
	})
	return err
}

func GetLogger() *zap.Logger {
	return loggerInstance
}

// ---- NEW: глобальный middleware-логгер ----
// Он логирует запросы даже если следующий middleware (например, HashValidation) вернёт 400
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// читаем тело (как есть после предыдущих middleware) и возвращаем в r.Body
		var raw []byte
		if r.Body != nil {
			b, err := io.ReadAll(r.Body)
			if err == nil {
				raw = b
				r.Body = io.NopCloser(bytes.NewReader(b))
			}
		}

		rd := &responseData{}
		lw := &loggingResponseWriter{ResponseWriter: w, responseData: rd}
		next.ServeHTTP(lw, r)

		// безопасно укоротим тело в лог (чтобы не засорять)
		const maxBody = 512
		bodyForLog := raw
		if len(bodyForLog) > maxBody {
			bodyForLog = bodyForLog[:maxBody]
		}

		GetLogger().Info("HTTP request",
			zap.String("method", r.Method),
			zap.String("url", r.URL.String()),
			zap.String("path", r.URL.Path),
			zap.String("host", r.Host),
			zap.String("remote", r.RemoteAddr),
			zap.String("userAgent", r.UserAgent()),
			zap.String("Content-Type", r.Header.Get("Content-Type")),
			zap.String("Content-Encoding", r.Header.Get("Content-Encoding")),
			zap.String("Accept-Encoding", r.Header.Get("Accept-Encoding")),
			zap.String("SHA256", r.Header.Get("HashSHA256")),
			zap.Int64("Content-Length", r.ContentLength),
			zap.Int("body_bytes", len(raw)),
			zap.String("body", string(bodyForLog)),
			zap.Int("status", rd.status),
			zap.Int("size", rd.size),
			zap.Duration("duration", time.Since(start)),
		)
	})
}

// Оставляем ваш обёрточный логгер — но он не нужен, если используете Middleware глобально.
// Можно убрать после проверки.
func HandlerLog(h http.HandlerFunc) http.HandlerFunc {
	logFn := func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		var requestBody []byte
		if r.Body != nil {
			bodyBytes, err := io.ReadAll(r.Body)
			if err != nil {
				return
			}
			requestBody = bodyBytes
			r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		responseData := &responseData{
			status: 0,
			size:   0,
		}
		lw := loggingResponseWriter{
			ResponseWriter: w,
			responseData:   responseData,
		}
		h.ServeHTTP(&lw, r)

		duration := time.Since(start)

		const maxBody = 512
		bodyForLog := requestBody
		if len(bodyForLog) > maxBody {
			bodyForLog = bodyForLog[:maxBody]
		}

		GetLogger().Info("HTTP request",
			zap.String("SHA256", r.Header.Get("HashSHA256")),
			zap.String("method", r.Method),
			zap.String("url", r.URL.String()),
			zap.String("path", r.URL.Path),
			zap.String("Content-Type", r.Header.Get("Content-Type")),
			zap.String("Content-Encoding", r.Header.Get("Content-Encoding")),
			zap.String("Accept-Encoding", r.Header.Get("Accept-Encoding")),
			zap.String("body", string(bodyForLog)),
			zap.Int("status", responseData.status),
			zap.Int("size", responseData.size),
			zap.Duration("duration", duration),
		)
	}
	return logFn
}
