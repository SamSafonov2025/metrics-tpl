package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/SamSafonov2025/metrics-tpl/internal/audit"
	"github.com/SamSafonov2025/metrics-tpl/internal/config"
	"github.com/SamSafonov2025/metrics-tpl/internal/dto"
	"github.com/SamSafonov2025/metrics-tpl/internal/router"
	"github.com/SamSafonov2025/metrics-tpl/internal/service"
	"github.com/SamSafonov2025/metrics-tpl/internal/storage"
	"github.com/go-chi/chi/v5"
)

// Вспомогательная функция для создания роутера с нужной подсетью
func setupRouter(trustedSubnet string) *chi.Mux {
	cfg := &config.ServerConfig{
		ServerAddress:   "localhost:8080",
		StoreInterval:   300 * time.Second,
		FileStoragePath: "/tmp/test-metrics.json",
		Restore:         false,
		TrustedSubnet:   trustedSubnet,
	}

	store := storage.NewStorage(cfg)
	svc := service.NewMetricsService(store, cfg.StoreInterval, func(ctx context.Context) error { return nil })
	auditPublisher := audit.NewAuditPublisher()

	return router.New(svc, "", nil, auditPublisher, cfg.TrustedSubnet)
}

// Вспомогательная функция для создания тестовой метрики
func createTestMetric() (dto.Metrics, []byte) {
	value := 42.5
	metric := dto.Metrics{ID: "TestMetric", MType: "gauge", Value: &value}
	body, _ := json.Marshal(metric)
	return metric, body
}

func TestTrustedSubnet_AllowedIP(t *testing.T) {
	r := setupRouter("192.168.1.0/24")
	_, body := createTestMetric()

	req := httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Real-IP", "192.168.1.100")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", rec.Code)
	}
}

func TestTrustedSubnet_BlockedIP(t *testing.T) {
	r := setupRouter("192.168.1.0/24")
	_, body := createTestMetric()

	req := httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Real-IP", "10.0.0.1")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("Expected 403, got %d", rec.Code)
	}
}

func TestTrustedSubnet_EmptySubnet(t *testing.T) {
	r := setupRouter("")
	_, body := createTestMetric()

	req := httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Real-IP", "203.0.113.1")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected 200 when trusted_subnet is empty, got %d", rec.Code)
	}
}

func TestTrustedSubnet_Localhost(t *testing.T) {
	r := setupRouter("127.0.0.0/8")
	_, body := createTestMetric()

	req := httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Real-IP", "127.0.0.1")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected 200 for localhost, got %d", rec.Code)
	}
}

func TestTrustedSubnet_BatchUpdate(t *testing.T) {
	r := setupRouter("10.0.0.0/8")

	value := 42.5
	delta := int64(10)
	metrics := []dto.Metrics{
		{ID: "M1", MType: "gauge", Value: &value},
		{ID: "M2", MType: "counter", Delta: &delta},
	}
	body, _ := json.Marshal(metrics)

	tests := []struct {
		ip   string
		want int
	}{
		{"10.1.2.3", http.StatusOK},
		{"192.168.1.1", http.StatusForbidden},
	}

	for _, tt := range tests {
		req := httptest.NewRequest(http.MethodPost, "/updates", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Real-IP", tt.ip)
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req)

		if rec.Code != tt.want {
			t.Errorf("IP %s: expected %d, got %d", tt.ip, tt.want, rec.Code)
		}
	}
}
