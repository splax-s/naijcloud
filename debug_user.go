package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

func main() {
	// Connect to the database
	db, err := sql.Open("postgres", "postgres://naijcloud:naijcloud123@localhost:5433/naijcloud?sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Test user ID
	userIDStr := "0e28a14a-787e-4bb3-b5ce-c725cec84ccd"
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		log.Fatal(err)
	}

	// Test the exact query
	query := `
		SELECT id, email, name, email_verified, avatar_url, 
		       COALESCE(settings::text, '{}'), created_at, updated_at
		FROM users 
		WHERE id = $1 AND deleted_at IS NULL
	`

	fmt.Printf("Testing query with user ID: %s\n", userID)

	var id uuid.UUID
	var email, name, avatarURL, settingsStr string
	var emailVerified bool
	var createdAt, updatedAt interface{}

	err = db.QueryRowContext(context.Background(), query, userID).Scan(
		&id, &email, &name, &emailVerified,
		&avatarURL, &settingsStr, &createdAt, &updatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Println("No user found")
		} else {
			fmt.Printf("Error: %v\n", err)
		}
		return
	}

	fmt.Printf("Found user: %s (%s)\n", email, name)
	fmt.Printf("Settings: %s\n", settingsStr)
}
