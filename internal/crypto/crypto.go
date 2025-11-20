package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt" // <— добавлено для форматирования сообщения об ошибке
	"io"
	"net/http"
	"strings"

	"go.uber.org/zap"

	"github.com/SamSafonov2025/metrics-tpl/internal/logger"
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

		logger.GetLogger().Info("HMAC: CryptoKey !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!", zapString("cryptoKey: ", c.Key))

		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			// логирование ошибки чтения тела
			logger.GetLogger().Warn("HMAC: unable to read request body", zapError(err))
			http.Error(w, "Unable to read request body", http.StatusInternalServerError)
			return
		}

		r.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))
		expectedHash := GenerateHash(bodyBytes, c.Key)

		receivedHash := r.Header.Get("HashSHA256")
		if receivedHash != expectedHash {
			// логирование несовпадения хэша
			logger.GetLogger().Warn(
				"HMAC: invalid hash",
				zapError(fmt.Errorf("received=%s expected=%s", receivedHash, expectedHash)),
			)
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

// маленьк
// ие помощники, чтобы не тащить zap в каждое место
func zapError(err error) zap.Field    { return zap.Error(err) }
func zapString(k, v string) zap.Field { return zap.String(k, v) }
