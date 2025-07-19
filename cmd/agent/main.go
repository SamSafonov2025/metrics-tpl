package main

import (
	"flag"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"time"
)

type Agent struct {
	pollInterval   time.Duration
	reportInterval time.Duration
	serverAddress  string
	pollCount      int64
}

func NewAgent(pollInterval, reportInterval time.Duration, serverAddress string) *Agent {
	return &Agent{
		pollInterval:   pollInterval,
		reportInterval: reportInterval,
		serverAddress:  serverAddress,
		pollCount:      0,
	}
}

func (a *Agent) collectMetrics() map[string]float64 {
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

func (a *Agent) sendMetrics(metricType, metricName string, value float64) {
	metricValue := strconv.FormatFloat(value, 'f', -1, 64)
	url := fmt.Sprintf("http://%s/update/%s/%s/%s", a.serverAddress, metricType, metricName, metricValue)
	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return
	}
	req.Header.Set("Content-type", "text-plain")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return
	}

	defer resp.Body.Close()
	fmt.Printf("Metrics %s (%s) with value %s sent successfully\n", metricName, metricType, metricValue)
}

func (a *Agent) start() {
	ticker := time.NewTicker(a.pollInterval)
	reportTicker := time.NewTicker(a.reportInterval)
	for {
		select {
		case <-ticker.C:
			a.pollCount++
			metrics := a.collectMetrics()
			metrics["PollCount"] = float64(a.pollCount)
		case <-reportTicker.C:
			metrics := a.collectMetrics()
			metrics["PollCount"] = float64(a.pollCount)

			for name, value := range metrics {
				a.sendMetrics("gauge", name, value)
			}
			a.sendMetrics("counter", "PollCount", float64(a.pollCount))
		}
	}
}

func main() {
	// Define flags
	serverAddress := flag.String("a", "localhost:8080", "HTTP server endpoint address")
	pollInterval := flag.Int("p", 2, "Poll interval in seconds")
	reportInterval := flag.Int("r", 10, "Report interval in seconds")

	// Parse flags
	flag.Parse()

	// Check for unknown flags
	if flag.NArg() > 0 {
		fmt.Fprintf(os.Stderr, "Error: unknown flag(s): %v\n", flag.Args())
		os.Exit(1)
	}

	// Create agent with configured values
	agent := NewAgent(
		time.Duration(*pollInterval)*time.Second,
		time.Duration(*reportInterval)*time.Second,
		*serverAddress,
	)

	agent.start()
}
