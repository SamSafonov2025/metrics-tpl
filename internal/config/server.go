package config

import (
	"encoding/json"
	"flag"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type ServerConfig struct {
	ServerAddress   string        `env:"ADDRESS" env-default:"localhost:8080"`
	StoreInterval   time.Duration `env:"STORE_INTERVAL" env-default:"300s"`
	FileStoragePath string        `env:"FILE_STORAGE_PATH" env-default:"/tmp/metrics-db.json"`
	Restore         bool          `env:"RESTORE" env-default:"false"`
	Database        string        `env:"DATABASE_DSN" env-default:"postgresql://postgres:arzamas17@localhost:5432/yandex_go?sslmode=disable&search_path=public"`
	CryptoKey       string        `env:"KEY" env-default:""`
	CryptoKeyPath   string        `env:"CRYPTO_KEY" env-default:""`
	AuditFile       string        `env:"AUDIT_FILE" env-default:""`
	AuditURL        string        `env:"AUDIT_URL" env-default:""`
}

func ParseServerFlags() *ServerConfig {
	cfg := &ServerConfig{}

	// Сначала читаем переменные окружения и значения по умолчанию через cleanenv
	_ = cleanenv.ReadEnv(cfg)

	// Парсим флаг конфигурационного файла отдельно
	var configFile string
	flag.StringVar(&configFile, "c", "", "Path to JSON configuration file")
	flag.StringVar(&configFile, "config", "", "Path to JSON configuration file")

	// Временные переменные для флагов (чтобы отличить явно заданные от дефолтных)
	var (
		addrFlag      string
		intervalFlag  time.Duration
		fileFlag      string
		restoreFlag   bool
		dbFlag        string
		keyFlag       string
		cryptoKeyFlag string
		auditFileFlag string
		auditURLFlag  string
	)

	// Парсим флаги во временные переменные
	flag.StringVar(&addrFlag, "a", "", "HTTP server endpoint address")
	flag.DurationVar(&intervalFlag, "i", 0, "Store interval (0 = sync mode)")
	flag.StringVar(&fileFlag, "f", "", "File storage path")
	flag.BoolVar(&restoreFlag, "r", false, "Restore metrics from file")
	flag.StringVar(&dbFlag, "d", "", "Database connection string")
	flag.StringVar(&keyFlag, "k", "", "Key for hash calculation")
	flag.StringVar(&cryptoKeyFlag, "crypto-key", "", "Path to RSA private key for decryption")
	flag.StringVar(&auditFileFlag, "audit-file", "", "Audit log file path")
	flag.StringVar(&auditURLFlag, "audit-url", "", "Audit log URL endpoint")
	flag.Parse()

	// Загружаем конфигурацию из JSON файла, если указан
	if configFile != "" {
		if err := loadJSONConfig(configFile, cfg); err != nil {
			// Логируем ошибку, но продолжаем с текущей конфигурацией
			_ = err
		}
	}

	// Применяем флаги поверх всего (наивысший приоритет), если они были явно указаны
	flag.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "a":
			cfg.ServerAddress = addrFlag
		case "i":
			cfg.StoreInterval = intervalFlag
		case "f":
			cfg.FileStoragePath = fileFlag
		case "r":
			cfg.Restore = restoreFlag
		case "d":
			cfg.Database = dbFlag
		case "k":
			cfg.CryptoKey = keyFlag
		case "crypto-key":
			cfg.CryptoKeyPath = cryptoKeyFlag
		case "audit-file":
			cfg.AuditFile = auditFileFlag
		case "audit-url":
			cfg.AuditURL = auditURLFlag
		}
	})

	return cfg
}

// JSONServerConfig represents the structure of the JSON configuration file
type JSONServerConfig struct {
	Address       string `json:"address"`
	Restore       bool   `json:"restore"`
	StoreInterval string `json:"store_interval"`
	StoreFile     string `json:"store_file"`
	DatabaseDSN   string `json:"database_dsn"`
	CryptoKey     string `json:"crypto_key"`
}

// loadJSONConfig loads configuration from JSON file
func loadJSONConfig(filename string, cfg *ServerConfig) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	var jsonCfg JSONServerConfig
	if err := json.Unmarshal(data, &jsonCfg); err != nil {
		return err
	}

	// Применяем значения из JSON, если они не пустые
	if jsonCfg.Address != "" {
		cfg.ServerAddress = jsonCfg.Address
	}
	if jsonCfg.StoreFile != "" {
		cfg.FileStoragePath = jsonCfg.StoreFile
	}
	if jsonCfg.DatabaseDSN != "" {
		cfg.Database = jsonCfg.DatabaseDSN
	}
	if jsonCfg.CryptoKey != "" {
		cfg.CryptoKeyPath = jsonCfg.CryptoKey
	}
	if jsonCfg.StoreInterval != "" {
		if interval, err := time.ParseDuration(jsonCfg.StoreInterval); err == nil {
			cfg.StoreInterval = interval
		}
	}
	// Restore - всегда применяем из JSON
	cfg.Restore = jsonCfg.Restore

	return nil
}
