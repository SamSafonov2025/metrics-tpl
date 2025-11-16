package config

import (
	"flag"
	"os"
	"strconv"
	"time"
)

type ServerConfig struct {
	ServerAddress   string
	StoreInterval   time.Duration
	FileStoragePath string
	Restore         bool
	Database        string
	CryptoKey       string
	AuditFile       string // путь к файлу для логов аудита
	AuditURL        string // URL для отправки логов аудита
}

func ParseServerFlags() *ServerConfig {
	cfg := &ServerConfig{}
	var storeSeconds int

	// 1) Значения по умолчанию для флагов (НЕ из env)
	flag.StringVar(&cfg.ServerAddress, "a", "localhost:8080", "HTTP server endpoint address")
	flag.IntVar(&storeSeconds, "i", 300, "Store interval in seconds (0 = sync mode)")
	flag.StringVar(&cfg.FileStoragePath, "f", "/tmp/metrics-db.json", "File storage path")
	flag.BoolVar(&cfg.Restore, "r", false, "Restore metrics from file")
	flag.StringVar(&cfg.Database, "d",
		"postgresql://postgres:arzamas17@localhost:5432/yandex_go?sslmode=disable&search_path=public",
		"Database connection string",
	)
	flag.StringVar(&cfg.CryptoKey, "k", "", "Key for hash calculation")
	flag.StringVar(&cfg.AuditFile, "audit-file", "", "Audit log file path")
	flag.StringVar(&cfg.AuditURL, "audit-url", "", "Audit log URL endpoint")

	flag.Parse()

	// 2) ENV перекрывает значения флагов, если задан
	if v := os.Getenv("ADDRESS"); v != "" {
		cfg.ServerAddress = v
	}
	if v := os.Getenv("STORE_INTERVAL"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			storeSeconds = n
		}
	}
	if v := os.Getenv("FILE_STORAGE_PATH"); v != "" {
		cfg.FileStoragePath = v
	}
	if v := os.Getenv("RESTORE"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			cfg.Restore = b
		}
	}
	if v := os.Getenv("DATABASE_DSN"); v != "" {
		cfg.Database = v
	}
	if v := os.Getenv("KEY"); v != "" {
		cfg.CryptoKey = v
	}
	if v := os.Getenv("AUDIT_FILE"); v != "" {
		cfg.AuditFile = v
	}
	if v := os.Getenv("AUDIT_URL"); v != "" {
		cfg.AuditURL = v
	}

	// 3) Производные поля
	cfg.StoreInterval = time.Duration(storeSeconds) * time.Second
	return cfg
}
