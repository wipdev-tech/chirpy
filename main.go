package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
	"github.com/wipdev-tech/chirpy/internal/service"
)

var s service.Service

func main() {
	godotenv.Load()
	s.InitDB()

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
	appRouter.Handle("/app/*", s.MiddlewareMetricsInc(http.StripPrefix("/app/", appFS)))
	appRouter.Handle("/app", s.MiddlewareMetricsInc(http.StripPrefix("/app", appFS)))
	appRouter.Mount("/api", apiRouter)
	appRouter.Mount("/admin", adminRouter)

	corsMux := s.MiddlewareCors(appRouter)

	s := http.Server{
		Addr:    ":8080",
		Handler: corsMux,
	}
	panic(s.ListenAndServe())
}
