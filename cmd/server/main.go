package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/SamSafonov2025/metrics-tpl/cmd/server/handlers"
	"github.com/SamSafonov2025/metrics-tpl/cmd/server/storage"
	"github.com/go-chi/chi/v5"
)

func main() {
	// Define and parse command-line flags
	addr := flag.String("a", "localhost:8080", "HTTP server endpoint address")
	flag.Parse()

	// Check for unknown flags
	if flag.NArg() > 0 {
		fmt.Fprintf(os.Stderr, "Error: unknown flag(s): %v\n", flag.Args())
		os.Exit(1)
	}

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

	log.Printf("Server is running on http://%s", *addr)
	err := http.ListenAndServe(*addr, router)
	if err != nil {
		log.Fatal("Server failed to start: ", err)
	}
}
