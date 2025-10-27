package database

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

// DB represents the database connection
type DB struct {
	conn *sql.DB
}

// New creates a new PostgreSQL database connection
// PostgreSQL format: "host=... user=... password=... dbname=... port=..."
func New(connStr string) (*DB, error) {
	conn, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{conn: conn}, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.conn.Close()
}

// Conn returns the underlying database connection
func (db *DB) Conn() *sql.DB {
	return db.conn
}
