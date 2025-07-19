package handlers

import (
	"log"
	"net/http"
	"strconv"

	"github.com/SamSafonov2025/metrics-tpl/cmd/server/storage"
	"github.com/go-chi/chi/v5"
)

func logRequest(r *http.Request) {
	log.Printf("Request: %s %s", r.Method, r.URL.Path)
}

func logResponse(status int) {
	log.Printf("Response: %d", status)
}

func HomeHandle(storage *storage.MemStorage, router *chi.Mux) {
	router.Get("/", func(rw http.ResponseWriter, r *http.Request) {
		logRequest(r)
		defer func() { logResponse(http.StatusOK) }()

		body := "<h4>Gauges</h4>"
		for gaugeName, value := range storage.GetAllGauges() {
			body += gaugeName + ": " + strconv.FormatFloat(value, 'f', -1, 64) + "</br>"
		}
		body += "<h4>Counters</h4>"

		for counterName, value := range storage.GetAllCounters() {
			body += counterName + ": " + strconv.FormatInt(value, 10) + "</br>"
		}
		rw.Header().Set("Content-Type", "text/html; charset=utf-8")
		rw.Write([]byte(body))
	})
}

func UpdateHandler(storage *storage.MemStorage, router *chi.Mux) {
	router.Post("/update/{metricType}/{metricName}/{metricValue}", func(rw http.ResponseWriter, r *http.Request) {
		logRequest(r)
		defer func() { logResponse(http.StatusOK) }()

		metricType := chi.URLParam(r, "metricType")
		metricName := chi.URLParam(r, "metricName")
		metricValue := chi.URLParam(r, "metricValue")

		switch metricType {
		case "counter":
			value, err := strconv.ParseInt(metricValue, 10, 64)
			if err != nil {
				logResponse(http.StatusBadRequest)
				http.Error(rw, "Bad request", http.StatusBadRequest)
				return
			}
			storage.IncrementCounter(metricName, value)
		case "gauge":
			value, err := strconv.ParseFloat(metricValue, 64)
			if err != nil {
				logResponse(http.StatusBadRequest)
				http.Error(rw, "Bad request", http.StatusBadRequest)
				return
			}
			storage.SetGauge(metricName, value)
		default:
			logResponse(http.StatusBadRequest)
			http.Error(rw, "Bad request", http.StatusBadRequest)
			return
		}
	})
}

func GetHandler(storage *storage.MemStorage, router *chi.Mux) {
	router.Get("/value/{metricType}/{metricName}", func(rw http.ResponseWriter, r *http.Request) {
		logRequest(r)
		defer func() { logResponse(http.StatusOK) }()

		metricType := chi.URLParam(r, "metricType")
		metricName := chi.URLParam(r, "metricName")

		if metricType != "gauge" && metricType != "counter" {
			logResponse(http.StatusBadRequest)
			http.Error(rw, "Invalid metric type", http.StatusBadRequest)
			return
		}

		switch metricType {
		case "gauge":
			value, exists := storage.GetGauge(metricName)
			if !exists {
				logResponse(http.StatusNotFound)
				http.Error(rw, "Metric not found", http.StatusNotFound)
				return
			}
			rw.Header().Set("Content-type", "text/plain")
			rw.Write([]byte(strconv.FormatFloat(value, 'f', -1, 64)))
		case "counter":
			value, exists := storage.GetCounter(metricName)
			if !exists {
				logResponse(http.StatusNotFound)
				http.Error(rw, "Metric not found", http.StatusNotFound)
				return
			}
			rw.Header().Set("Content-type", "text/plain")
			rw.Write([]byte(strconv.FormatInt(value, 10)))
		default:
			logResponse(http.StatusBadRequest)
			http.Error(rw, "Invalid metric type", http.StatusBadRequest)
			return
		}
	})
}
