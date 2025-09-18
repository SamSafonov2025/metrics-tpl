package main

import (
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMetricsSender_SendBatchJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// маршрут и метод
		assert.Equal(t, "/updates/", r.URL.Path, "Expected path /updates/")
		assert.Equal(t, http.MethodPost, r.Method, "Expected POST method")

		// заголовки
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "gzip", r.Header.Get("Content-Encoding"))

		// подпись передаётся, если ключ задан
		hash := r.Header.Get("HashSHA256")
		assert.NotEmpty(t, hash, "HashSHA256 header should be set")

		// распакуем тело
		gr, err := gzip.NewReader(r.Body)
		assert.NoError(t, err, "Failed to create gzip reader")
		defer gr.Close()
		raw, err := io.ReadAll(gr)
		assert.NoError(t, err, "Failed to read gzipped body")

		// это батч (массив) метрик
		var batch []Metrics
		err = json.Unmarshal(raw, &batch)
		assert.NoError(t, err, "Failed to decode JSON array")
		assert.Len(t, batch, 1, "Expected single metric in batch")

		metric := batch[0]
		assert.Equal(t, "testMetric", metric.ID)
		assert.Equal(t, "gauge", metric.MType)
		if assert.NotNil(t, metric.Value) {
			assert.Equal(t, 42.5, *metric.Value)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := NewMetricsSender(server.Listener.Addr().String(), "123")

	value := 42.5
	metric := Metrics{ID: "testMetric", MType: "gauge", Value: &value}

	err := sender.SendBatchJSON([]Metrics{metric})
	assert.NoError(t, err, "Sending should not produce error")
}
