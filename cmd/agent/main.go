package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"runtime"
	"time"

	"github.com/SamSafonov2025/metrics-tpl/internal/config"
)

type Metrics struct {
	ID    string   `json:"id"`
	MType string   `json:"type"`
	Delta *int64   `json:"delta,omitempty"`
	Value *float64 `json:"value,omitempty"`
}

type MetricsCollector struct {
	pollCount int64
}

func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{pollCount: 0}
}

func (m *MetricsCollector) IncrementPollCount() {
	m.pollCount++
}

func (m *MetricsCollector) Collect() map[string]float64 {
	memStats := new(runtime.MemStats)
	runtime.ReadMemStats(memStats)

	metrics := map[string]float64{
		"Alloc":         float64(memStats.Alloc),
		"BuckHashSys":   float64(memStats.BuckHashSys),
		"Frees":         float64(memStats.Frees),
		"GCCPUFraction": memStats.GCCPUFraction,
		"GCSys":         float64(memStats.GCSys),
		"HeapAlloc":     float64(memStats.HeapAlloc),
		"HeapIdle":      float64(memStats.HeapIdle),
		"HeapInuse":     float64(memStats.HeapInuse),
		"HeapObjects":   float64(memStats.HeapObjects),
		"HeapReleased":  float64(memStats.HeapReleased),
		"HeapSys":       float64(memStats.HeapSys),
		"LastGC":        float64(memStats.LastGC),
		"Lookups":       float64(memStats.Lookups),
		"MCacheInuse":   float64(memStats.MCacheInuse),
		"MCacheSys":     float64(memStats.MCacheSys),
		"MSpanInuse":    float64(memStats.MSpanInuse),
		"MSpanSys":      float64(memStats.MSpanSys),
		"Mallocs":       float64(memStats.Mallocs),
		"NextGC":        float64(memStats.NextGC),
		"NumForcedGC":   float64(memStats.NumForcedGC),
		"NumGC":         float64(memStats.NumGC),
		"OtherSys":      float64(memStats.OtherSys),
		"PauseTotalNs":  float64(memStats.PauseTotalNs),
		"StackInuse":    float64(memStats.StackInuse),
		"StackSys":      float64(memStats.StackSys),
		"Sys":           float64(memStats.Sys),
		"TotalAlloc":    float64(memStats.TotalAlloc),
		"RandomValue":   rand.Float64() * 100,
	}
	return metrics
}

type MetricsSender struct {
	serverAddress string
	client        *http.Client
}

func NewMetricsSender(serverAddress string) *MetricsSender {
	return &MetricsSender{
		serverAddress: serverAddress,
		client:        &http.Client{},
	}
}

func (s *MetricsSender) SendJSON(metric Metrics) error {
	jsonData, err := json.Marshal(metric)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write(jsonData); err != nil {
		return err
	}
	if err := gz.Close(); err != nil {
		return err
	}

	url := fmt.Sprintf("http://%s/update", s.serverAddress)
	req, err := http.NewRequest(http.MethodPost, url, &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned non-200 status: %d", resp.StatusCode)
	}

	return nil
}

type Agent struct {
	pollInterval   time.Duration
	reportInterval time.Duration
	collector      *MetricsCollector
	sender         *MetricsSender
}

func NewAgent(pollInterval, reportInterval time.Duration, serverAddress string) *Agent {
	return &Agent{
		pollInterval:   pollInterval,
		reportInterval: reportInterval,
		collector:      NewMetricsCollector(),
		sender:         NewMetricsSender(serverAddress),
	}
}

func (a *Agent) start() {
	ticker := time.NewTicker(a.pollInterval)
	reportTicker := time.NewTicker(a.reportInterval)
	for {
		select {
		case <-ticker.C:
			a.collector.IncrementPollCount()
		case <-reportTicker.C:
			metrics := a.collector.Collect()
			for name, value := range metrics {
				val := value
				err := a.sender.SendJSON(Metrics{ID: name, MType: "gauge", Value: &val})
				if err != nil {
					fmt.Printf("Error sending gauge %s: %v\n", name, err)
				}
			}
			delta := a.collector.pollCount
			err := a.sender.SendJSON(Metrics{ID: "PollCount", MType: "counter", Delta: &delta})
			if err != nil {
				fmt.Printf("Error sending counter PollCount: %v\n", err)
			}
			a.collector.pollCount = 0
		}
	}
}

func main() {
	cfg := config.ParseFlags()

	agent := NewAgent(
		cfg.PollInterval,
		cfg.ReportInterval,
		cfg.ServerAddress,
	)

	agent.start()
}
