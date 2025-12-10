package config

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"
)

type AgentConfig struct {
	ServerAddress  string
	PollInterval   time.Duration
	ReportInterval time.Duration
	CryptoKey      string // HMAC signing key
	CryptoKeyPath  string // Path to RSA public key file for encryption
	RateLimit      int
}

func ParseAgentFlags() *AgentConfig {
	cfg := &AgentConfig{}
	var pollSeconds, reportSeconds int

	// env defaults
	addr := getEnv("ADDRESS", "localhost:8080")
	poll := atoiEnv("POLL_INTERVAL", 2)
	report := atoiEnv("REPORT_INTERVAL", 10)
	key := getEnv("KEY", "")
	cryptoKeyPath := getEnv("CRYPTO_KEY", "")
	rate := atoiEnv("RATE_LIMIT", 4)

	// flags (флаг имеет приоритет над env)
	flag.StringVar(&cfg.ServerAddress, "a", addr, "HTTP server endpoint address")
	flag.IntVar(&pollSeconds, "p", poll, "Poll interval in seconds")
	flag.IntVar(&reportSeconds, "r", report, "Report interval in seconds")
	flag.StringVar(&cfg.CryptoKey, "k", key, "Key for hash calculation")
	flag.StringVar(&cfg.CryptoKeyPath, "crypto-key", cryptoKeyPath, "Path to RSA public key file for encryption")
	flag.IntVar(&cfg.RateLimit, "l", rate, "Max concurrent outbound requests (rate limit)")
	flag.Parse()

	// нормализация и перевод в duration
	if cfg.RateLimit < 1 {
		cfg.RateLimit = 1
	}
	cfg.PollInterval = time.Duration(pollSeconds) * time.Second
	cfg.ReportInterval = time.Duration(reportSeconds) * time.Second

	// debug-выводы (оставил как в исходнике)
	fmt.Printf("AGENT: key (%s) !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!", key)
	fmt.Printf("AGENT: cfg.CryptoKey (%s) !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!", cfg.CryptoKey)
	fmt.Printf("AGENT: rate_limit (%d)\n", cfg.RateLimit)

	return cfg
}

func atoiEnv(k string, def int) int {
	if v, ok := os.LookupEnv(k); ok {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getEnv(k, def string) string {
	if v, ok := os.LookupEnv(k); ok {
		return v
	}
	return def
}
