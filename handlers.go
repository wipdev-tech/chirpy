// This is the main entry point for the application. It includes the routing
// code and the associated handlers
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
)

// reqUserData is used by handlers to decode user data from incoming HTTP
// requests
type reqUserData struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte("OK"))
	if err != nil {
		panic(err)
	}
}

func handleMetrics(w http.ResponseWriter, _ *http.Request) {
	html, err := os.ReadFile("static/admin/metrics/index.html")
	if err != nil {
		panic(err)
	}
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte(fmt.Sprintf(string(html), s.FileserverHits)))
	if err != nil {
		panic(err)
	}
}

func handleReset(w http.ResponseWriter, _ *http.Request) {
	s.FileserverHits = 0
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte("OK"))
	if err != nil {
		panic(err)
	}
}

func handleGetChirps(w http.ResponseWriter, r *http.Request) {
	authorIDParam := r.URL.Query().Get("author_id")
	sortParam := r.URL.Query().Get("sort")
	sortAsc := true
	if sortParam == "desc" {
		sortAsc = false
	}

	if authorIDParam == "" {
		chirps := s.GetChirps(sortAsc)
		w.WriteHeader(http.StatusOK)
		err := json.NewEncoder(w).Encode(chirps)
		if err != nil {
			panic(err)
		}
		return
	}

	authorID, err := strconv.Atoi(authorIDParam)
	if err != nil {
		panic(err)
	}

	chirps := s.GetChirpsByAuthor(authorID, sortAsc)
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(chirps)
	if err != nil {
		panic(err)
	}
}

func handleGetChirp(w http.ResponseWriter, r *http.Request) {
	chirpID := chi.URLParam(r, "chirpID")
	chirp, ok := s.GetChirp(chirpID)

	if ok {
		w.WriteHeader(http.StatusOK)
		err := json.NewEncoder(w).Encode(chirp)
		if err != nil {
			panic(err)
		}
		return
	}

	w.WriteHeader(http.StatusNotFound)
}

func handleCreateChirp(w http.ResponseWriter, r *http.Request) {
	type msg struct {
		Body string
	}
	inMsg := msg{}
	err := json.NewDecoder(r.Body).Decode(&inMsg)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	bearer := strings.Replace(r.Header.Get("Authorization"), "Bearer ", "", -1)
	authorID, err := s.AuthorizeUser(bearer)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if len(inMsg.Body) <= 140 {
		newChirp, err := s.CreateChirp(authorID, inMsg.Body)
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

func handleCreateUser(w http.ResponseWriter, r *http.Request) {
	type OutUsr struct {
		ID          int    `json:"id"`
		Email       string `json:"email"`
		IsChirpyRed bool   `json:"is_chirpy_red"`
	}

	inUsr := reqUserData{}
	err := json.NewDecoder(r.Body).Decode(&inUsr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	dbUser, err := s.CreateUser(inUsr.Email, inUsr.Password)
	if err != nil {
		fmt.Println("Error creating new user", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	outUsr := OutUsr{
		ID:          dbUser.ID,
		Email:       dbUser.Email,
		IsChirpyRed: dbUser.IsChirpyRed,
	}
	w.WriteHeader(http.StatusCreated)
	err = json.NewEncoder(w).Encode(outUsr)
	if err != nil {
		panic(err)
	}
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	inUsr := reqUserData{}
	err := json.NewDecoder(r.Body).Decode(&inUsr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	user, err := s.Login(inUsr.Email, inUsr.Password)
	if err != nil && err.Error() == "user doesn't exist" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if err != nil {
		fmt.Println("Error logging in:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(user)
	if err != nil {
		panic(err)
	}
}

func handleUpdateUser(w http.ResponseWriter, r *http.Request) {
	inUsr := reqUserData{}
	err := json.NewDecoder(r.Body).Decode(&inUsr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	bearer := strings.Replace(r.Header.Get("Authorization"), "Bearer ", "", -1)
	userID, err := s.AuthorizeUser(bearer)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	newUser, err := s.UpdateUser(userID, inUsr.Email, inUsr.Password)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = json.NewEncoder(w).Encode(newUser)
	if err != nil {
		panic(err)
	}
}

func handleRefresh(w http.ResponseWriter, r *http.Request) {
	bearer := strings.Replace(r.Header.Get("Authorization"), "Bearer ", "", -1)
	userID, err := s.AuthorizeRefresh(bearer)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	newAccess, err := s.Refresh(userID)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = json.NewEncoder(w).Encode(newAccess)
	if err != nil {
		panic(err)
	}
}

func handleRevoke(w http.ResponseWriter, r *http.Request) {
	bearer := strings.Replace(r.Header.Get("Authorization"), "Bearer ", "", -1)
	_, err := s.AuthorizeRefresh(bearer)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	err = s.Revoke(bearer)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func handleDeleteChirp(w http.ResponseWriter, r *http.Request) {
	bearer := strings.Replace(r.Header.Get("Authorization"), "Bearer ", "", -1)
	authorID, err := s.AuthorizeUser(bearer)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	chirpID := chi.URLParam(r, "chirpID")
	chirp, ok := s.GetChirp(chirpID)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if authorID != chirp.AuthorID {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	err = s.DeleteChirp(chirpID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func handlePolkaWebhook(w http.ResponseWriter, r *http.Request) {
	type InEvent struct {
		Event string `json:"event"`
		Data  struct {
			UserID int `json:"user_id"`
		} `json:"data"`
	}

	if r.Header.Get("Authorization") != "ApiKey "+os.Getenv("POLKA_KEY") {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	inEvent := InEvent{}
	err := json.NewDecoder(r.Body).Decode(&inEvent)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if inEvent.Event == "user.upgraded" {
		userID := inEvent.Data.UserID
		err := s.UpgradeChirpyRed(userID)
		if err != nil {
			fmt.Println(err)
		}
	}

	w.WriteHeader(http.StatusOK)
}
