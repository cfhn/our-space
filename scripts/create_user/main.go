package main

import (
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"

	pb "github.com/cfhn/our-space/ourspace-backend/proto"
	"github.com/cfhn/our-space/pkg/database"
	"github.com/cfhn/our-space/pkg/pwhash"
)

func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
	}
}

func run() error {
	dbURI := os.Getenv("DB_URI")
	if dbURI == "" {
		dbURI = "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
	}

	name := os.Getenv("NAME")
	if name == "" {
		name = "Initial User"
	}

	username := os.Getenv("LOGIN_USERNAME")
	if username == "" {
		username = "admin"
	}

	password := os.Getenv("LOGIN_PASSWORD")
	generatedPassword := false
	if password == "" {
		password = uuid.NewString()
		generatedPassword = true
	}

	hash, err := pwhash.Create(password)
	if err != nil {
		return err
	}

	db, err := database.Connect(database.Config{
		URI:          dbURI,
		MaxOpenConns: 8,
	})
	if err != nil {
		return err
	}

	userID := uuid.NewString()
	_, err = db.Exec(`insert into members (id, name, membership_start, age_category) values ($1, $2, $3, $4)`, userID, name, time.Now(), pb.AgeCategory_AGE_CATEGORY_ADULT)
	if err != nil {
		return err
	}

	_, err = db.Exec(`insert into members_auth (id, username, password_hash) values ($1, $2, $3)`, userID, username, hash)
	if err != nil {
		return err
	}

	fmt.Printf("Created user %q with id %s\n", username, userID)
	if generatedPassword {
		fmt.Println("Generated password:\n")
		fmt.Println("    ", password, "\n")
		fmt.Println("")
	}

	return nil
}
