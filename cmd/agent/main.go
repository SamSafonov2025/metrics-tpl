package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/SamSafonov2025/metrics-tpl/internal/config"
)

type Metrics struct {
	ID    string   `json:"id"`
	MType string   `json:"type"`
	Delta *int64   `json:"delta,omitempty"`
	Value *float64 `json:"value,omitempty"`
}

type MetricsCollector struct{ pollCount int64 }

func NewMetricsCollector() *MetricsCollector    { return &MetricsCollector{} }
func (m *MetricsCollector) IncrementPollCount() { m.pollCount++ }

func (m *MetricsCollector) Collect() map[string]float64 {
	memStats := new(runtime.MemStats)
	runtime.ReadMemStats(memStats)
	return map[string]float64{
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
}

type MetricsSender struct {
	serverAddress string
	client        *http.Client
}

func NewMetricsSender(serverAddress string) *MetricsSender {
	return &MetricsSender{
		serverAddress: serverAddress,
		client:        &http.Client{Timeout: 5 * time.Second},
	}
}

// ———— RETRY CORE ————

var backoffs = []time.Duration{1 * time.Second, 3 * time.Second, 5 * time.Second}

type httpStatusError int

func (e httpStatusError) Error() string { return fmt.Sprintf("http status %d", int(e)) }

func isRetryableHTTPOrNetErr(err error) bool {
	if err == nil {
		return false
	}
	// сетевые/транспортные ошибки
	var ue *url.Error
	if errors.As(err, &ue) {
		// ContextCanceled — не ретраем; DeadlineExceeded — ретраем
		if errors.Is(ue.Err, context.Canceled) {
			return false
		}
		return true
	}
	var ne net.Error
	if errors.As(err, &ne) {
		return ne.Timeout()
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	// HTTP коды (заворачиваем в httpStatusError ниже)
	var se httpStatusError
	if errors.As(err, &se) {
		switch int(se) {
		case 408, 425, 429, 500, 502, 503, 504:
			return true
		default:
			return false
		}
	}
	return false
}

func retryCtx(ctx context.Context, fn func() error, isRetryable func(error) bool) error {
	attempts := len(backoffs) + 1
	for i := 0; i < attempts; i++ {
		err := fn()
		if err == nil {
			return nil
		}
		if !isRetryable(err) || i == len(backoffs) {
			return err
		}
		select {
		case <-time.After(backoffs[i]):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

// ———— HTTP helpers ————

func (s *MetricsSender) postGzJSONCtx(ctx context.Context, path string, payload any) error {
	jsonData, err := json.Marshal(payload)
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

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("http://%s%s", s.serverAddress, path), &buf)
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
		return httpStatusError(resp.StatusCode)
	}
	return nil
}

// Старая одиночная отправка с контекстом + retry
func (s *MetricsSender) SendJSONCtx(ctx context.Context, metric Metrics) error {
	return retryCtx(ctx, func() error {
		return s.postGzJSONCtx(ctx, "/update", metric)
	}, isRetryableHTTPOrNetErr)
}

// Батч с фолбэком по одной
func (s *MetricsSender) SendBatchJSONCtx(ctx context.Context, batch []Metrics) error {
	if len(batch) == 0 {
		return nil
	}
	// пробуем /updates/ c ретраями
	err := retryCtx(ctx, func() error {
		return s.postGzJSONCtx(ctx, "/updates/", batch)
	}, isRetryableHTTPOrNetErr)
	if err == nil {
		return nil
	}

	// фолбэк: по одной (каждая с собственными ретраями) — не сбиваем весь батч из-за одной метрики
	fmt.Printf("Batch send failed (%v), falling back to single sends...\n", err)
	var firstErr error
	for _, m := range batch {
		if e := s.SendJSONCtx(ctx, m); e != nil && firstErr == nil {
			firstErr = e
		}
	}
	return firstErr
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

func (a *Agent) Start(ctx context.Context) {
	pollTicker := time.NewTicker(a.pollInterval)
	reportTicker := time.NewTicker(a.reportInterval)
	defer pollTicker.Stop()
	defer reportTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			fmt.Println("agent: shutdown")
			return

		case <-pollTicker.C:
			a.collector.IncrementPollCount()

		case <-reportTicker.C:
			collected := a.collector.Collect()
			batch := make([]Metrics, 0, len(collected)+1)
			for name, value := range collected {
				val := value
				batch = append(batch, Metrics{ID: name, MType: "gauge", Value: &val})
			}
			delta := a.collector.pollCount
			batch = append(batch, Metrics{ID: "PollCount", MType: "counter", Delta: &delta})

			// локальный deadline на отправку всего батча
			sendCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
			err := a.sender.SendBatchJSONCtx(sendCtx, batch)
			cancel()

			if err != nil {
				fmt.Printf("Error sending batch: %v\n", err)
				// не обнуляем pollCount — доберём в следующем репорте
			} else {
				a.collector.pollCount = 0
			}
		}
	}
}

func main() {
	cfg := config.ParseAgentFlags()

	agent := NewAgent(cfg.PollInterval, cfg.ReportInterval, cfg.ServerAddress)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	agent.Start(ctx)
}
