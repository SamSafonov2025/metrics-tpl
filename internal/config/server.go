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
}

func ParseServerFlags() *ServerConfig {
	cfg := &ServerConfig{}
	var storeSeconds int

	addr := getEnv("ADDRESS", "localhost:8080")
	store := atoiEnv("STORE_INTERVAL", 300)
	path := getEnv("FILE_STORAGE_PATH", "/tmp/metrics-db.json")
	restore := boolEnv("RESTORE", false)
	database := getEnv("DATABASE_DSN", "//postgresql://postgres:arzamas17@localhost:5432/yandex_go?schema=public")

	flag.StringVar(&cfg.ServerAddress, "a", addr, "HTTP server endpoint address")
	flag.IntVar(&storeSeconds, "i", store, "Store interval in seconds (0 = sync mode)")
	flag.StringVar(&path, "f", path, "File storage path")
	flag.BoolVar(&restore, "r", restore, "Restore metrics from file")
	flag.StringVar(&database, "d", database, "Database connection string")
	flag.Parse()

	cfg.StoreInterval = time.Duration(storeSeconds) * time.Second
	cfg.FileStoragePath = path
	cfg.Restore = restore
	cfg.Database = database
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
