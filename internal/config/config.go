package config

import (
	"flag"
	"fmt"
	"os"
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

	flag.StringVar(&cfg.ServerAddress, "a", "localhost:8080", "HTTP server endpoint address")
	flag.IntVar(&pollSeconds, "p", 2, "Poll interval in seconds")
	flag.IntVar(&reportSeconds, "r", 10, "Report interval in seconds")

	flag.Parse()

	if flag.NArg() > 0 {
		fmt.Fprintf(os.Stderr, "Error: unknown flag(s): %v\n", flag.Args())
		os.Exit(1)
	}

	cfg.PollInterval = time.Duration(pollSeconds) * time.Second
	cfg.ReportInterval = time.Duration(reportSeconds) * time.Second

	return cfg
}
