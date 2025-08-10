package logger

import (
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

func HandlerLog(h http.HandlerFunc) http.HandlerFunc {
	logFn := func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

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

		GetLogger().Info("HTTP request",
			zap.String("method", r.Method),
			zap.String("url", r.URL.String()),
			zap.Int("status", responseData.status),
			zap.Int("size", responseData.size),
			zap.Duration("duration", duration),
		)
	}
	return logFn
}
