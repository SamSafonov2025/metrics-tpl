package main

import (
	"fmt"
	"github.com/SamSafonov2025/metrics-tpl.git/cmd/server/metrics"
	"github.com/SamSafonov2025/metrics-tpl.git/cmd/server/storage"
	"net/http"
	"strconv"
	"strings"
)

func main() {
	storage := storage.NewStorage()
	fmt.Println("Server is running on http://localhost:8080")
	mux := http.NewServeMux()
	mux.HandleFunc("/update/", updateHandler(storage))
	err := http.ListenAndServe(":8080", mux)
	if err != nil {
		panic(err)
	}
}

func badRequest(w http.ResponseWriter) {
	http.Error(w, "Bad request", http.StatusBadRequest)
}

func updateHandler(storage *storage.MemStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost {
			http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
			return
		}

		path := strings.Split(req.URL.Path, "/")
		fmt.Println(path)
		if len(path) != 5 {
			http.Error(w, "Invalid URL format", http.StatusNotFound)
			return
		}
		if req.Header.Get("Content-Type") != "text/plain" {
			http.Error(w, "Invalid data format", http.StatusNotFound)
		}

		metricType, metricName, metricValue := path[2], path[3], path[4]

		switch metricType {
		case "counter":
			pathValue, err := strconv.ParseInt(metricValue, 10, 64)
			if err != nil {
				badRequest(w)
				return
			}
			storage.UpdateCounter(metricName, metrics.Counter(pathValue))
			w.WriteHeader(http.StatusOK)
		case "gauge":
			pathValue, err := strconv.ParseFloat(metricValue, 64)
			if err != nil {
				badRequest(w)
				return
			}
			storage.UpdateGuage(metricName, metrics.Gauge(pathValue))
		default:
			badRequest(w)
		}
	}
}
