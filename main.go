package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
	api "github.com/wipdev-tech/chirpy/internal/apiconfig"
	"github.com/wipdev-tech/chirpy/internal/db"
)

var cfg api.Config
var chirpsDB *db.DB

func main() {
	godotenv.Load()

	newDB, err := db.NewDB("database.json")
	if err != nil {
		panic(err)
	}
	chirpsDB = newDB

	appFS := http.FileServer(http.Dir("./static"))

	apiRouter := chi.NewRouter()
	apiRouter.Get("/healthz", handleHealth)
	apiRouter.HandleFunc("/reset", handleReset)
	apiRouter.Post("/chirps", handleNewChirp)
	apiRouter.Get("/chirps", handleGetChirps)
	apiRouter.Get("/chirps/{chirpID}", handleGetChirp)
	apiRouter.Post("/users", handleNewUser)
	apiRouter.Post("/login", handleLogin)

	adminRouter := chi.NewRouter()
	adminRouter.Get("/metrics", handleMetrics)
	adminRouter.Get("/metrics/", handleMetrics)

	appRouter := chi.NewRouter()
	appRouter.Handle("/app/*", cfg.MiddlewareMetricsInc(http.StripPrefix("/app/", appFS)))
	appRouter.Handle("/app", cfg.MiddlewareMetricsInc(http.StripPrefix("/app", appFS)))
	appRouter.Mount("/api", apiRouter)
	appRouter.Mount("/admin", adminRouter)

	corsMux := api.MiddlewareCors(appRouter)

	s := http.Server{
		Addr:    ":8080",
		Handler: corsMux,
	}
	panic(s.ListenAndServe())
}
