package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/yourusername/textanalyzer/internal/models"
)

// SaveAnalysis saves an analysis to the database
func (db *DB) SaveAnalysis(analysis *models.Analysis) error {
	metadataJSON, err := json.Marshal(analysis.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert analysis
	_, err = tx.Exec(`
		INSERT INTO analyses (id, text, metadata, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
	`, analysis.ID, analysis.Text, metadataJSON, analysis.CreatedAt, analysis.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to insert analysis: %w", err)
	}

	// Insert tags
	for _, tag := range analysis.Metadata.Tags {
		_, err = tx.Exec(`
			INSERT INTO tags (analysis_id, tag)
			VALUES (?, ?)
		`, analysis.ID, tag)
		if err != nil {
			return fmt.Errorf("failed to insert tag: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetAnalysis retrieves an analysis by ID
func (db *DB) GetAnalysis(id string) (*models.Analysis, error) {
	var (
		text         string
		metadataJSON string
		createdAt    time.Time
		updatedAt    time.Time
	)

	err := db.conn.QueryRow(`
		SELECT text, metadata, created_at, updated_at
		FROM analyses
		WHERE id = ?
	`, id).Scan(&text, &metadataJSON, &createdAt, &updatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("analysis not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get analysis: %w", err)
	}

	var metadata models.Metadata
	if err := json.Unmarshal([]byte(metadataJSON), &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return &models.Analysis{
		ID:        id,
		Text:      text,
		Metadata:  metadata,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}, nil
}

// GetAnalysesByTag retrieves all analyses with a specific tag
func (db *DB) GetAnalysesByTag(tag string) ([]*models.Analysis, error) {
	rows, err := db.conn.Query(`
		SELECT DISTINCT a.id, a.text, a.metadata, a.created_at, a.updated_at
		FROM analyses a
		INNER JOIN tags t ON a.id = t.analysis_id
		WHERE t.tag = ?
		ORDER BY a.created_at DESC
	`, tag)
	if err != nil {
		return nil, fmt.Errorf("failed to query analyses by tag: %w", err)
	}
	defer rows.Close()

	var analyses []*models.Analysis
	for rows.Next() {
		var (
			id           string
			text         string
			metadataJSON string
			createdAt    time.Time
			updatedAt    time.Time
		)

		if err := rows.Scan(&id, &text, &metadataJSON, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		var metadata models.Metadata
		if err := json.Unmarshal([]byte(metadataJSON), &metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}

		analyses = append(analyses, &models.Analysis{
			ID:        id,
			Text:      text,
			Metadata:  metadata,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return analyses, nil
}

// ListAnalyses retrieves all analyses with pagination
func (db *DB) ListAnalyses(limit, offset int) ([]*models.Analysis, error) {
	rows, err := db.conn.Query(`
		SELECT id, text, metadata, created_at, updated_at
		FROM analyses
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query analyses: %w", err)
	}
	defer rows.Close()

	var analyses []*models.Analysis
	for rows.Next() {
		var (
			id           string
			text         string
			metadataJSON string
			createdAt    time.Time
			updatedAt    time.Time
		)

		if err := rows.Scan(&id, &text, &metadataJSON, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		var metadata models.Metadata
		if err := json.Unmarshal([]byte(metadataJSON), &metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}

		analyses = append(analyses, &models.Analysis{
			ID:        id,
			Text:      text,
			Metadata:  metadata,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return analyses, nil
}

// DeleteAnalysis deletes an analysis by ID
func (db *DB) DeleteAnalysis(id string) error {
	result, err := db.conn.Exec("DELETE FROM analyses WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete analysis: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("analysis not found")
	}

	return nil
}
