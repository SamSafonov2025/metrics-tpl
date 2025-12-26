package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type AgentConfig struct {
	ServerAddress  string        `env:"ADDRESS" env-default:"localhost:8080"`
	PollInterval   time.Duration `env:"POLL_INTERVAL" env-default:"2s"`
	ReportInterval time.Duration `env:"REPORT_INTERVAL" env-default:"10s"`
	CryptoKey      string        `env:"KEY" env-default:""`
	CryptoKeyPath  string        `env:"CRYPTO_KEY" env-default:""`
	RateLimit      int           `env:"RATE_LIMIT" env-default:"4"`
}

func ParseAgentFlags() *AgentConfig {
	cfg := &AgentConfig{}

	// Сначала читаем переменные окружения и значения по умолчанию через cleanenv
	_ = cleanenv.ReadEnv(cfg)

	// Парсим флаг конфигурационного файла отдельно
	var configFile string
	flag.StringVar(&configFile, "c", "", "Path to JSON configuration file")
	flag.StringVar(&configFile, "config", "", "Path to JSON configuration file")

	// Временные переменные для флагов (чтобы отличить явно заданные от дефолтных)
	var (
		addrFlag      string
		pollFlag      time.Duration
		reportFlag    time.Duration
		keyFlag       string
		cryptoKeyFlag string
		rateLimitFlag int
	)

	// Парсим флаги во временные переменные
	flag.StringVar(&addrFlag, "a", "", "HTTP server endpoint address")
	flag.DurationVar(&pollFlag, "p", 0, "Poll interval")
	flag.DurationVar(&reportFlag, "r", 0, "Report interval")
	flag.StringVar(&keyFlag, "k", "", "Key for hash calculation")
	flag.StringVar(&cryptoKeyFlag, "crypto-key", "", "Path to RSA public key for encryption")
	flag.IntVar(&rateLimitFlag, "l", 0, "Max concurrent outbound requests (rate limit)")
	flag.Parse()

	// Загружаем конфигурацию из JSON файла, если указан
	if configFile != "" {
		if err := loadAgentJSONConfig(configFile, cfg); err != nil {
			// Логируем ошибку, но продолжаем с текущей конфигурацией
			_ = err
		}
	}

	// Применяем флаги поверх всего (наивысший приоритет), если они были явно указаны
	flag.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "a":
			cfg.ServerAddress = addrFlag
		case "p":
			cfg.PollInterval = pollFlag
		case "r":
			cfg.ReportInterval = reportFlag
		case "k":
			cfg.CryptoKey = keyFlag
		case "crypto-key":
			cfg.CryptoKeyPath = cryptoKeyFlag
		case "l":
			cfg.RateLimit = rateLimitFlag
		}
	})

	// Валидация
	if cfg.RateLimit < 1 {
		cfg.RateLimit = 1
	}

	// debug-выводы (оставил как в исходнике)
	fmt.Printf("AGENT: key (%s) !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!", cfg.CryptoKey)
	fmt.Printf("AGENT: cfg.CryptoKey (%s) !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!", cfg.CryptoKey)
	fmt.Printf("AGENT: rate_limit (%d)\n", cfg.RateLimit)

	return cfg
}

// JSONAgentConfig represents the structure of the JSON configuration file
type JSONAgentConfig struct {
	Address        string `json:"address"`
	ReportInterval string `json:"report_interval"`
	PollInterval   string `json:"poll_interval"`
	CryptoKey      string `json:"crypto_key"`
}

// loadAgentJSONConfig loads configuration from JSON file
func loadAgentJSONConfig(filename string, cfg *AgentConfig) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	var jsonCfg JSONAgentConfig
	if err := json.Unmarshal(data, &jsonCfg); err != nil {
		return err
	}

	// Применяем значения из JSON, если они не пустые
	if jsonCfg.Address != "" {
		cfg.ServerAddress = jsonCfg.Address
	}
	if jsonCfg.CryptoKey != "" {
		cfg.CryptoKeyPath = jsonCfg.CryptoKey
	}
	if jsonCfg.PollInterval != "" {
		if interval, err := time.ParseDuration(jsonCfg.PollInterval); err == nil {
			cfg.PollInterval = interval
		}
	}
	if jsonCfg.ReportInterval != "" {
		if interval, err := time.ParseDuration(jsonCfg.ReportInterval); err == nil {
			cfg.ReportInterval = interval
		}
	}

	return nil
}
