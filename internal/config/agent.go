package config

import (
	"flag"
	"os"
	"strconv"
	"time"
)

type AgentConfig struct {
	ServerAddress  string
	PollInterval   time.Duration
	ReportInterval time.Duration
	CryptoKey      string
}

func ParseAgentFlags() *AgentConfig {
	cfg := &AgentConfig{}
	var pollSeconds, reportSeconds int

	// env
	addr := getEnv("ADDRESS", "localhost:8080")
	poll := atoiEnv("POLL_INTERVAL", 2)
	report := atoiEnv("REPORT_INTERVAL", 10)
	key := getEnv("KEY", "")

	flag.StringVar(&cfg.ServerAddress, "a", addr, "HTTP server endpoint address")
	flag.IntVar(&pollSeconds, "p", poll, "Poll interval in seconds")
	flag.IntVar(&reportSeconds, "r", report, "Report interval in seconds")
	flag.StringVar(&cfg.CryptoKey, "k", key, "Key for hash calculation")
	flag.Parse()

	cfg.PollInterval = time.Duration(pollSeconds) * time.Second
	cfg.ReportInterval = time.Duration(reportSeconds) * time.Second

	//if cfg.CryptoKey != "" {
	//	cfg.CryptoKey = "123"
	//}

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
