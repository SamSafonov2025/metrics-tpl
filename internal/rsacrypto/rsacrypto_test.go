package rsacrypto

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
)

// generateTestKeys создает пару ключей RSA для тестирования.
func generateTestKeys(t *testing.T) (*rsa.PrivateKey, *rsa.PublicKey) {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate private key: %v", err)
	}

	return privateKey, &privateKey.PublicKey
}

// saveKeyToFile сохраняет ключ в файл в формате PEM.
func savePrivateKeyToFile(t *testing.T, key *rsa.PrivateKey, path string) {
	t.Helper()

	keyBytes := x509.MarshalPKCS1PrivateKey(key)
	block := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: keyBytes,
	}

	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("failed to create private key file: %v", err)
	}
	defer file.Close()

	if err := pem.Encode(file, block); err != nil {
		t.Fatalf("failed to encode private key: %v", err)
	}
}

func savePublicKeyToFile(t *testing.T, key *rsa.PublicKey, path string) {
	t.Helper()

	keyBytes, err := x509.MarshalPKIXPublicKey(key)
	if err != nil {
		t.Fatalf("failed to marshal public key: %v", err)
	}

	block := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: keyBytes,
	}

	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("failed to create public key file: %v", err)
	}
	defer file.Close()

	if err := pem.Encode(file, block); err != nil {
		t.Fatalf("failed to encode public key: %v", err)
	}
}

func TestEncryptDecrypt(t *testing.T) {
	privateKey, publicKey := generateTestKeys(t)

	testCases := []struct {
		name string
		data []byte
	}{
		{"empty data", []byte{}},
		{"small data", []byte("Hello, World!")},
		{"medium data", []byte("Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.")},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Encrypt
			encrypted, err := Encrypt(tc.data, publicKey)
			if err != nil {
				t.Fatalf("failed to encrypt: %v", err)
			}

			// Decrypt
			decrypted, err := Decrypt(encrypted, privateKey)
			if err != nil {
				t.Fatalf("failed to decrypt: %v", err)
			}

			// Compare
			if string(decrypted) != string(tc.data) {
				t.Errorf("decrypted data does not match original.\nGot: %q\nWant: %q", decrypted, tc.data)
			}
		})
	}
}

func TestEncryptDecryptChunked(t *testing.T) {
	privateKey, publicKey := generateTestKeys(t)

	testCases := []struct {
		name string
		data []byte
	}{
		{"small data", []byte("Hello, World!")},
		{"exact chunk size", make([]byte, publicKey.Size()-11)},
		{"large data", make([]byte, 1024)}, // Больше одного блока
		{"very large data", make([]byte, 4096)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Fill with test data
			for i := range tc.data {
				tc.data[i] = byte(i % 256)
			}

			// Encrypt
			encrypted, err := EncryptChunked(tc.data, publicKey)
			if err != nil {
				t.Fatalf("failed to encrypt chunked: %v", err)
			}

			// Decrypt
			decrypted, err := DecryptChunked(encrypted, privateKey)
			if err != nil {
				t.Fatalf("failed to decrypt chunked: %v", err)
			}

			// Compare
			if len(decrypted) != len(tc.data) {
				t.Errorf("decrypted data length mismatch. Got: %d, Want: %d", len(decrypted), len(tc.data))
			}

			for i := range tc.data {
				if decrypted[i] != tc.data[i] {
					t.Errorf("byte at position %d does not match. Got: %d, Want: %d", i, decrypted[i], tc.data[i])
					break
				}
			}
		})
	}
}

func TestLoadKeys(t *testing.T) {
	tmpDir := t.TempDir()
	privateKeyPath := filepath.Join(tmpDir, "private.pem")
	publicKeyPath := filepath.Join(tmpDir, "public.pem")

	// Generate and save keys
	privateKey, publicKey := generateTestKeys(t)
	savePrivateKeyToFile(t, privateKey, privateKeyPath)
	savePublicKeyToFile(t, publicKey, publicKeyPath)

	// Load private key
	loadedPrivateKey, err := LoadPrivateKey(privateKeyPath)
	if err != nil {
		t.Fatalf("failed to load private key: %v", err)
	}

	// Load public key
	loadedPublicKey, err := LoadPublicKey(publicKeyPath)
	if err != nil {
		t.Fatalf("failed to load public key: %v", err)
	}

	// Test encryption/decryption with loaded keys
	testData := []byte("Test data for loaded keys")
	encrypted, err := Encrypt(testData, loadedPublicKey)
	if err != nil {
		t.Fatalf("failed to encrypt with loaded public key: %v", err)
	}

	decrypted, err := Decrypt(encrypted, loadedPrivateKey)
	if err != nil {
		t.Fatalf("failed to decrypt with loaded private key: %v", err)
	}

	if string(decrypted) != string(testData) {
		t.Errorf("decrypted data does not match original.\nGot: %q\nWant: %q", decrypted, testData)
	}
}

func TestLoadKeysErrors(t *testing.T) {
	t.Run("non-existent file", func(t *testing.T) {
		_, err := LoadPrivateKey("/non/existent/path")
		if err == nil {
			t.Error("expected error for non-existent file")
		}

		_, err = LoadPublicKey("/non/existent/path")
		if err == nil {
			t.Error("expected error for non-existent file")
		}
	})

	t.Run("invalid PEM format", func(t *testing.T) {
		tmpDir := t.TempDir()
		invalidPath := filepath.Join(tmpDir, "invalid.pem")

		if err := os.WriteFile(invalidPath, []byte("invalid data"), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		_, err := LoadPrivateKey(invalidPath)
		if err == nil {
			t.Error("expected error for invalid PEM format")
		}

		_, err = LoadPublicKey(invalidPath)
		if err == nil {
			t.Error("expected error for invalid PEM format")
		}
	})
}
