package database

import (
	"fmt"
)

// Migration represents a database migration
type Migration struct {
	Version int
	Name    string
	SQL     string
}

// migrations contains all database migrations in order
var migrations = []Migration{
	{
		Version: 1,
		Name:    "create_analyses_table",
		SQL: `
			CREATE TABLE IF NOT EXISTS analyses (
				id TEXT PRIMARY KEY,
				text TEXT NOT NULL,
				metadata TEXT NOT NULL,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
			);
			CREATE INDEX IF NOT EXISTS idx_analyses_created_at ON analyses(created_at);
		`,
	},
	{
		Version: 2,
		Name:    "create_tags_table",
		SQL: `
			CREATE TABLE IF NOT EXISTS tags (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				analysis_id TEXT NOT NULL,
				tag TEXT NOT NULL,
				FOREIGN KEY (analysis_id) REFERENCES analyses(id) ON DELETE CASCADE
			);
			CREATE INDEX IF NOT EXISTS idx_tags_analysis_id ON tags(analysis_id);
			CREATE INDEX IF NOT EXISTS idx_tags_tag ON tags(tag);
		`,
	},
	{
		Version: 3,
		Name:    "create_schema_version_table",
		SQL: `
			CREATE TABLE IF NOT EXISTS schema_version (
				version INTEGER PRIMARY KEY,
				applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
			);
		`,
	},
	{
		Version: 4,
		Name:    "create_text_references_table",
		SQL: `
			CREATE TABLE IF NOT EXISTS text_references (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				analysis_id TEXT NOT NULL,
				text TEXT NOT NULL,
				type TEXT NOT NULL,
				context TEXT,
				confidence TEXT,
				FOREIGN KEY (analysis_id) REFERENCES analyses(id) ON DELETE CASCADE
			);
			CREATE INDEX IF NOT EXISTS idx_text_references_analysis_id ON text_references(analysis_id);
			CREATE INDEX IF NOT EXISTS idx_text_references_text ON text_references(text);
			CREATE INDEX IF NOT EXISTS idx_text_references_type ON text_references(type);
		`,
	},
}

// Migrate runs all pending migrations
func (db *DB) Migrate() error {
	// Ensure schema_version table exists
	if _, err := db.conn.Exec(migrations[2].SQL); err != nil {
		return fmt.Errorf("failed to create schema_version table: %w", err)
	}

	// Get current version
	var currentVersion int
	err := db.conn.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_version").Scan(&currentVersion)
	if err != nil {
		return fmt.Errorf("failed to get current version: %w", err)
	}

	// Run pending migrations
	for _, migration := range migrations {
		if migration.Version <= currentVersion {
			continue
		}

		if _, err := db.conn.Exec(migration.SQL); err != nil {
			return fmt.Errorf("failed to run migration %d (%s): %w", migration.Version, migration.Name, err)
		}

		if _, err := db.conn.Exec("INSERT INTO schema_version (version) VALUES (?)", migration.Version); err != nil {
			return fmt.Errorf("failed to record migration %d: %w", migration.Version, err)
		}

		fmt.Printf("Applied migration %d: %s\n", migration.Version, migration.Name)
	}

	return nil
}
