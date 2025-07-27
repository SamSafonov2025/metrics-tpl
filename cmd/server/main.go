package main

import (
	"log"
	"net/http"

	"github.com/SamSafonov2025/metrics-tpl/cmd/server/handlers"
	"github.com/SamSafonov2025/metrics-tpl/cmd/server/storage"
	"github.com/SamSafonov2025/metrics-tpl/internal/config"
	"github.com/go-chi/chi/v5"
)

func main() {
	cfg := config.ParseFlags()

	storage := storage.NewStorage()
	router := chi.NewRouter()

	// Add logging middleware
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Printf("Received %s request for %s", r.Method, r.URL.Path)
			next.ServeHTTP(w, r)
		})
	})

	handlers.HomeHandle(storage, router)
	handlers.UpdateHandler(storage, router)
	handlers.GetHandler(storage, router)

	log.Printf("Server is running on http://%s", cfg.ServerAddress)
	err := http.ListenAndServe(cfg.ServerAddress, router)
	if err != nil {
		log.Fatal("Server failed to start: ", err)
	}
}
