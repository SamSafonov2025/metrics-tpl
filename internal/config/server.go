package config

import (
	"flag"
	"log"
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
	CryptoKey       string // HMAC signing key
	CryptoKeyPath   string // Path to RSA private key file for decryption
	AuditFile       string // путь к файлу для логов аудита
	AuditURL        string // URL для отправки логов аудита
}

func ParseServerFlags() *ServerConfig {
	cfg := &ServerConfig{}
	var configFile string

	// 0) Определяем путь к JSON конфигу (из флага или env)
	flag.StringVar(&configFile, "c", "", "Path to JSON configuration file")
	flag.StringVar(&configFile, "config", "", "Path to JSON configuration file")

	// 1) Объявляем флаги
	var (
		flagAddress   string
		flagStoreInt  int
		flagRestore   bool
		flagFile      string
		flagDatabase  string
		flagKey       string
		flagCryptoKey string
		flagAuditFile string
		flagAuditURL  string
	)

	flag.StringVar(&flagAddress, "a", "", "HTTP server endpoint address")
	flag.IntVar(&flagStoreInt, "i", -1, "Store interval in seconds (0 = sync mode)")
	flag.StringVar(&flagFile, "f", "", "File storage path")
	flag.BoolVar(&flagRestore, "r", false, "Restore metrics from file")
	flag.StringVar(&flagDatabase, "d", "", "Database connection string")
	flag.StringVar(&flagKey, "k", "", "Key for hash calculation")
	flag.StringVar(&flagCryptoKey, "crypto-key", "", "Path to RSA private key file for decryption")
	flag.StringVar(&flagAuditFile, "audit-file", "", "Audit log file path")
	flag.StringVar(&flagAuditURL, "audit-url", "", "Audit log URL endpoint")

	flag.Parse()

	// Проверяем переменную окружения для конфигурационного файла
	if configFile == "" {
		if v, ok := os.LookupEnv("CONFIG"); ok {
			configFile = v
		}
	}

	// 2) Загружаем JSON конфигурацию (если указана)
	var jsonCfg *ServerJSONConfig
	if configFile != "" {
		var err error
		jsonCfg, err = LoadServerJSONConfig(configFile)
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

	// StoreInterval
	storeSeconds := 300 // default
	if jsonCfg != nil && jsonCfg.StoreInterval != "" {
		if d, err := time.ParseDuration(jsonCfg.StoreInterval); err == nil {
			storeSeconds = int(d.Seconds())
		}
	}
	if v, ok := os.LookupEnv("STORE_INTERVAL"); ok {
		if n, err := strconv.Atoi(v); err == nil {
			storeSeconds = n
		}
	}
	if flagStoreInt >= 0 {
		storeSeconds = flagStoreInt
	}

	// FileStoragePath
	cfg.FileStoragePath = "/tmp/metrics-db.json" // default
	if jsonCfg != nil && jsonCfg.StoreFile != "" {
		cfg.FileStoragePath = jsonCfg.StoreFile
	}
	if v, ok := os.LookupEnv("FILE_STORAGE_PATH"); ok {
		cfg.FileStoragePath = v
	}
	if flagFile != "" {
		cfg.FileStoragePath = flagFile
	}

	// Restore
	cfg.Restore = false // default
	if jsonCfg != nil && jsonCfg.Restore != nil {
		cfg.Restore = *jsonCfg.Restore
	}
	if v, ok := os.LookupEnv("RESTORE"); ok {
		if b, err := strconv.ParseBool(v); err == nil {
			cfg.Restore = b
		}
	}
	// Для boolean флага проверяем, был ли он явно установлен
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "r" {
			cfg.Restore = flagRestore
		}
	})

	// Database
	cfg.Database = "postgresql://postgres:password@localhost:5432/yandex_go?sslmode=disable&search_path=public" // default
	if jsonCfg != nil && jsonCfg.DatabaseDSN != "" {
		cfg.Database = jsonCfg.DatabaseDSN
	}
	if v, ok := os.LookupEnv("DATABASE_DSN"); ok {
		cfg.Database = v
	}
	if flagDatabase != "" {
		cfg.Database = flagDatabase
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

	// AuditFile
	cfg.AuditFile = "" // default
	if v, ok := os.LookupEnv("AUDIT_FILE"); ok {
		cfg.AuditFile = v
	}
	if flagAuditFile != "" {
		cfg.AuditFile = flagAuditFile
	}

	// AuditURL
	cfg.AuditURL = "" // default
	if v, ok := os.LookupEnv("AUDIT_URL"); ok {
		cfg.AuditURL = v
	}
	if flagAuditURL != "" {
		cfg.AuditURL = flagAuditURL
	}

	// 4) Производные поля
	cfg.StoreInterval = time.Duration(storeSeconds) * time.Second
	return cfg
}
