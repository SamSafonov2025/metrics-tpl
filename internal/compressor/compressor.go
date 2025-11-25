package compressor

import (
	"compress/gzip"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
)

var (
	// gzipWriterPool переиспользует gzip writers для компрессии
	gzipWriterPool = sync.Pool{
		New: func() interface{} {
			return gzip.NewWriter(io.Discard)
		},
	}
)

func GzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Request URL: %s, Content-Encoding: %s, Accept-Encoding: %s", r.URL, r.Header.Get("Content-Encoding"), r.Header.Get("Accept-Encoding"))

		if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
			// Создаем новый reader для каждого запроса
			// gzip.Reader содержит внутреннее состояние, поэтому пулинг не эффективен
			gzipReader, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(w, "Unable to read gzip data", http.StatusBadRequest)
				return
			}
			defer func() {
				if err := gzipReader.Close(); err != nil {
					http.Error(w, "Unable to close gzip data", http.StatusBadRequest)
				}
			}()

			r.Body = io.NopCloser(gzipReader)
		}

		if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			// Получаем writer из пула
			gz := gzipWriterPool.Get().(*gzip.Writer)
			defer gzipWriterPool.Put(gz)

			// Переинициализируем writer для нового response
			gz.Reset(w)
			defer func() {
				if err := gz.Close(); err != nil {
					http.Error(w, "Unable to close gzip data", http.StatusBadRequest)
				}
			}()

			w.Header().Set("Content-Encoding", "gzip")
			w.Header().Set("Vary", "Accept-Encoding")

			gzw := &gzipResponseWriter{ResponseWriter: w, Writer: gz}
			next.ServeHTTP(gzw, r)
		} else {
			next.ServeHTTP(w, r)
		}
	})
}

type gzipResponseWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

func (g *gzipResponseWriter) Write(b []byte) (int, error) {
	return g.Writer.Write(b)
}
