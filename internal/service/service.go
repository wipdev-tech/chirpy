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

// ResUserData holds user data to be used by handlers in HTTP responses
type ResUserData struct {
	ID    int    `json:"id"`
	Email string `json:"email"`
}

// ResUserDataT embeds resUserData with the addition of access and refresh JWTS
type ResUserDataT struct {
	ResUserData
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
}

// ResRefresh holds only a new access JWT generated after a successful refresh
type ResRefresh struct {
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
func (s *Service) CreateChirp(authorID int, body string) (db.Chirp, error) {
	inFields := strings.Fields(body)
	for i, f := range inFields {
		lower := strings.ToLower(f)
		if lower == "kerfuffle" || lower == "sharbert" || lower == "fornax" {
			inFields[i] = "****"
		}
	}
	cleaned := strings.Join(inFields, " ")

	return s.dbConn.CreateChirp(authorID, cleaned)
}

// CreateUser adds a new user to the database after hashing the given password.
func (s *Service) CreateUser(email string, password string) (db.User, error) {
	users, err := s.dbConn.GetUsers()
	if err != nil {
		return db.User{}, err
	}

	for _, c := range users {
		if c.Email != email {
			return db.User{}, fmt.Errorf("email %v already exists", email)
		}
	}

	hPassword, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	if err != nil {
		return db.User{}, err
	}

	return s.dbConn.CreateUser(email, string(hPassword))
}

// Login simply matches the email and password against the ones currently
// stored at the database. It returns the the user data with access and refresh
// JWTs.
func (s *Service) Login(email string, password string) (ResUserDataT, error) {
	var outUser ResUserDataT

	users, err := s.dbConn.GetUsers()
	if err != nil {
		fmt.Println("error getting users")
		return outUser, err
	}

	for _, u := range users {
		emailMatch := u.Email == email
		passMatch := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password)) == nil
		if emailMatch && passMatch {
			accessStr, err := generateAccess(u.ID)
			if err != nil {
				return outUser, err
			}

			refreshStr, err := generateRefresh(u.ID)
			if err != nil {
				return outUser, err
			}

			outUser.ID = u.ID
			outUser.Email = u.Email
			outUser.Token = accessStr
			outUser.RefreshToken = refreshStr
			return outUser, nil
		}
	}

	return outUser, fmt.Errorf("user doesn't exist")
}

func generateAccess(userID int) (accessStr string, err error) {
	jwtSecret := os.Getenv("JWT_SECRET")

	access := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		jwt.RegisteredClaims{
			Issuer:   "chirpy-access",
			IssuedAt: jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(
				time.Now().Add(1 * time.Hour),
			),
			Subject: fmt.Sprint(userID),
		},
	)

	accessStr, err = access.SignedString([]byte(jwtSecret))
	if err != nil {
		return "", fmt.Errorf("couldn't sign access token: %v", err)
	}

	return accessStr, err
}

func generateRefresh(userID int) (refreshStr string, err error) {
	jwtSecret := os.Getenv("JWT_SECRET")

	refresh := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		jwt.RegisteredClaims{
			Issuer:   "chirpy-refresh",
			IssuedAt: jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(
				time.Now().Add(60 * 24 * time.Hour),
			),
			Subject: fmt.Sprint(userID),
		},
	)

	refreshStr, err = refresh.SignedString([]byte(jwtSecret))
	if err != nil {
		return "", fmt.Errorf("couldn't sign refresh token: %v", err)
	}

	return refreshStr, err
}

// AuthorizeUser takes a bearer token and returns the integer ID of the user
// that owns the token
func (s *Service) AuthorizeUser(bearer string) (int, error) {
	claims := &jwt.RegisteredClaims{}
	keyfunc := func(toke *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("JWT_SECRET")), nil
	}
	token, err := jwt.ParseWithClaims(bearer, claims, keyfunc)
	if err != nil {
		return 0, err
	}

	issuer, err := token.Claims.GetIssuer()
	if err != nil {
		return 0, err
	}
	if issuer != "chirpy-access" {
		return 0, fmt.Errorf("wrong issuer")
	}

	userIDStr, err := token.Claims.GetSubject()
	if err != nil {
		return 0, err
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		return 0, err
	}

	return userID, err
}

// UpdateUser updates the email and password of the user whose ID is provided
// in the first argument
func (s *Service) UpdateUser(id int, newEmail string, newPassword string) (ResUserData, error) {
	hNewPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), 10)
	if err != nil {
		return ResUserData{}, err
	}

	updatedUser, err := s.dbConn.UpdateUser(id, newEmail, string(hNewPassword))
	if err != nil {
		return ResUserData{}, err
	}

	out := ResUserData{
		ID:    updatedUser.ID,
		Email: updatedUser.Email,
	}

	return out, nil
}

// AuthorizeRefresh checks if a refresh token is valid, which means it is 1)
// not revoked 2) a valid JWT and 3) issued as a refresh token.
func (s *Service) AuthorizeRefresh(bearer string) (userID int, err error) {
	revokedTokens, err := s.dbConn.GetRevokedTokens()
	if err != nil {
		return 0, err
	}
	for _, rt := range revokedTokens {
		if rt.TokenStr == bearer {
			return 0, fmt.Errorf("revoked token")
		}
	}

	claims := &jwt.RegisteredClaims{}
	keyfunc := func(toke *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("JWT_SECRET")), nil
	}
	token, err := jwt.ParseWithClaims(bearer, claims, keyfunc)
	if err != nil {
		return 0, err
	}

	issuer, err := token.Claims.GetIssuer()
	if err != nil {
		return 0, err
	}
	if issuer != "chirpy-refresh" {
		return 0, fmt.Errorf("wrong issuer")
	}

	userIDStr, err := token.Claims.GetSubject()
	if err != nil {
		return 0, err
	}

	userID, err = strconv.Atoi(userIDStr)
	if err != nil {
		return 0, err
	}

	return userID, err
}

// Refresh generates a new access token for the given user ID
func (s *Service) Refresh(userID int) (ResRefresh, error) {
	newAccessStr, err := generateAccess(userID)
	if err != nil {
		return ResRefresh{}, err
	}

	newAccess := ResRefresh{Token: newAccessStr}
	return newAccess, err
}

// Revoke stores the given bearer token in the database
func (s *Service) Revoke(bearer string) error {
	err := s.dbConn.AddRevokedToken(bearer, time.Now())
	return err
}
