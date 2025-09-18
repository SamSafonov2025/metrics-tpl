// cmd/agent/main.go
package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/SamSafonov2025/metrics-tpl/internal/crypto"
	"github.com/SamSafonov2025/metrics-tpl/internal/logger"
	"go.uber.org/zap"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/SamSafonov2025/metrics-tpl/internal/config"

	// NEW: gopsutil
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
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

func (m *MetricsCollector) CollectRuntime() map[string]float64 {
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

// ---- NEW: системные метрики через gopsutil
func CollectSystemGauges() []Metrics {
	var out []Metrics

	vm, err := mem.VirtualMemory()
	if err == nil {
		tm := float64(vm.Total)
		fm := float64(vm.Free)
		out = append(out,
			Metrics{ID: "TotalMemory", MType: "gauge", Value: &tm},
			Metrics{ID: "FreeMemory", MType: "gauge", Value: &fm},
		)
	}

	// per-CPU загрузка; длина среза == числу CPU на хосте в рантайме
	// cpu.Percent(0,true) — мгновенный срез с момента предыдущего вызова
	if per, err := cpu.Percent(0, true); err == nil {
		for i, v := range per {
			val := v
			out = append(out, Metrics{
				ID:    fmt.Sprintf("CPUutilization%d", i+1),
				MType: "gauge",
				Value: &val,
			})
		}
	}
	return out
}

type MetricsSender struct {
	serverAddress string
	client        *http.Client
	cryptoKey     string
}

func NewMetricsSender(serverAddress string, cryptoKey string) *MetricsSender {
	return &MetricsSender{
		serverAddress: serverAddress,
		client:        &http.Client{Timeout: 5 * time.Second},
		cryptoKey:     cryptoKey,
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
		// ContextCanceled — не ретраем; всё остальное — ретраем
		return !errors.Is(ue.Err, context.Canceled)
	}
	var ne net.Error
	if errors.As(err, &ne) {
		return ne.Timeout()
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
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
			if i > 0 {
				fmt.Printf("agent: retry succeeded on attempt %d/%d\n", i+1, attempts)
			}
			return nil
		}
		retry := isRetryable(err) && i < len(backoffs)
		if retry {
			fmt.Printf("agent: attempt %d/%d failed: %v — next in %s\n", i+1, attempts, err, backoffs[i])
		} else {
			fmt.Printf("agent: attempt %d/%d failed: %v — stop\n", i+1, attempts, err)
			return err
		}
		select {
		case <-time.After(backoffs[i]):
		case <-ctx.Done():
			fmt.Println("agent: retry aborted:", ctx.Err())
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

	urlStr := fmt.Sprintf("http://%s%s", s.serverAddress, path)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, urlStr, &buf)
	if err != nil {
		return err
	}

	hash := crypto.GenerateHash(jsonData, s.cryptoKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	if s.cryptoKey != "" {
		req.Header.Set("HashSHA256", hash) // подписываем ДЕГЗИПНУТОЕ json-тело
	}

	const maxDump = 512
	fmt.Printf("agent: POST %s | json=%dB gz=%dB | hash=%s\n",
		urlStr, len(jsonData), buf.Len(), maskHash(hash))

	start := time.Now()
	resp, err := s.client.Do(req)
	dur := time.Since(start)
	if err != nil {
		fmt.Printf("agent: request error (%s) after %s: %v\n", path, dur, err)
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, int64(maxDump)))
	fmt.Printf("agent: response %s -> %d in %s | preview=%s\n",
		path, resp.StatusCode, dur, shortStr(strings.TrimSpace(string(body)), maxDump))

	if resp.StatusCode != http.StatusOK {
		return httpStatusError(resp.StatusCode)
	}
	return nil
}

func (s *MetricsSender) SendBatchJSONCtx(ctx context.Context, batch []Metrics) error {
	if len(batch) == 0 {
		return nil
	}
	fmt.Printf("agent: sending batch (%d metrics) -> /updates/\n", len(batch))
	err := retryCtx(ctx, func() error { return s.postGzJSONCtx(ctx, "/updates/", batch) }, isRetryableHTTPOrNetErr)
	if err == nil {
		fmt.Println("agent: batch sent successfully")
		return nil
	}
	// fallback: по одной
	fmt.Printf("agent: batch send failed (%v), fallback to singles...\n", err)
	var firstErr error
	for i, m := range batch {
		e := retryCtx(ctx, func() error { return s.postGzJSONCtx(ctx, "/update", m) }, isRetryableHTTPOrNetErr)
		if e != nil && firstErr == nil {
			firstErr = e
		}
		if e != nil {
			fmt.Printf("agent: single send failed for #%d (%s/%s): %v\n", i, m.MType, m.ID, e)
		}
	}
	return firstErr
}

// ———— Agent c worker pool ————
type Agent struct {
	pollInterval   time.Duration
	reportInterval time.Duration
	collector      *MetricsCollector
	sender         *MetricsSender

	rateLimit int
	jobs      chan []Metrics
}

func NewAgent(pollInterval, reportInterval time.Duration, serverAddress, cryptoKey string, rateLimit int) *Agent {
	if rateLimit < 1 {
		rateLimit = 1
	}
	return &Agent{
		pollInterval:   pollInterval,
		reportInterval: reportInterval,
		collector:      NewMetricsCollector(),
		sender:         NewMetricsSender(serverAddress, cryptoKey),
		rateLimit:      rateLimit,
		// небольшой буфер, чтобы сбор не стопорился при кратковременных всплесках
		jobs: make(chan []Metrics, rateLimit*2),
	}
}

func (a *Agent) Start(ctx context.Context) {
	// отдельные горутины: (1) runtime-sampler, (2) runtime-reporter, (3) system-collector, (4) воркеры отправки
	pollTicker := time.NewTicker(a.pollInterval)
	reportTicker := time.NewTicker(a.reportInterval)
	defer pollTicker.Stop()
	defer reportTicker.Stop()

	fmt.Printf("agent: started | poll=%s report=%s | server=%s | hmac=%t | workers=%d\n",
		a.pollInterval, a.reportInterval, a.sender.serverAddress, a.sender.cryptoKey != "", a.rateLimit)

	// (4) стартуем пул отправителей
	for i := 0; i < a.rateLimit; i++ {
		go func(id int) {
			for {
				select {
				case <-ctx.Done():
					return
				case batch := <-a.jobs:
					sendCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
					_ = a.sender.SendBatchJSONCtx(sendCtx, batch)
					cancel()
				}
			}
		}(i + 1)
	}

	// (1) инкрементируем pollCount по pollInterval
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-pollTicker.C:
				a.collector.IncrementPollCount()
			}
		}
	}()

	// (2) каждые reportInterval — формируем батч из runtime + PollCount и кладём в очередь
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-reportTicker.C:
				collected := a.collector.CollectRuntime()
				batch := make([]Metrics, 0, len(collected)+1)
				for name, value := range collected {
					val := value
					batch = append(batch, Metrics{ID: name, MType: "gauge", Value: &val})
				}
				delta := a.collector.pollCount
				batch = append(batch, Metrics{ID: "PollCount", MType: "counter", Delta: &delta})

				fmt.Printf("agent: enqueue runtime report | gauges=%d counters=1 pollCount=%d\n", len(collected), delta)
				a.jobs <- batch
				// сбрасываем pollCount только ПОСЛЕ постановки в очередь
				a.collector.pollCount = 0
			}
		}
	}()

	// (3) системные метрики с тем же pollInterval (можно сделать отдельный интервал, если нужно)
	go func() {
		sysTicker := time.NewTicker(a.pollInterval)
		defer sysTicker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-sysTicker.C:
				sysMetrics := CollectSystemGauges()
				if len(sysMetrics) > 0 {
					fmt.Printf("agent: enqueue system gauges | n=%d\n", len(sysMetrics))
					a.jobs <- sysMetrics
				}
			}
		}
	}()

	// ожидание сигнала завершения
	<-ctx.Done()
	fmt.Println("agent: shutdown")
}

// — обёртки для тестов (оставляем как было) —
func (s *MetricsSender) SendBatchJSON(batch []Metrics) error {
	return s.SendBatchJSONCtx(context.Background(), batch)
}

// ===== helpers (оставлены как в оригинале) =====
func shortStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "...(truncated)"
}
func maskHash(h string) string {
	if h == "" {
		return ""
	}
	if len(h) <= 10 {
		return h
	}
	return h[:10] + "…"
}

// ---- main ----
func main() {
	cfg := config.ParseAgentFlags()

	if err := logger.Init(); err != nil {
		panic(err)
	}

	logger.GetLogger().Info("Agent config loaded",
		zap.String("server_address", cfg.ServerAddress),
		zap.Duration("poll_interval", cfg.PollInterval),
		zap.Duration("report_interval", cfg.ReportInterval),
		zap.String("crypto_key", cfg.CryptoKey),
		zap.Int("rate_limit", cfg.RateLimit),
	)

	agent := NewAgent(cfg.PollInterval, cfg.ReportInterval, cfg.ServerAddress, cfg.CryptoKey, cfg.RateLimit)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	agent.Start(ctx)
}
