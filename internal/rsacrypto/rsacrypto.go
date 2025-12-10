// Package rsacrypto provides RSA encryption and decryption functionality for secure communication.
package rsacrypto

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
)

// LoadPublicKey загружает публичный ключ RSA из файла в формате PEM.
func LoadPublicKey(path string) (*rsa.PublicKey, error) {
	keyBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read public key file: %w", err)
	}

	block, _ := pem.Decode(keyBytes)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block from public key")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("key is not RSA public key")
	}

	return rsaPub, nil
}

// LoadPrivateKey загружает приватный ключ RSA из файла в формате PEM.
func LoadPrivateKey(path string) (*rsa.PrivateKey, error) {
	keyBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key file: %w", err)
	}

	block, _ := pem.Decode(keyBytes)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block from private key")
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	return privateKey, nil
}

// Encrypt шифрует данные с использованием публичного ключа RSA.
// Возвращает зашифрованные данные или ошибку.
func Encrypt(data []byte, publicKey *rsa.PublicKey) ([]byte, error) {
	// Используем PKCS1v15 для совместимости
	encrypted, err := rsa.EncryptPKCS1v15(rand.Reader, publicKey, data)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt data: %w", err)
	}
	return encrypted, nil
}

// Decrypt расшифровывает данные с использованием приватного ключа RSA.
// Возвращает расшифрованные данные или ошибку.
func Decrypt(encrypted []byte, privateKey *rsa.PrivateKey) ([]byte, error) {
	decrypted, err := rsa.DecryptPKCS1v15(rand.Reader, privateKey, encrypted)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt data: %w", err)
	}
	return decrypted, nil
}

// EncryptChunked шифрует большие данные по частям.
// RSA может шифровать только данные размером меньше размера ключа.
// Эта функция разбивает данные на части и шифрует каждую часть отдельно.
func EncryptChunked(data []byte, publicKey *rsa.PublicKey) ([]byte, error) {
	// Максимальный размер данных для шифрования одним блоком
	// Для PKCS1v15: keySize - 11 байт
	chunkSize := publicKey.Size() - 11

	if len(data) <= chunkSize {
		// Данные помещаются в один блок
		return Encrypt(data, publicKey)
	}

	// Разбиваем данные на части
	var result []byte
	for i := 0; i < len(data); i += chunkSize {
		end := i + chunkSize
		if end > len(data) {
			end = len(data)
		}

		chunk := data[i:end]
		encrypted, err := Encrypt(chunk, publicKey)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt chunk at position %d: %w", i, err)
		}

		// Добавляем размер зашифрованного блока (2 байта) и сам блок
		blockSize := uint16(len(encrypted))
		result = append(result, byte(blockSize>>8), byte(blockSize))
		result = append(result, encrypted...)
	}

	return result, nil
}

// DecryptChunked расшифровывает данные, зашифрованные с помощью EncryptChunked.
func DecryptChunked(encrypted []byte, privateKey *rsa.PrivateKey) ([]byte, error) {
	keySize := privateKey.Size()

	if len(encrypted) <= keySize {
		// Данные в одном блоке
		return Decrypt(encrypted, privateKey)
	}

	// Расшифровываем по частям
	var result []byte
	pos := 0

	for pos < len(encrypted) {
		if pos+2 > len(encrypted) {
			return nil, fmt.Errorf("invalid encrypted data: incomplete block size header at position %d", pos)
		}

		// Читаем размер блока
		blockSize := int(uint16(encrypted[pos])<<8 | uint16(encrypted[pos+1]))
		pos += 2

		if pos+blockSize > len(encrypted) {
			return nil, fmt.Errorf("invalid encrypted data: incomplete block at position %d", pos)
		}

		// Расшифровываем блок
		chunk := encrypted[pos : pos+blockSize]
		decrypted, err := Decrypt(chunk, privateKey)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt chunk at position %d: %w", pos, err)
		}

		result = append(result, decrypted...)
		pos += blockSize
	}

	return result, nil
}
