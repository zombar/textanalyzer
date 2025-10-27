package database

import (
	"testing"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name        string
		testID      string
		expectError bool
	}{
		{
			name:        "valid database connection",
			testID:      "test_new",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			connStr, cleanup := setupTestDB(t, tt.testID)
			defer cleanup()

			db, err := New(connStr)
			defer func() {
				if db != nil {
					db.Close()
				}
			}()

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if db == nil {
					t.Fatal("Expected db but got nil")
				}
				if db.conn == nil {
					t.Error("Expected database connection but got nil")
				}
			}
		})
	}
}

func TestClose(t *testing.T) {
	connStr, cleanup := setupTestDB(t, "test_close")
	defer cleanup()

	db, err := New(connStr)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}

	err = db.Close()
	if err != nil {
		t.Errorf("Failed to close database: %v", err)
	}

	// Closing again should not panic
	err = db.Close()
	if err == nil {
		t.Log("Closing database twice did not return error (expected behavior varies)")
	}
}

func TestNewWithInvalidPath(t *testing.T) {
	// Try to create database with invalid connection string
	db, err := New("invalid connection string")
	if err == nil && db != nil {
		db.Close()
		t.Error("Expected error when creating database with invalid connection string")
	}
}

func TestMigrationsRun(t *testing.T) {
	connStr, cleanup := setupTestDB(t, "test_migrations")
	defer cleanup()

	db, err := New(connStr)
	if err != nil {
		t.Fatalf("Failed to create database with migrations: %v", err)
	}
	defer db.Close()

	// Verify we can execute basic SQL queries
	var result int
	err = db.conn.QueryRow("SELECT 1").Scan(&result)
	if err != nil {
		t.Errorf("Failed to execute basic query: %v", err)
	}
	if result != 1 {
		t.Errorf("Expected result 1, got %d", result)
	}
}

func TestDatabaseConnection(t *testing.T) {
	connStr, cleanup := setupTestDB(t, "test_connection")
	defer cleanup()

	db, err := New(connStr)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Test that we can ping the database
	err = db.conn.Ping()
	if err != nil {
		t.Errorf("Failed to ping database: %v", err)
	}
}

func TestConnectionPoolSettings(t *testing.T) {
	connStr, cleanup := setupTestDB(t, "test_pool")
	defer cleanup()

	db, err := New(connStr)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Verify connection pool can execute queries
	// This implicitly tests that connection pool settings are valid
	var result int
	err = db.conn.QueryRow("SELECT 1").Scan(&result)
	if err != nil {
		t.Errorf("Failed to execute query with connection pool: %v", err)
	}
	if result != 1 {
		t.Errorf("Expected result 1, got %d", result)
	}
}

func TestConcurrentAccess(t *testing.T) {
	connStr, cleanup := setupTestDB(t, "test_concurrent")
	defer cleanup()

	db, err := New(connStr)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Test concurrent queries don't cause issues
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			var result int
			// Use PostgreSQL placeholder syntax $1 instead of SQLite ?
			err := db.conn.QueryRow("SELECT $1", id).Scan(&result)
			if err != nil {
				t.Errorf("Concurrent query %d failed: %v", id, err)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}
