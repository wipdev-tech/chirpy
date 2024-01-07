package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/wipdev-tech/chirpy/internal/db"
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

var chirpsDB *db.DB

func main() {
    newDB, err := db.NewDB("database.json")
	if err != nil {
		panic(err)
	}
    chirpsDB = newDB

	appFS := http.FileServer(http.Dir("."))

	apiRouter := chi.NewRouter()
	apiRouter.Get("/healthz", handleHealth)
	apiRouter.HandleFunc("/reset", handleReset)
	apiRouter.Post("/chirps", handleNewChirp)
	apiRouter.Get("/chirps", handleGetChirps)
	apiRouter.Get("/chirps/{chirpID}", handleGetChirp)
	apiRouter.Post("/users", handleNewUser)

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
	panic(s.ListenAndServe())
}
