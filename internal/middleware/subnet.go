package middleware

import (
	"net"
	"net/http"
	"strings"
)

// TrustedSubnetMiddleware проверяет, что IP-адрес клиента входит в доверенную подсеть
func TrustedSubnetMiddleware(trustedSubnet string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Если trusted_subnet пустой, пропускаем проверку
			if trustedSubnet == "" {
				next.ServeHTTP(w, r)
				return
			}

			// Получаем IP-адрес из заголовка X-Real-IP
			clientIP := r.Header.Get("X-Real-IP")
			if clientIP == "" {
				// Если заголовок пустой, пробуем получить из RemoteAddr
				clientIP, _, _ = net.SplitHostPort(r.RemoteAddr)
			}

			// Проверяем, входит ли IP в доверенную подсеть
			if !isIPInSubnet(clientIP, trustedSubnet) {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// isIPInSubnet проверяет, входит ли IP-адрес в указанную подсеть CIDR
func isIPInSubnet(ipStr, cidr string) bool {
	// Парсим CIDR
	_, subnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return false
	}

	// Парсим IP-адрес
	ip := net.ParseIP(strings.TrimSpace(ipStr))
	if ip == nil {
		return false
	}

	// Проверяем, содержится ли IP в подсети
	return subnet.Contains(ip)
}
