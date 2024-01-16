package db

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

type DB struct {
	path string
	mux  *sync.RWMutex
}

type DBStructure struct {
	Chirps map[int]Chirp `json:"chirps"`
	Users  map[int]User  `json:"users"`
}

type Chirp struct {
	ID   int    `json:"id"`
	Body string `json:"body"`
}

type User struct {
	ID       int    `json:"id"`
	Email    string `json:"email"`
	Password string `json:"user"`
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

	dbStr, err := db.loadDB()
	if err != nil {
		return users, err
	}

	for _, u := range dbStr.Users {
		users = append(users, u)
	}

	return users, err
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
		DBStructure{
			Chirps: map[int]Chirp{},
			Users:  map[int]User{},
		},
	)
	if err != nil {
		return err
	}
	err = os.WriteFile(db.path, emptyDB, 0644)
	return err
}

// loadDB reads the database file into memory
func (db *DB) loadDB() (DBStructure, error) {
	dbStr := DBStructure{}

	dbBytes, err := os.ReadFile(db.path)
	if err != nil {
		return dbStr, err
	}

	err = json.Unmarshal(dbBytes, &dbStr)
	return dbStr, err
}

// writeDB writes the database file to disk
func (db *DB) writeDB(dbStr DBStructure) error {
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
