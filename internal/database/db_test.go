package database

import (
	"os"
	"testing"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name        string
		dbPath      string
		expectError bool
	}{
		{
			name:        "valid database path",
			dbPath:      "test_new.db",
			expectError: false,
		},
		{
			name:        "memory database",
			dbPath:      ":memory:",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, err := New(tt.dbPath)

			// Cleanup
			defer func() {
				if db != nil {
					db.Close()
				}
				if tt.dbPath != ":memory:" {
					os.Remove(tt.dbPath)
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
	db, err := New(":memory:")
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
	// Try to create database in non-existent directory
	db, err := New("/nonexistent/directory/test.db")
	if err == nil && db != nil {
		db.Close()
		t.Error("Expected error when creating database in non-existent directory")
	}
}

func TestMigrationsRun(t *testing.T) {
	// This test verifies that database can be created successfully
	// The migrations are tested implicitly by the queries package tests
	dbPath := "test_migrations.db"
	defer os.Remove(dbPath)

	db, err := New(dbPath)
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
	db, err := New(":memory:")
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
	db, err := New(":memory:")
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
	db, err := New(":memory:")
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Test concurrent queries don't cause issues
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			var result int
			err := db.conn.QueryRow("SELECT ?", id).Scan(&result)
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
