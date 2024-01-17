// Package service contains the logic between handlers and models (DB). It
// should be the only package in the app with access to the DB connection
// through the Service struct.
package service

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/wipdev-tech/chirpy/internal/db"
	"golang.org/x/crypto/bcrypt"
)

type outUser struct {
	ID    int    `json:"id"`
	Email string `json:"email"`
}

type outUserWithToken struct {
	ID    int    `json:"id"`
	Email string `json:"email"`
	Token string `json:"token"`
}

// Service contains the app data (right now it's only the server hits and DB
// connection), middleware functions, business logic, and calls to the DB.
type Service struct {
	FileserverHits int
	dbConn         *db.DB
}

// MiddlewareMetricsInc wraps around app (user-facing) HTTP handlers to
// register the number of hits.
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

// MiddlewareCors wraps around the whole router to add CORS headers to the
// HTTP response
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

// InitDB initializes a new database and registers its connection into the
// service
func (s *Service) InitDB() {
	newDB, err := db.NewDB("database.json")
	if err != nil {
		panic(err)
	}
	s.dbConn = newDB
}

// GetChirp queries the database a chirp by its ID. It returns a chirp and
// boolean indicating whether the chirp was found (to be used in a comma-ok
// idiom).
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

// GetChirps queries the database for all chirps, returning them in a slice.
func (s *Service) GetChirps() []db.Chirp {
	chirps, err := s.dbConn.GetChirps()
	if err != nil {
		panic(err)
	}
	return chirps
}

// CreateChirp adds a new chirp to the database after cleaning profane words.
// Note that the 140-character validation happens at the handler level because
// it is considered a bad request to send a longer chirp.
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

// CreateUser adds a new user to the database after hashing the given password.
func (s *Service) CreateUser(email string, password string) (db.User, error) {
	users, err := s.dbConn.GetUsers()
	if err != nil {
		return db.User{}, err
	}

	for _, c := range users {
		if c.Email != email {
			return db.User{}, fmt.Errorf("Email %v already exists", email)
		}
	}

	hPassword, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	if err != nil {
		return db.User{}, err
	}

	return s.dbConn.CreateUser(email, string(hPassword))
}

// Login simply matches the email and password against the ones currently
// stored at the database. It returns the user, a boolean indicating whether it
// was found (to be used with a comma-ok idiom), and an error if it happened
// when calling the DB.
func (s *Service) Login(email string, password string, expiry int) (outUserWithToken, error) {
	var outUser outUserWithToken

	users, err := s.dbConn.GetUsers()
	if err != nil {
		fmt.Println("error getting users")
		return outUser, err
	}

	for _, u := range users {
		emailMatch := u.Email == email
		passMatch := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password)) == nil
		if emailMatch && passMatch {
			token := jwt.NewWithClaims(
				jwt.SigningMethodHS256,
				jwt.RegisteredClaims{
					Issuer:   "chirpy",
					IssuedAt: jwt.NewNumericDate(time.Now()),
					ExpiresAt: jwt.NewNumericDate(
						time.Now().Add(time.Duration(expiry) * time.Second),
					),
					Subject: fmt.Sprint(u.ID),
				},
			)

			jwtSecret := os.Getenv("JWT_SECRET")
			tokenStr, err := token.SignedString([]byte(jwtSecret))
			if err != nil {
				return outUser, err
			}

			outUser = outUserWithToken{
				ID:    u.ID,
				Email: u.Email,
				Token: tokenStr,
			}

			return outUser, nil
		}
	}

	return outUser, fmt.Errorf("user doesn't exist")
}

func (s *Service) AuthorizeUser(bearer string) (int, error) {
	claims := &jwt.RegisteredClaims{}
	keyfunc := func(toke *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("JWT_SECRET")), nil
	}
	token, err := jwt.ParseWithClaims(bearer, claims, keyfunc)
	if err != nil {
		return 0, err
	}

	userIdStr, err := token.Claims.GetSubject()
	if err != nil {
		return 0, err
	}

	userId, err := strconv.Atoi(userIdStr)
	if err != nil {
		return 0, err
	}

	return userId, err
}

func (s *Service) UpdateUser(id int, newEmail string, newPassword string) (outUser, error) {
	hNewPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), 10)
	if err != nil {
		return outUser{}, err
	}

	updatedUser, err := s.dbConn.UpdateUser(id, newEmail, string(hNewPassword))
	if err != nil {
		return outUser{}, err
	}

	out := outUser{
		ID:    updatedUser.ID,
		Email: updatedUser.Email,
	}

	return out, nil
}
