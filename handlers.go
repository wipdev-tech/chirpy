package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi/v5"
)

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte("OK"))
	if err != nil {
		panic(err)
	}
}

func handleMetrics(w http.ResponseWriter, r *http.Request) {
	html, err := os.ReadFile("admin/metrics/index.html")
	if err != nil {
		panic(err)
	}
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte(fmt.Sprintf(string(html), cfg.fileserverHits)))
	if err != nil {
		panic(err)
	}
}

func handleReset(w http.ResponseWriter, r *http.Request) {
	cfg.fileserverHits = 0
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte("OK"))
	if err != nil {
		panic(err)
	}
}

func handleGetChirps(w http.ResponseWriter, r *http.Request) {
	chirps, err := chirpsDB.GetChirps()
	if err != nil {
		panic(err)
	}

	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(chirps)
	if err != nil {
		panic(err)
	}
}

func handleGetChirp(w http.ResponseWriter, r *http.Request) {
	chirps, err := chirpsDB.GetChirps()
	if err != nil {
		panic(err)
	}

	chirpID := chi.URLParam(r, "chirpID")
	fmt.Println(chirpID)
	for _, c := range chirps {
		if fmt.Sprintf("%d", c.ID) == chirpID {
			w.WriteHeader(http.StatusOK)
			err = json.NewEncoder(w).Encode(c)
			if err != nil {
				panic(err)
			}
			return
		}
	}

	w.WriteHeader(http.StatusNotFound)
}

func handleNewChirp(w http.ResponseWriter, r *http.Request) {
	type msg struct {
		Body string
	}

	inMsg := msg{}
	err := json.NewDecoder(r.Body).Decode(&inMsg)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
	}

	if len(inMsg.Body) <= 140 {
		inFields := strings.Fields(inMsg.Body)
		for i, f := range inFields {
			lower := strings.ToLower(f)
			if lower == "kerfuffle" || lower == "sharbert" || lower == "fornax" {
				inFields[i] = "****"
			}
		}
		cleaned := strings.Join(inFields, " ")

		newChirp, err := chirpsDB.CreateChirp(cleaned)
		if err != nil {
			fmt.Println("Error creating new chirp")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		err = json.NewEncoder(w).Encode(newChirp)
		if err != nil {
			panic(err)
		}
		return
	}

	w.WriteHeader(http.StatusBadRequest)
}

func handleNewUser(w http.ResponseWriter, r *http.Request) {
	type usr struct {
		Email string
	}

	inUsr := usr{}
	err := json.NewDecoder(r.Body).Decode(&inUsr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
	}

	newUser, err := chirpsDB.CreateUser(inUsr.Email)
	if err != nil {
		fmt.Println("Error creating new user")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	err = json.NewEncoder(w).Encode(newUser)
	if err != nil {
		panic(err)
	}
	return
}
