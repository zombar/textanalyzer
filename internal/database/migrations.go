package database

import (
	"fmt"
	"log"
)

// Migration represents a database migration
type Migration struct {
	Version int
	Name    string
	SQL     string
}

// migrations contains all PostgreSQL database migrations in order
var migrations = []Migration{
	{
		Version: 1,
		Name:    "create_analyses_table",
		SQL: `
			CREATE TABLE IF NOT EXISTS analyses (
				id TEXT PRIMARY KEY,
				text TEXT NOT NULL,
				metadata JSONB NOT NULL,
				created_at TIMESTAMPTZ DEFAULT NOW(),
				updated_at TIMESTAMPTZ DEFAULT NOW()
			);
			CREATE INDEX IF NOT EXISTS idx_analyses_created_at ON analyses(created_at);
		`,
	},
	{
		Version: 2,
		Name:    "create_tags_table",
		SQL: `
			CREATE TABLE IF NOT EXISTS tags (
				id SERIAL PRIMARY KEY,
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
				applied_at TIMESTAMPTZ DEFAULT NOW()
			);
		`,
	},
	{
		Version: 4,
		Name:    "create_text_references_table",
		SQL: `
			CREATE TABLE IF NOT EXISTS text_references (
				id SERIAL PRIMARY KEY,
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
	{
		Version: 5,
		Name:    "add_job_tracking_columns",
		SQL: `
			ALTER TABLE analyses ADD COLUMN IF NOT EXISTS processing_stage TEXT DEFAULT 'offline';
			ALTER TABLE analyses ADD COLUMN IF NOT EXISTS enqueued_at TIMESTAMPTZ;
			ALTER TABLE analyses ADD COLUMN IF NOT EXISTS started_at TIMESTAMPTZ;
			ALTER TABLE analyses ADD COLUMN IF NOT EXISTS completed_at TIMESTAMPTZ;
			ALTER TABLE analyses ADD COLUMN IF NOT EXISTS retry_count INTEGER DEFAULT 0;
			ALTER TABLE analyses ADD COLUMN IF NOT EXISTS max_retries INTEGER DEFAULT 10;
			ALTER TABLE analyses ADD COLUMN IF NOT EXISTS last_error TEXT;
			CREATE INDEX IF NOT EXISTS idx_analyses_processing_stage ON analyses(processing_stage);
			CREATE INDEX IF NOT EXISTS idx_analyses_enqueued_at ON analyses(enqueued_at);
		`,
	},
	{
		Version: 6,
		Name:    "add_original_html_column",
		SQL: `
			ALTER TABLE analyses ADD COLUMN IF NOT EXISTS original_html TEXT;
		`,
	},
}

// Migrate runs all pending PostgreSQL migrations
func (db *DB) Migrate() error {
	log.Println("Creating schema_version table...")
	// Ensure schema_version table exists
	if _, err := db.conn.Exec(migrations[2].SQL); err != nil {
		return fmt.Errorf("failed to create schema_version table: %w", err)
	}

	log.Println("Checking current schema version...")
	// Get current version
	var currentVersion int
	err := db.conn.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_version").Scan(&currentVersion)
	if err != nil {
		return fmt.Errorf("failed to get current version: %w", err)
	}
	log.Printf("Current schema version: %d", currentVersion)

	// Run pending migrations
	for _, migration := range migrations {
		if migration.Version <= currentVersion {
			log.Printf("Skipping migration %d (already applied)", migration.Version)
			continue
		}

		log.Printf("Applying migration %d: %s", migration.Version, migration.Name)
		tx, err := db.conn.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction for migration %d: %w", migration.Version, err)
		}

		if _, err := tx.Exec(migration.SQL); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to run migration %d (%s): %w", migration.Version, migration.Name, err)
		}

		// Use PostgreSQL $1 placeholder instead of ?
		if _, err := tx.Exec("INSERT INTO schema_version (version) VALUES ($1)", migration.Version); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to record migration %d: %w", migration.Version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration %d: %w", migration.Version, err)
		}

		log.Printf("✓ Applied migration %d: %s", migration.Version, migration.Name)
	}

	log.Println("All migrations complete")
	return nil
}
