package db

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

func InitDB(dbPath string) (*sql.DB, error) {
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open db : %w", err)
	}
	log.Println("DB opened")

	// testing connexion
	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping db : %w", err)
	}
	log.Println("Connexion ok")

	// activer les foreign keys
	if _, err := conn.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}
	log.Println("PRAGMA foreign_keys = ON")

	return conn, nil
}
