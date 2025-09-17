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
}

func ParseServerFlags() *ServerConfig {
	cfg := &ServerConfig{}
	var storeSeconds int

	addr := getEnv("ADDRESS", "localhost:8080")
	store := atoiEnv("STORE_INTERVAL", 300)
	path := getEnv("FILE_STORAGE_PATH", "/tmp/metrics-db.json")
	restore := boolEnv("RESTORE", false)
	database := getEnv("DATABASE_DSN", "postgresql://postgres:arzamas17@localhost:5432/yandex_go?sslmode=disable&search_path=public")
	сryptoKey := getEnv("KEY", "")

	flag.StringVar(&cfg.ServerAddress, "a", addr, "HTTP server endpoint address")
	flag.IntVar(&storeSeconds, "i", store, "Store interval in seconds (0 = sync mode)")
	flag.StringVar(&cfg.FileStoragePath, "f", path, "File storage path")
	flag.BoolVar(&cfg.Restore, "r", restore, "Restore metrics from file")
	flag.StringVar(&cfg.Database, "d", database, "Database connection string")
	flag.StringVar(&cfg.CryptoKey, "k", сryptoKey, "Key for hash calculation")

	flag.Parse()

	cfg.StoreInterval = time.Duration(storeSeconds) * time.Second
	return cfg
}

func boolEnv(k string, def bool) bool {
	if v, ok := os.LookupEnv(k); ok {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}
