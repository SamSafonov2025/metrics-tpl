package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func testMiddleware(t *testing.T, subnet, ip string, wantCode int) {
	t.Helper()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := TrustedSubnetMiddleware(subnet)
	req := httptest.NewRequest(http.MethodPost, "/update", nil)
	if ip != "" {
		req.Header.Set("X-Real-IP", ip)
	}
	rec := httptest.NewRecorder()

	middleware(handler).ServeHTTP(rec, req)

	if rec.Code != wantCode {
		t.Errorf("Expected %d, got %d", wantCode, rec.Code)
	}
}

func TestTrustedSubnetMiddleware(t *testing.T) {
	tests := []struct {
		name   string
		subnet string
		ip     string
		want   int
	}{
		{"Empty subnet allows all", "", "192.168.1.100", http.StatusOK},
		{"IP in subnet", "192.168.1.0/24", "192.168.1.50", http.StatusOK},
		{"IP not in subnet", "192.168.1.0/24", "10.0.0.1", http.StatusForbidden},
		{"Localhost in /8", "127.0.0.0/8", "127.0.0.1", http.StatusOK},
		{"Invalid CIDR blocks all", "invalid", "192.168.1.1", http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testMiddleware(t, tt.subnet, tt.ip, tt.want)
		})
	}
}

func TestTrustedSubnetMiddleware_RemoteAddrFallback(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := TrustedSubnetMiddleware("192.168.1.0/24")
	req := httptest.NewRequest(http.MethodPost, "/update", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	rec := httptest.NewRecorder()

	middleware(handler).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected 200 with RemoteAddr fallback, got %d", rec.Code)
	}
}

func TestIsIPInSubnet(t *testing.T) {
	tests := []struct {
		ip   string
		cidr string
		want bool
	}{
		{"192.168.1.50", "192.168.1.0/24", true},
		{"192.168.2.50", "192.168.1.0/24", false},
		{"127.0.0.1", "127.0.0.0/8", true},
		{"10.5.5.10", "10.0.0.0/8", true},
		{"invalid-ip", "192.168.1.0/24", false},
		{"192.168.1.1", "invalid-cidr", false},
	}

	for _, tt := range tests {
		got := isIPInSubnet(tt.ip, tt.cidr)
		if got != tt.want {
			t.Errorf("isIPInSubnet(%s, %s) = %v, want %v", tt.ip, tt.cidr, got, tt.want)
		}
	}
}
