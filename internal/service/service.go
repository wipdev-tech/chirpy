// package service contains the logic between handlers and models (DB)
package service

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/wipdev-tech/chirpy/internal/db"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	FileserverHits int
	dbConn         *db.DB
}

func (s *Service) MiddlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.FileserverHits++
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Service) MiddlewareCors(next http.Handler) http.Handler {
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

func (s *Service) InitDB() {
	newDB, err := db.NewDB("database.json")
	if err != nil {
		panic(err)
	}
	s.dbConn = newDB
}

func (s *Service) GetChirp(chirpID string) (db.Chirp, bool) {
	chirps, err := s.dbConn.GetChirps()
	if err != nil {
		panic(err)
	}
	for _, c := range chirps {
		if fmt.Sprintf("%d", c.ID) == chirpID {
			return c, true
		}
	}
	return db.Chirp{}, false
}

func (s *Service) GetChirps() []db.Chirp {
	chirps, err := s.dbConn.GetChirps()
	if err != nil {
		panic(err)
	}
	return chirps
}

func (s *Service) CreateChirp(body string) (db.Chirp, error) {
	inFields := strings.Fields(body)
	for i, f := range inFields {
		lower := strings.ToLower(f)
		if lower == "kerfuffle" || lower == "sharbert" || lower == "fornax" {
			inFields[i] = "****"
		}
	}
	cleaned := strings.Join(inFields, " ")

	return s.dbConn.CreateChirp(cleaned)
}

func (s *Service) CreateUser(email string, password string) (db.User, error) {
	hPassword, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	if err != nil {
		panic(err)
	}

	return s.dbConn.CreateUser(email, string(hPassword))
}

func (s *Service) Login(email string, password string) (db.User, bool, error) {
	var user db.User

	users, err := s.dbConn.GetUsers()
	if err != nil {
		fmt.Println("Error getting users")
		return user, false, err
	}

	for _, u := range users {
		emailMatch := u.Email == email
		passwordMatch := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
		if emailMatch && passwordMatch == nil {
			return u, true, nil
		}
	}

	return user, false, nil
}
