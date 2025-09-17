package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/SamSafonov2025/metrics-tpl/internal/crypto"
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
			if i > 0 {
				fmt.Printf("agent: retry succeeded on attempt %d/%d\n", i+1, attempts)
			}
			return nil
		}
		// логируем результат попытки
		retry := isRetryable(err) && i < len(backoffs)
		if retry {
			fmt.Printf("agent: attempt %d/%d failed: %v — next retry in %s\n", i+1, attempts, err, backoffs[i])
		} else {
			fmt.Printf("agent: attempt %d/%d failed: %v — no more retries\n", i+1, attempts, err)
			return err
		}
		select {
		case <-time.After(backoffs[i]):
		case <-ctx.Done():
			fmt.Println("agent: retry aborted by context:", ctx.Err())
			return ctx.Err()
		}
	}
	return nil
}

// ———— helpers ————

func shortStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "...(truncated)"
}

func shortBytes(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "...(truncated)"
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
	req.Header.Set("HashSHA256", hash)

	// лог до запроса
	const maxDump = 512
	fmt.Printf(
		"agent: POST %s | json=%dB gz=%dB | hash=%s | headers={Content-Type:%s, Content-Encoding:%s}\njson_preview=%s\n",
		urlStr, len(jsonData), buf.Len(), maskHash(hash),
		req.Header.Get("Content-Type"), req.Header.Get("Content-Encoding"),
		shortBytes(jsonData, maxDump),
	)

	start := time.Now()
	resp, err := s.client.Do(req)
	dur := time.Since(start)
	if err != nil {
		fmt.Printf("agent: request error (%s) after %s: %v\n", path, dur, err)
		return err
	}
	defer resp.Body.Close()

	// читаем кусок ответа (полезно при 400 от middleware)
	body, _ := io.ReadAll(io.LimitReader(resp.Body, int64(maxDump)))
	bodyStr := strings.TrimSpace(string(body))

	fmt.Printf("agent: response %s -> %d in %s | resp_preview=%s\n", path, resp.StatusCode, dur, shortStr(bodyStr, maxDump))

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
	fmt.Printf("agent: sending batch (%d metrics) -> /updates/\n", len(batch))

	// пробуем /updates/ c ретраями
	err := retryCtx(ctx, func() error {
		return s.postGzJSONCtx(ctx, "/updates/", batch)
	}, isRetryableHTTPOrNetErr)
	if err == nil {
		fmt.Println("agent: batch sent successfully")
		return nil
	}

	// фолбэк: по одной (каждая с собственными ретраями) — не сбиваем весь батч из-за одной метрики
	fmt.Printf("agent: batch send failed (%v), falling back to single sends...\n", err)
	var firstErr error
	for i, m := range batch {
		if e := s.SendJSONCtx(ctx, m); e != nil {
			fmt.Printf("agent: single send failed for #%d (%s/%s): %v\n", i, m.MType, m.ID, e)
			if firstErr == nil {
				firstErr = e
			}
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

func NewAgent(pollInterval, reportInterval time.Duration, serverAddress string, cryptoKey string) *Agent {
	return &Agent{
		pollInterval:   pollInterval,
		reportInterval: reportInterval,
		collector:      NewMetricsCollector(),
		sender:         NewMetricsSender(serverAddress, cryptoKey),
	}
}

func (a *Agent) Start(ctx context.Context) {
	pollTicker := time.NewTicker(a.pollInterval)
	reportTicker := time.NewTicker(a.reportInterval)
	defer pollTicker.Stop()
	defer reportTicker.Stop()

	fmt.Printf("agent: started | poll=%s report=%s | server=%s | hmac=%t\n",
		a.pollInterval, a.reportInterval, a.sender.serverAddress, a.sender.cryptoKey != "")

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

			// немного телеметрии перед отправкой
			fmt.Printf("agent: report tick | gauges=%d counters=1 pollCount=%d\n", len(collected), delta)

			// локальный deadline на отправку всего батча
			sendCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
			err := a.sender.SendBatchJSONCtx(sendCtx, batch)
			cancel()

			if err != nil {
				fmt.Printf("agent: error sending batch: %v\n", err)
				// не обнуляем pollCount — доберём в следующем репорте
			} else {
				a.collector.pollCount = 0
			}
		}
	}
}

// --- Backward-compat wrappers for tests ---

// SendJSON — старая сигнатура, нужна для тестов.
// Делегирует в SendJSONCtx с context.Background().
func (s *MetricsSender) SendJSON(metric Metrics) error {
	return s.SendJSONCtx(context.Background(), metric)
}

// SendBatchJSON — на случай, если тесты дернут батч без ctx.
func (s *MetricsSender) SendBatchJSON(batch []Metrics) error {
	return s.SendBatchJSONCtx(context.Background(), batch)
}

func main() {
	cfg := config.ParseAgentFlags()

	agent := NewAgent(cfg.PollInterval, cfg.ReportInterval, cfg.ServerAddress, cfg.CryptoKey)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	agent.Start(ctx)
}
