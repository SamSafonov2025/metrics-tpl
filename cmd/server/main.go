package main

import (
	"log"
	"net/http"

	"github.com/SamSafonov2025/metrics-tpl.git/cmd/server/handlers"
	"github.com/SamSafonov2025/metrics-tpl.git/cmd/server/storage"
	"github.com/go-chi/chi/v5"
)

func main() {
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

	log.Println("Server is running on http://localhost:8080")
	err := http.ListenAndServe("localhost:8080", router)
	if err != nil {
		log.Fatal("Server failed to start: ", err)
	}
}
