package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"path"
	"runtime"
	"strconv"
	"time"

	"github.com/SamSafonov2025/metrics-tpl/internal/config"
)

type MetricsCollector struct {
	pollCount int64
}

func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{pollCount: 0}
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
		"PollCount":     float64(m.pollCount),
	}

	m.pollCount++
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

func (s *MetricsSender) Send(metricType, metricName string, value float64) {
	metricValue := strconv.FormatFloat(value, 'f', -1, 64)
	baseURL := fmt.Sprintf("http://%s", s.serverAddress)
	u, _ := url.Parse(baseURL)
	u.Path = path.Join("/update", metricType, metricName, metricValue)
	url := u.String()

	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return
	}
	req.Header.Set("Content-Type", "text/plain")

	resp, err := s.client.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return
	}

	defer resp.Body.Close()
	fmt.Printf("Metrics %s (%s) with value %s sent successfully\n", metricName, metricType, metricValue)
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
			a.collector.Collect()
		case <-reportTicker.C:
			metrics := a.collector.Collect()
			for name, value := range metrics {
				a.sender.Send("gauge", name, value)
			}
			a.sender.Send("counter", "PollCount", float64(a.collector.pollCount-1))
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
