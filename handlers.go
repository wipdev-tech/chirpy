// This is the main entry point for the application. It includes the routing
// code and the associated handlers
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/bcrypt"
)

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

func handleGetChirps(w http.ResponseWriter, _ *http.Request) {
	chirps := s.GetChirps()
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(chirps)
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

func handleNewChirp(w http.ResponseWriter, r *http.Request) {
	type msg struct {
		Body string
	}

	inMsg := msg{}
	err := json.NewDecoder(r.Body).Decode(&inMsg)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if len(inMsg.Body) <= 140 {
		newChirp, err := s.CreateChirp(inMsg.Body)
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
	type InUsr struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	type OutUsr struct {
		ID    int    `json:"id"`
		Email string `json:"email"`
	}

	inUsr := InUsr{}
	err := json.NewDecoder(r.Body).Decode(&inUsr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	fmt.Printf("Hashing new password: `%s`\n", inUsr.Password)
	hPassword, err := bcrypt.GenerateFromPassword([]byte(inUsr.Password), 10)
	if err != nil {
		panic(err)
	}
	fmt.Println("Hashed as:", string(hPassword))

	dbUser, err := chirpsDB.CreateUser(inUsr.Email, string(hPassword))
	if err != nil {
		fmt.Println("Error creating new user")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	outUsr := OutUsr{ID: dbUser.ID, Email: dbUser.Email}
	w.WriteHeader(http.StatusCreated)
	err = json.NewEncoder(w).Encode(outUsr)
	if err != nil {
		panic(err)
	}
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	type InUsr struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	type OutUsr struct {
		ID    int    `json:"id"`
		Email string `json:"email"`
	}

	inUsr := InUsr{}
	err := json.NewDecoder(r.Body).Decode(&inUsr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	users, err := chirpsDB.GetUsers()
	if err != nil {
		fmt.Println("Error getting users")
		w.WriteHeader(http.StatusInternalServerError)
	}
	fmt.Println(len(users))

	for _, u := range users {
		fmt.Printf("In-DB PW for user %s is %s\n", u.Email, u.Password)
		fmt.Println("Checking:", inUsr.Password)
		emailMatch := u.Email == inUsr.Email
		passwordMatch := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(inUsr.Password))
		if emailMatch && passwordMatch == nil {
			fmt.Println("match!!!")
			outUsr := OutUsr{ID: u.ID, Email: u.Email}
			w.WriteHeader(http.StatusOK)
			err = json.NewEncoder(w).Encode(outUsr)
			if err != nil {
				panic(err)
			}
			return
		}
	}

	w.WriteHeader(http.StatusUnauthorized)
}
