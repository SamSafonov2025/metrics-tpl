// internal/audit/http_observer.go
package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// URLAuditObserver наблюдатель для отправки на удаленный сервер
type URLAuditObserver struct {
	url    string
	client *http.Client
}

// NewURLAuditObserver создает наблюдателя для отправки по HTTP
func NewURLAuditObserver(url string) *URLAuditObserver {
	return &URLAuditObserver{
		url: url,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// Notify отправляет событие на удаленный сервер
func (u *URLAuditObserver) Notify(event AuditEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal audit event: %w", err)
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, u.url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create audit request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := u.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send audit event: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("audit server returned status: %d", resp.StatusCode)
	}

	return nil
}

// Close закрывает HTTP клиент
func (u *URLAuditObserver) Close() error {
	u.client.CloseIdleConnections()
	return nil
}
