package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"strings"
)

func GenerateHash(data []byte, key string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

type Crypto struct {
	Key string
}

func (c *Crypto) HashValidationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if c.Key == "" || r.Header.Get("HashSHA256") == "" {
			next.ServeHTTP(w, r)
			return
		}

		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Unable to read request body", http.StatusInternalServerError)
			return
		}

		r.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))
		expectedHash := GenerateHash(bodyBytes, c.Key)

		receivedHash := r.Header.Get("HashSHA256")
		if receivedHash != expectedHash {
			http.Error(w, "Invalid hash", http.StatusBadRequest)
			return
		}

		responseWriter := &responseHashWriter{ResponseWriter: w, key: c.Key}

		next.ServeHTTP(responseWriter, r)
	})
}

type responseHashWriter struct {
	http.ResponseWriter
	key  string
	body []byte
}

func (rw *responseHashWriter) Write(b []byte) (int, error) {
	rw.body = append(rw.body, b...)
	return rw.ResponseWriter.Write(b)
}

func (rw *responseHashWriter) WriteHeader(statusCode int) {
	if rw.key != "" && len(rw.body) > 0 {
		h := hmac.New(sha256.New, []byte(rw.key))
		h.Write(rw.body)
		hash := hex.EncodeToString(h.Sum(nil))
		rw.Header().Set("HashSHA256", hash)
	}
	rw.ResponseWriter.WriteHeader(statusCode)
}
