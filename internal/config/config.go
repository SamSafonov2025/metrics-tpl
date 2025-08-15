package config

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	ServerAddress  string
	PollInterval   time.Duration
	ReportInterval time.Duration
}

func ParseFlags() *Config {
	cfg := &Config{}

	var pollSeconds, reportSeconds int

	// Get values from environment variables if they exist
	serverAddrEnv, serverAddrExists := os.LookupEnv("ADDRESS")
	pollIntervalEnv, pollIntervalExists := os.LookupEnv("POLL_INTERVAL")
	reportIntervalEnv, reportIntervalExists := os.LookupEnv("REPORT_INTERVAL")

	// Set default values for flags
	defaultServerAddr := "localhost:8080"
	defaultPollInterval := 2
	defaultReportInterval := 10

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

	flag.StringVar(&cfg.ServerAddress, "a", defaultServerAddr, "HTTP server endpoint address")
	flag.IntVar(&pollSeconds, "p", defaultPollInterval, "Poll interval in seconds")
	flag.IntVar(&reportSeconds, "r", defaultReportInterval, "Report interval in seconds")

	flag.Parse()

	if flag.NArg() > 0 {
		fmt.Fprintf(os.Stderr, "Error: unknown flag(s): %v\n", flag.Args())
		os.Exit(1)
	}

	cfg.PollInterval = time.Duration(pollSeconds) * time.Second
	cfg.ReportInterval = time.Duration(reportSeconds) * time.Second

	return cfg
}
