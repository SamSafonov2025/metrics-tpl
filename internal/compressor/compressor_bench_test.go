package compressor

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// BenchmarkGzipMiddleware_Decompress измеряет производительность распаковки gzip
func BenchmarkGzipMiddleware_Decompress(b *testing.B) {
	// Подготовка сжатых данных разного размера
	sizes := []int{100, 1000, 10000}

	for _, size := range sizes {
		b.Run(formatSize(size), func(b *testing.B) {
			// Создаем тестовые данные
			data := strings.Repeat("test data ", size)

			// Сжимаем данные
			var buf bytes.Buffer
			gz := gzip.NewWriter(&buf)
			_, _ = gz.Write([]byte(data))
			_ = gz.Close()
			compressed := buf.Bytes()

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Читаем распакованное тело
				_, _ = io.ReadAll(r.Body)
				w.WriteHeader(http.StatusOK)
			})

			middleware := GzipMiddleware(handler)

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(compressed))
				req.Header.Set("Content-Encoding", "gzip")
				w := httptest.NewRecorder()

				middleware.ServeHTTP(w, req)
			}
		})
	}
}

// BenchmarkGzipMiddleware_Compress измеряет производительность сжатия gzip
func BenchmarkGzipMiddleware_Compress(b *testing.B) {
	sizes := []int{100, 1000, 10000}

	for _, size := range sizes {
		b.Run(formatSize(size), func(b *testing.B) {
			data := strings.Repeat("test data ", size)

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(data))
			})

			middleware := GzipMiddleware(handler)

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				req := httptest.NewRequest(http.MethodGet, "/", nil)
				req.Header.Set("Accept-Encoding", "gzip")
				w := httptest.NewRecorder()

				middleware.ServeHTTP(w, req)
			}
		})
	}
}

// BenchmarkGzipMiddleware_NoCompression измеряет overhead middleware без сжатия
func BenchmarkGzipMiddleware_NoCompression(b *testing.B) {
	data := strings.Repeat("test data ", 1000)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(data))
	})

	middleware := GzipMiddleware(handler)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		middleware.ServeHTTP(w, req)
	}
}

func formatSize(size int) string {
	if size < 1000 {
		return "size_" + intToString(size) + "B"
	}
	return "size_" + intToString(size/1000) + "KB"
}

func intToString(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf) - 1
	for n > 0 {
		buf[i] = byte('0' + n%10)
		n /= 10
		i--
	}
	return string(buf[i+1:])
}
