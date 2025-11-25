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
	if v, ok := os.LookupEnv("ADDRESS"); ok {
		cfg.ServerAddress = v
	}
	if v, ok := os.LookupEnv("STORE_INTERVAL"); ok {
		if n, err := strconv.Atoi(v); err == nil {
			storeSeconds = n
		}
	}
	if v, ok := os.LookupEnv("FILE_STORAGE_PATH"); ok {
		cfg.FileStoragePath = v
	}
	if v, ok := os.LookupEnv("RESTORE"); ok {
		if b, err := strconv.ParseBool(v); err == nil {
			cfg.Restore = b
		}
	}
	if v, ok := os.LookupEnv("DATABASE_DSN"); ok {
		cfg.Database = v
	}
	if v, ok := os.LookupEnv("KEY"); ok {
		cfg.CryptoKey = v
	}
	if v, ok := os.LookupEnv("AUDIT_FILE"); ok {
		cfg.AuditFile = v
	}
	if v, ok := os.LookupEnv("AUDIT_URL"); ok {
		cfg.AuditURL = v
	}

	// 3) Производные поля
	cfg.StoreInterval = time.Duration(storeSeconds) * time.Second
	return cfg
}
