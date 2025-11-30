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
	// Маршализация JSON вне критической секции (CPU-интенсивная операция)
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal audit event: %w", err)
	}

	// Добавляем перевод строки вне критической секции
	data = append(data, '\n')

	// Критическая секция: только запись в файл и синхронизация с диском
	f.mu.Lock()
	defer f.mu.Unlock()

	if _, err := f.file.Write(data); err != nil {
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
		err := f.file.Close()
		f.file = nil // Устанавливаем в nil, чтобы избежать повторного закрытия
		return err
	}
	return nil
}
