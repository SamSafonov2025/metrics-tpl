// Package rsadecryptor provides middleware for decrypting RSA-encrypted HTTP request bodies.
package rsadecryptor

import (
	"bytes"
	"crypto/rsa"
	"io"
	"net/http"

	"github.com/SamSafonov2025/metrics-tpl/internal/logger"
	"github.com/SamSafonov2025/metrics-tpl/internal/rsacrypto"
	"go.uber.org/zap"
)

// RSADecryptMiddleware creates a middleware that decrypts RSA-encrypted request bodies.
// If the request has the "X-Encrypted" header set to "true", it will decrypt the body
// using the provided private key before passing the request to the next handler.
func RSADecryptMiddleware(privateKey *rsa.PrivateKey) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if the request is encrypted
			if r.Header.Get("X-Encrypted") != "true" {
				// Not encrypted, pass through
				next.ServeHTTP(w, r)
				return
			}

			// If no private key is provided, return error
			if privateKey == nil {
				logger.GetLogger().Error("rsadecryptor: received encrypted request but no private key is configured")
				http.Error(w, "Server not configured to decrypt encrypted requests", http.StatusBadRequest)
				return
			}

			// Read the encrypted body
			encryptedBody, err := io.ReadAll(r.Body)
			if err != nil {
				logger.GetLogger().Error("rsadecryptor: failed to read encrypted body", zap.Error(err))
				http.Error(w, "Failed to read encrypted body", http.StatusBadRequest)
				return
			}
			r.Body.Close()

			// Decrypt the body
			decryptedBody, err := rsacrypto.DecryptChunked(encryptedBody, privateKey)
			if err != nil {
				logger.GetLogger().Error("rsadecryptor: failed to decrypt body", zap.Error(err))
				http.Error(w, "Failed to decrypt body", http.StatusBadRequest)
				return
			}

			logger.GetLogger().Info("rsadecryptor: decrypted request body",
				zap.Int("encrypted_bytes", len(encryptedBody)),
				zap.Int("decrypted_bytes", len(decryptedBody)))

			// Replace the request body with decrypted data
			r.Body = io.NopCloser(bytes.NewReader(decryptedBody))
			r.ContentLength = int64(len(decryptedBody))

			// Remove the encryption header since the body is now decrypted
			r.Header.Del("X-Encrypted")

			// Pass the request with decrypted body to the next handler
			next.ServeHTTP(w, r)
		})
	}
}
