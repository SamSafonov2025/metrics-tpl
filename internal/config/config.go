package config

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	ServerAddress   string
	PollInterval    time.Duration
	ReportInterval  time.Duration
	StoreInterval   time.Duration
	FileStoragePath string
	Restore         bool
}

func ParseFlags() *Config {
	cfg := &Config{}

	var pollSeconds, reportSeconds, storeSeconds int
	var restore bool

	// Get values from environment variables if they exist
	serverAddrEnv, serverAddrExists := os.LookupEnv("ADDRESS")
	pollIntervalEnv, pollIntervalExists := os.LookupEnv("POLL_INTERVAL")
	reportIntervalEnv, reportIntervalExists := os.LookupEnv("REPORT_INTERVAL")
	storeIntervalEnv, storeIntervalExists := os.LookupEnv("STORE_INTERVAL")
	fileStoragePathEnv, fileStoragePathExists := os.LookupEnv("FILE_STORAGE_PATH")
	restoreEnv, restoreExists := os.LookupEnv("RESTORE")

	// Set default values for flags
	defaultServerAddr := "localhost:8080"
	defaultPollInterval := 2
	defaultReportInterval := 10
	defaultStoreInterval := 300
	defaultFileStoragePath := "/tmp/metrics-db.json"
	defaultRestore := true

	if serverAddrExists {
		defaultServerAddr = serverAddrEnv
	}
	if pollIntervalExists {
		if val, err := strconv.Atoi(pollIntervalEnv); err == nil {
			defaultPollInterval = val
		}
	}
	if reportIntervalExists {
		if val, err := strconv.Atoi(reportIntervalEnv); err == nil {
			defaultReportInterval = val
		}
	}
	if storeIntervalExists {
		if val, err := strconv.Atoi(storeIntervalEnv); err == nil {
			defaultStoreInterval = val
		}
	}
	if fileStoragePathExists {
		defaultFileStoragePath = fileStoragePathEnv
	}
	if restoreExists {
		if val, err := strconv.ParseBool(restoreEnv); err == nil {
			defaultRestore = val
		}
	}

	flag.StringVar(&cfg.ServerAddress, "a", defaultServerAddr, "HTTP server endpoint address")
	flag.IntVar(&pollSeconds, "p", defaultPollInterval, "Poll interval in seconds")
	flag.IntVar(&reportSeconds, "r", defaultReportInterval, "Report interval in seconds")
	flag.IntVar(&storeSeconds, "i", defaultStoreInterval, "Store interval in seconds")
	flag.StringVar(&cfg.FileStoragePath, "f", defaultFileStoragePath, "File storage path")
	flag.BoolVar(&restore, "restore", defaultRestore, "Restore metrics from file")

	flag.Parse()

	if flag.NArg() > 0 {
		fmt.Fprintf(os.Stderr, "Error: unknown flag(s): %v\n", flag.Args())
		os.Exit(1)
	}

	cfg.PollInterval = time.Duration(pollSeconds) * time.Second
	cfg.ReportInterval = time.Duration(reportSeconds) * time.Second
	cfg.StoreInterval = time.Duration(storeSeconds) * time.Second
	cfg.Restore = restore

	return cfg
}
