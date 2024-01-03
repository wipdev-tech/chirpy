package main

import (
	"fmt"
	"net/http"
	"os"

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
	appFS := http.FileServer(http.Dir("."))

	apiRouter := chi.NewRouter()
	apiRouter.Get("/healthz", handleHealth)
	apiRouter.HandleFunc("/reset", handleReset)

	adminRouter := chi.NewRouter()
	adminRouter.Get("/metrics", handleMetrics)

	appRouter := chi.NewRouter()
	appRouter.Handle("/app/*", cfg.middlewareMetricsInc(http.StripPrefix("/app/", appFS)))
	appRouter.Handle("/app", cfg.middlewareMetricsInc(http.StripPrefix("/app", appFS)))
	appRouter.Mount("/api", apiRouter)
	appRouter.Mount("/admin", adminRouter)

	corsMux := middlewareCors(appRouter)

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
	html, err := os.ReadFile("admin/metrics/index.html")
	if err != nil {
		panic(err)
	}
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf(string(html), cfg.fileserverHits)))
}

func handleReset(w http.ResponseWriter, r *http.Request) {
	cfg.fileserverHits = 0
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
