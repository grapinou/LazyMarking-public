package db

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

func InitDB(dbPath string) (*sql.DB, error) {
	conn, err := sql.Open("sqlite3", sqliteDSN(dbPath))
	if err != nil {
		return nil, fmt.Errorf("failed to open db : %w", err)
	}
	log.Println("DB opened")

	// testing connexion
	if err := conn.Ping(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to ping db : %w", err)
	}
	log.Println("Connexion ok")

	return conn, nil
}

func sqliteDSN(dbPath string) string {
	if strings.Contains(dbPath, "_foreign_keys=") || strings.Contains(dbPath, "_fk=") {
		return dbPath
	}
	separator := "?"
	if strings.Contains(dbPath, "?") {
		separator = "&"
	}
	return dbPath + separator + "_foreign_keys=on"
}
