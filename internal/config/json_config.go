package config

import (
	"encoding/json"
	"os"
)

// ServerJSONConfig represents the JSON configuration format for server
type ServerJSONConfig struct {
	Address       string `json:"address"`
	Restore       *bool  `json:"restore,omitempty"`
	StoreInterval string `json:"store_interval,omitempty"`
	StoreFile     string `json:"store_file,omitempty"`
	DatabaseDSN   string `json:"database_dsn,omitempty"`
	CryptoKey     string `json:"crypto_key,omitempty"`
}

// AgentJSONConfig represents the JSON configuration format for agent
type AgentJSONConfig struct {
	Address        string `json:"address"`
	ReportInterval string `json:"report_interval,omitempty"`
	PollInterval   string `json:"poll_interval,omitempty"`
	CryptoKey      string `json:"crypto_key,omitempty"`
}

// LoadServerJSONConfig loads server configuration from JSON file
func LoadServerJSONConfig(filename string) (*ServerJSONConfig, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var cfg ServerJSONConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// LoadAgentJSONConfig loads agent configuration from JSON file
func LoadAgentJSONConfig(filename string) (*AgentJSONConfig, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var cfg AgentJSONConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
