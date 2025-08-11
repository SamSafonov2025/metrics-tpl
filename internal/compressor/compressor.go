package compressor

import (
	"bytes"
	"compress/gzip"
	"net/http"
	"strings"
)

type responseRecorder struct {
	http.ResponseWriter
	buf         *bytes.Buffer
	statusCode  int
	header      http.Header
	wroteHeader bool
}

func (r *responseRecorder) Header() http.Header {
	return r.header
}

func (r *responseRecorder) WriteHeader(statusCode int) {
	if !r.wroteHeader {
		r.statusCode = statusCode
		r.wroteHeader = true
	}
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}
	return r.buf.Write(b)
}

func GzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
			gz, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(w, "Unable to read gzip data", http.StatusBadRequest)
				return
			}
			defer gz.Close()
			r.Body = gz
		}

		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		recorder := &responseRecorder{
			ResponseWriter: w,
			buf:            bytes.NewBuffer(nil),
			statusCode:     http.StatusOK,
			header:         make(http.Header),
		}

		next.ServeHTTP(recorder, r)

		contentType := recorder.Header().Get("Content-Type")

		if contentType == "application/json" || contentType == "text/html" {
			var compressedBuf bytes.Buffer
			gz := gzip.NewWriter(&compressedBuf)
			if _, err := gz.Write(recorder.buf.Bytes()); err != nil {
				http.Error(w, "Compression error", http.StatusInternalServerError)
				return
			}
			if err := gz.Close(); err != nil {
				http.Error(w, "Compression error", http.StatusInternalServerError)
				return
			}

			for k, v := range recorder.Header() {
				w.Header()[k] = v
			}
			w.Header().Set("Content-Encoding", "gzip")
			w.Header().Set("Vary", "Accept-Encoding")
			w.Header().Del("Content-Length")
			w.WriteHeader(recorder.statusCode)
			if _, err := w.Write(compressedBuf.Bytes()); err != nil {
				http.Error(w, "Write failed", http.StatusInternalServerError)
			}
		} else {
			for k, v := range recorder.Header() {
				w.Header()[k] = v
			}
			w.WriteHeader(recorder.statusCode)
			if _, err := w.Write(recorder.buf.Bytes()); err != nil {
				http.Error(w, "Write failed", http.StatusInternalServerError)
			}
		}
	})
}
