// Package db has the DB types and functionality
package db

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// DB is the database connection struct
type DB struct {
	path string
	mux  *sync.RWMutex
}

// dStruct is the struct representation of the database
type dStruct struct {
	Chirps        map[int]Chirp           `json:"chirps"`
	Users         map[int]User            `json:"users"`
	RevokedTokens map[string]RevokedToken `json:"revoked_tokens"`
}

// Chirp holds data associated with a chirp in the chirps database table
type Chirp struct {
	ID   int    `json:"id"`
	Body string `json:"body"`
}

// User holds data associated with a user in the users database table
type User struct {
	ID       int    `json:"id"`
	Email    string `json:"email"`
	Password string `json:"user"`
}

// RevokedToken holds data associated with a revoked token in the
// revoked_tokens database table
type RevokedToken struct {
	TokenStr  string    `json:"token"`
	RevokedAt time.Time `json:"revoked_at"`
}

// NewDB creates a new database connection
// and creates the database file if it doesn't exist
func NewDB(path string) (*DB, error) {
	fmt.Println("Making new DB...")
	newDB := &DB{path: path, mux: &sync.RWMutex{}}
	err := newDB.ensureDB()
	return newDB, err
}

// CreateChirp creates a new chirp and saves it to disk
func (db *DB) CreateChirp(body string) (Chirp, error) {
	fmt.Println("Creating chirp...")
	db.mux.Lock()
	defer db.mux.Unlock()

	newChirp := Chirp{}

	dbStr, err := db.loadDB()
	if err != nil {
		fmt.Println("Err loading DB...", err)
		return newChirp, err
	}

	id := 0
	for {
		id++
		if _, ok := dbStr.Chirps[id]; !ok {
			break
		}
	}

	newChirp.ID = id
	newChirp.Body = body

	dbStr.Chirps[id] = newChirp
	err = db.writeDB(dbStr)
	if err != nil {
		fmt.Printf("%v\n", err)
		return newChirp, err
	}

	return newChirp, nil
}

// CreateUser creates a new user and saves it to disk
func (db *DB) CreateUser(email string, hPassword string) (User, error) {
	fmt.Println("Creating user...")
	db.mux.Lock()
	defer db.mux.Unlock()

	newUser := User{}

	dbStr, err := db.loadDB()
	if err != nil {
		fmt.Println("Err loading DB...", err)
		return newUser, err
	}

	id := 0
	for {
		id++
		if _, ok := dbStr.Users[id]; !ok {
			break
		}
	}

	newUser.ID = id
	newUser.Email = email
	newUser.Password = hPassword

	dbStr.Users[id] = newUser
	err = db.writeDB(dbStr)
	if err != nil {
		fmt.Printf("%v\n", err)
		return newUser, err
	}

	return newUser, nil
}

// GetUsers returns all users in the database
func (db *DB) GetUsers() ([]User, error) {
	users := []User{}

	dbStruct, err := db.loadDB()
	if err != nil {
		return users, err
	}

	for _, u := range dbStruct.Users {
		users = append(users, u)
	}

	return users, err
}

// UpdateUser updates the data for the given ID with a new email and (hashed)
// password
func (db *DB) UpdateUser(id int, newEmail string, hNewPassword string) (User, error) {
	newUser := User{}

	dbStruct, err := db.loadDB()
	if err != nil {
		return newUser, err
	}

	for i, u := range dbStruct.Users {
		if u.ID == id {
			newUser = User{
				ID:       id,
				Email:    newEmail,
				Password: hNewPassword,
			}
			dbStruct.Users[i] = newUser
			break
		}
	}

	err = db.writeDB(dbStruct)
	return newUser, err
}

// GetChirps returns all chirps in the database
func (db *DB) GetChirps() ([]Chirp, error) {
	chirps := []Chirp{}

	dbStr, err := db.loadDB()
	if err != nil {
		return chirps, err
	}

	for _, c := range dbStr.Chirps {
		chirps = append(chirps, c)
	}

	return chirps, err
}

// ensureDB creates a new database file if it doesn't exist
func (db *DB) ensureDB() error {
	if _, err := os.Stat(db.path); os.IsNotExist(err) {
		_, err := os.Create(db.path)
		if err != nil {
			return err
		}
	}

	emptyDB, err := json.Marshal(
		dStruct{
			Chirps:        map[int]Chirp{},
			Users:         map[int]User{},
			RevokedTokens: map[string]RevokedToken{},
		},
	)
	if err != nil {
		return err
	}
	err = os.WriteFile(db.path, emptyDB, 0644)
	return err
}

// loadDB reads the database file into memory
func (db *DB) loadDB() (dStruct, error) {
	dbStr := dStruct{}

	dbBytes, err := os.ReadFile(db.path)
	if err != nil {
		return dbStr, err
	}

	err = json.Unmarshal(dbBytes, &dbStr)
	return dbStr, err
}

// writeDB writes the database file to disk
func (db *DB) writeDB(dbStr dStruct) error {
	fmt.Println("Saving to disk...")
	dbBytes, err := json.Marshal(dbStr)
	if err != nil {
		return err
	}

	err = os.WriteFile(db.path, dbBytes, 0644)
	if err != nil {
		return err
	}

	return nil
}

// AddRevokedToken adds the given token to the revoked tokens database table to
// prevent future use.
func (db *DB) AddRevokedToken(token string, revokedAt time.Time) error {
	db.mux.Lock()
	defer db.mux.Unlock()

	dbStr, err := db.loadDB()
	if err != nil {
		return err
	}

	for _, t := range dbStr.RevokedTokens {
		if t.TokenStr == token {
			return nil
		}
	}

	newRevokedToken := RevokedToken{
		TokenStr:  token,
		RevokedAt: revokedAt,
	}

	dbStr.RevokedTokens[token] = newRevokedToken
	err = db.writeDB(dbStr)
	return err
}

// GetRevokedTokens returns all revoked tokens in the database
func (db *DB) GetRevokedTokens() ([]RevokedToken, error) {
	tokens := []RevokedToken{}

	dbStruct, err := db.loadDB()
	if err != nil {
		return tokens, err
	}

	for _, t := range dbStruct.RevokedTokens {
		tokens = append(tokens, t)
	}

	return tokens, err
}
