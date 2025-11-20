// internal/audit/file_observer.go
package audit

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

// FileAuditObserver наблюдатель для записи в файл
type FileAuditObserver struct {
	mu       sync.Mutex
	filePath string
	file     *os.File
}

// NewFileAuditObserver создает наблюдателя для записи в файл
func NewFileAuditObserver(filePath string) (*FileAuditObserver, error) {
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open audit file: %w", err)
	}
	return &FileAuditObserver{
		filePath: filePath,
		file:     file,
	}, nil
}

// Notify записывает событие в файл
func (f *FileAuditObserver) Notify(event AuditEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal audit event: %w", err)
	}

	// Добавляем событие на новой строке
	if _, err := f.file.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write audit event to file: %w", err)
	}

	// Синхронизируем запись с диском
	if err := f.file.Sync(); err != nil {
		return fmt.Errorf("failed to sync audit file: %w", err)
	}

	return nil
}

// Close закрывает файл
func (f *FileAuditObserver) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.file != nil {
		return f.file.Close()
	}
	return nil
}
