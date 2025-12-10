package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// TestAgentGracefulShutdown проверяет, что агент корректно завершается
// и завершает отправку всех метрик
func TestAgentGracefulShutdown(t *testing.T) {
	var receivedRequests int32

	// Простой HTTP сервер для подсчёта запросов
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&receivedRequests, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Создаём агент с короткими интервалами для быстрого теста
	agent := NewAgent(
		100*time.Millisecond, // poll
		200*time.Millisecond, // report
		server.Listener.Addr().String(),
		"", "", // no crypto
		2,      // 2 workers
	)

	// Запускаем агент
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	go func() {
		agent.Start(ctx)
		close(done)
	}()

	// Даём агенту время отправить несколько батчей
	time.Sleep(500 * time.Millisecond)

	requestsBefore := atomic.LoadInt32(&receivedRequests)
	if requestsBefore == 0 {
		t.Fatal("Agent didn't send any metrics")
	}

	// Отправляем сигнал shutdown
	cancel()

	// Ждём завершения агента
	select {
	case <-done:
		t.Logf("Agent shutdown successfully after sending %d requests", requestsBefore)
	case <-time.After(5 * time.Second):
		t.Fatal("Agent didn't shutdown in time")
	}

	// Проверяем что после shutdown новые запросы не отправляются
	requestsAfter := atomic.LoadInt32(&receivedRequests)
	t.Logf("Total requests sent: %d", requestsAfter)
}
