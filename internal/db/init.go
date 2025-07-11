package db

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

func InitDB(dbPath string) (*sql.DB, error) {
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("Failed to open db : %w", err)
	}

	// testing connexion
	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("Failed to ping db : %w", err)
	}

	return conn, nil
}
