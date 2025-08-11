package main

import (
	"compress/gzip"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMetricsSender_SendJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/update", r.URL.Path, "Expected path /update")
		assert.Equal(t, http.MethodPost, r.Method, "Expected POST method")
		contentType := r.Header.Get("Content-Type")
		assert.Equal(t, "application/json", contentType, "Expected Content-Type: application/json")

		contentEncoding := r.Header.Get("Content-Encoding")
		assert.Equal(t, "gzip", contentEncoding, "Expected Content-Encoding: gzip")

		gzipReader, err := gzip.NewReader(r.Body)
		if err != nil {
			t.Fatalf("Failed to create gzip reader: %v", err)
		}
		defer gzipReader.Close()

		var metric Metrics
		err = json.NewDecoder(gzipReader).Decode(&metric)
		if err != nil {
			t.Fatalf("Failed to decode JSON: %v", err)
		}

		assert.Equal(t, "testMetric", metric.ID, "Expected metric ID 'testMetric'")
		assert.Equal(t, "gauge", metric.MType, "Expected metric type 'gauge'")
		assert.NotNil(t, metric.Value, "Value should not be nil")
		assert.Equal(t, 42.5, *metric.Value, "Expected value 42.5")

		w.WriteHeader(http.StatusOK)
	}))
	
	defer server.Close()

	sender := NewMetricsSender(server.Listener.Addr().String())

	value := 42.5
	metric := Metrics{ID: "testMetric", MType: "gauge", Value: &value}

	err := sender.SendJSON(metric)
	assert.NoError(t, err, "Sending should not produce error")
}
