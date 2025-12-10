// Package rsadecryptor provides middleware for decrypting RSA-encrypted HTTP request bodies.
package rsadecryptor

import (
	"bytes"
	"crypto/rsa"
	"io"
	"log"
	"net/http"

	"github.com/SamSafonov2025/metrics-tpl/internal/rsacrypto"
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
				log.Println("rsadecryptor: received encrypted request but no private key is configured")
				http.Error(w, "Server not configured to decrypt encrypted requests", http.StatusBadRequest)
				return
			}

			// Read the encrypted body
			encryptedBody, err := io.ReadAll(r.Body)
			if err != nil {
				log.Printf("rsadecryptor: failed to read encrypted body: %v", err)
				http.Error(w, "Failed to read encrypted body", http.StatusBadRequest)
				return
			}
			r.Body.Close()

			// Decrypt the body
			decryptedBody, err := rsacrypto.DecryptChunked(encryptedBody, privateKey)
			if err != nil {
				log.Printf("rsadecryptor: failed to decrypt body: %v", err)
				http.Error(w, "Failed to decrypt body", http.StatusBadRequest)
				return
			}

			log.Printf("rsadecryptor: decrypted request body: encrypted=%dB -> decrypted=%dB", len(encryptedBody), len(decryptedBody))

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
