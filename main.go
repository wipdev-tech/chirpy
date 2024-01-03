package main

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type apiConfig struct {
	fileserverHits int
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits++
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

var cfg apiConfig

func main() {

	fs := http.FileServer(http.Dir("."))

	// mux := http.NewServeMux()
	r := chi.NewRouter()
	r.Handle("/app/*", cfg.middlewareMetricsInc(http.StripPrefix("/app/", fs)))
	r.Handle("/app", cfg.middlewareMetricsInc(http.StripPrefix("/app", fs)))
	r.Get("/healthz", handleHealth)
	r.Get("/metrics", handleMetrics)
	r.HandleFunc("/reset", handleReset)

	corsMux := middlewareCors(r)

	s := http.Server{
		Addr:    ":8080",
		Handler: corsMux,
	}
	s.ListenAndServe()
}

func middlewareCors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func handleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("Hits: %v", cfg.fileserverHits)))
}

func handleReset(w http.ResponseWriter, r *http.Request) {
	cfg.fileserverHits = 0
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
