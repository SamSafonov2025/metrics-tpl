package config

import (
	"flag"
	"fmt"
	"log"
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
	var configFile string

	// 0) Определяем путь к JSON конфигу (из флага или env)
	flag.StringVar(&configFile, "c", "", "Path to JSON configuration file")
	flag.StringVar(&configFile, "config", "", "Path to JSON configuration file")

	// 1) Объявляем флаги
	var (
		flagAddress        string
		flagPollInt        int
		flagReportInt      int
		flagKey            string
		flagCryptoKey      string
		flagRateLimit      int
	)

	flag.StringVar(&flagAddress, "a", "", "HTTP server endpoint address")
	flag.IntVar(&flagPollInt, "p", -1, "Poll interval in seconds")
	flag.IntVar(&flagReportInt, "r", -1, "Report interval in seconds")
	flag.StringVar(&flagKey, "k", "", "Key for hash calculation")
	flag.StringVar(&flagCryptoKey, "crypto-key", "", "Path to RSA public key file for encryption")
	flag.IntVar(&flagRateLimit, "l", -1, "Max concurrent outbound requests (rate limit)")

	flag.Parse()

	// Проверяем переменную окружения для конфигурационного файла
	if configFile == "" {
		if v, ok := os.LookupEnv("CONFIG"); ok {
			configFile = v
		}
	}

	// 2) Загружаем JSON конфигурацию (если указана)
	var jsonCfg *AgentJSONConfig
	if configFile != "" {
		var err error
		jsonCfg, err = LoadAgentJSONConfig(configFile)
		if err != nil {
			log.Printf("Warning: failed to load JSON config from %s: %v", configFile, err)
		}
	}

	// 3) Устанавливаем значения с приоритетом: флаги > env > JSON > defaults

	// Address
	cfg.ServerAddress = "localhost:8080" // default
	if jsonCfg != nil && jsonCfg.Address != "" {
		cfg.ServerAddress = jsonCfg.Address
	}
	if v, ok := os.LookupEnv("ADDRESS"); ok {
		cfg.ServerAddress = v
	}
	if flagAddress != "" {
		cfg.ServerAddress = flagAddress
	}

	// PollInterval
	pollSeconds := 2 // default
	if jsonCfg != nil && jsonCfg.PollInterval != "" {
		if d, err := time.ParseDuration(jsonCfg.PollInterval); err == nil {
			pollSeconds = int(d.Seconds())
		}
	}
	if v, ok := os.LookupEnv("POLL_INTERVAL"); ok {
		if n, err := strconv.Atoi(v); err == nil {
			pollSeconds = n
		}
	}
	if flagPollInt >= 0 {
		pollSeconds = flagPollInt
	}

	// ReportInterval
	reportSeconds := 10 // default
	if jsonCfg != nil && jsonCfg.ReportInterval != "" {
		if d, err := time.ParseDuration(jsonCfg.ReportInterval); err == nil {
			reportSeconds = int(d.Seconds())
		}
	}
	if v, ok := os.LookupEnv("REPORT_INTERVAL"); ok {
		if n, err := strconv.Atoi(v); err == nil {
			reportSeconds = n
		}
	}
	if flagReportInt >= 0 {
		reportSeconds = flagReportInt
	}

	// CryptoKeyPath (для RSA)
	cfg.CryptoKeyPath = "" // default
	if jsonCfg != nil && jsonCfg.CryptoKey != "" {
		cfg.CryptoKeyPath = jsonCfg.CryptoKey
	}
	if v, ok := os.LookupEnv("CRYPTO_KEY"); ok {
		cfg.CryptoKeyPath = v
	}
	if flagCryptoKey != "" {
		cfg.CryptoKeyPath = flagCryptoKey
	}

	// CryptoKey (для HMAC)
	cfg.CryptoKey = "" // default
	if v, ok := os.LookupEnv("KEY"); ok {
		cfg.CryptoKey = v
	}
	if flagKey != "" {
		cfg.CryptoKey = flagKey
	}

	// RateLimit
	cfg.RateLimit = 4 // default
	if v, ok := os.LookupEnv("RATE_LIMIT"); ok {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.RateLimit = n
		}
	}
	if flagRateLimit >= 0 {
		cfg.RateLimit = flagRateLimit
	}

	// Нормализация
	if cfg.RateLimit < 1 {
		cfg.RateLimit = 1
	}

	// 4) Производные поля
	cfg.PollInterval = time.Duration(pollSeconds) * time.Second
	cfg.ReportInterval = time.Duration(reportSeconds) * time.Second

	// Debug-выводы (сохраняем для совместимости)
	fmt.Printf("AGENT: key (%s) !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!", cfg.CryptoKey)
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
