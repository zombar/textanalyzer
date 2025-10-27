package database

import (
	"fmt"
	"testing"
	"time"

	"github.com/zombar/textanalyzer/internal/models"
)

func setupTestDatabase(t *testing.T) (*DB, func()) {
	t.Helper()
	testName := fmt.Sprintf("queries_%d", time.Now().UnixNano())
	connStr, dbCleanup := setupTestDB(t, testName)

	db, err := New(connStr)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	if err := db.Migrate(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	cleanup := func() {
		db.Close()
		dbCleanup()
	}

	return db, cleanup
}

func createTestAnalysis(id string) *models.Analysis {
	return &models.Analysis{
		ID:   id,
		Text: "This is a test text for analysis.",
		Metadata: models.Metadata{
			CharacterCount:    35,
			WordCount:         7,
			SentenceCount:     1,
			ParagraphCount:    1,
			AverageWordLength: 5.0,
			Sentiment:         "neutral",
			SentimentScore:    0.0,
			TopWords: []models.WordFrequency{
				{Word: "test", Count: 1},
				{Word: "analysis", Count: 1},
			},
			TopPhrases:         []models.PhraseInfo{},
			UniqueWords:        7,
			KeyTerms:           []string{"test", "analysis"},
			NamedEntities:      []string{},
			PotentialDates:     []string{},
			PotentialURLs:      []string{},
			EmailAddresses:     []string{},
			ReadabilityScore:   85.0,
			ReadabilityLevel:   "easy",
			ComplexWordCount:   0,
			AvgSentenceLength:  7.0,
			References:         []models.Reference{},
			Tags:               []string{"short", "neutral", "easy"},
			Language:           "english",
			QuestionCount:      0,
			ExclamationCount:   0,
			CapitalizedPercent: 14.29,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func TestSaveAnalysis(t *testing.T) {
	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	analysis := createTestAnalysis("test-001")

	err := db.SaveAnalysis(analysis)
	if err != nil {
		t.Fatalf("Failed to save analysis: %v", err)
	}
}

func TestGetAnalysis(t *testing.T) {
	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	analysis := createTestAnalysis("test-002")

	if err := db.SaveAnalysis(analysis); err != nil {
		t.Fatalf("Failed to save analysis: %v", err)
	}

	retrieved, err := db.GetAnalysis("test-002")
	if err != nil {
		t.Fatalf("Failed to get analysis: %v", err)
	}

	if retrieved.ID != analysis.ID {
		t.Errorf("Expected ID %s, got %s", analysis.ID, retrieved.ID)
	}

	if retrieved.Text != analysis.Text {
		t.Errorf("Expected text %s, got %s", analysis.Text, retrieved.Text)
	}

	if retrieved.Metadata.WordCount != analysis.Metadata.WordCount {
		t.Errorf("Expected word count %d, got %d", analysis.Metadata.WordCount, retrieved.Metadata.WordCount)
	}
}

func TestGetAnalysisNotFound(t *testing.T) {
	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	_, err := db.GetAnalysis("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent analysis")
	}

	if err.Error() != "analysis not found" {
		t.Errorf("Expected 'analysis not found' error, got %v", err)
	}
}

func TestListAnalyses(t *testing.T) {
	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	// Save multiple analyses
	for i := 1; i <= 5; i++ {
		analysis := createTestAnalysis("test-" + string(rune('0'+i)))
		if err := db.SaveAnalysis(analysis); err != nil {
			t.Fatalf("Failed to save analysis %d: %v", i, err)
		}
		time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	}

	// Test pagination
	analyses, err := db.ListAnalyses(3, 0)
	if err != nil {
		t.Fatalf("Failed to list analyses: %v", err)
	}

	if len(analyses) != 3 {
		t.Errorf("Expected 3 analyses, got %d", len(analyses))
	}

	// Test offset
	analyses, err = db.ListAnalyses(3, 3)
	if err != nil {
		t.Fatalf("Failed to list analyses with offset: %v", err)
	}

	if len(analyses) != 2 {
		t.Errorf("Expected 2 analyses with offset, got %d", len(analyses))
	}
}

func TestGetAnalysesByTag(t *testing.T) {
	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	// Create analyses with different tags
	analysis1 := createTestAnalysis("test-tag-001")
	analysis1.Metadata.Tags = []string{"positive", "long"}

	analysis2 := createTestAnalysis("test-tag-002")
	analysis2.Metadata.Tags = []string{"positive", "short"}

	analysis3 := createTestAnalysis("test-tag-003")
	analysis3.Metadata.Tags = []string{"negative", "long"}

	if err := db.SaveAnalysis(analysis1); err != nil {
		t.Fatalf("Failed to save analysis 1: %v", err)
	}
	if err := db.SaveAnalysis(analysis2); err != nil {
		t.Fatalf("Failed to save analysis 2: %v", err)
	}
	if err := db.SaveAnalysis(analysis3); err != nil {
		t.Fatalf("Failed to save analysis 3: %v", err)
	}

	// Search by tag
	analyses, err := db.GetAnalysesByTag("positive")
	if err != nil {
		t.Fatalf("Failed to get analyses by tag: %v", err)
	}

	if len(analyses) != 2 {
		t.Errorf("Expected 2 analyses with 'positive' tag, got %d", len(analyses))
	}

	// Search by another tag
	analyses, err = db.GetAnalysesByTag("long")
	if err != nil {
		t.Fatalf("Failed to get analyses by tag: %v", err)
	}

	if len(analyses) != 2 {
		t.Errorf("Expected 2 analyses with 'long' tag, got %d", len(analyses))
	}

	// Search by nonexistent tag
	analyses, err = db.GetAnalysesByTag("nonexistent")
	if err != nil {
		t.Fatalf("Failed to get analyses by tag: %v", err)
	}

	if len(analyses) != 0 {
		t.Errorf("Expected 0 analyses with 'nonexistent' tag, got %d", len(analyses))
	}
}

func TestDeleteAnalysis(t *testing.T) {
	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	analysis := createTestAnalysis("test-delete-001")

	if err := db.SaveAnalysis(analysis); err != nil {
		t.Fatalf("Failed to save analysis: %v", err)
	}

	// Delete the analysis
	err := db.DeleteAnalysis("test-delete-001")
	if err != nil {
		t.Fatalf("Failed to delete analysis: %v", err)
	}

	// Verify it's deleted
	_, err = db.GetAnalysis("test-delete-001")
	if err == nil {
		t.Error("Expected error when getting deleted analysis")
	}
}

func TestDeleteAnalysisNotFound(t *testing.T) {
	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	err := db.DeleteAnalysis("nonexistent")
	if err == nil {
		t.Error("Expected error when deleting nonexistent analysis")
	}

	if err.Error() != "analysis not found" {
		t.Errorf("Expected 'analysis not found' error, got %v", err)
	}
}

func TestMigrations(t *testing.T) {
	connStr, dbCleanup := setupTestDB(t, "test_migrations")
	defer dbCleanup()

	db, err := New(connStr)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Run migrations
	if err := db.Migrate(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Verify tables exist using PostgreSQL information_schema
	var count int
	err = db.conn.QueryRow("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema='public' AND table_name='analyses'").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to check analyses table: %v", err)
	}
	if count != 1 {
		t.Error("analyses table should exist")
	}

	err = db.conn.QueryRow("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema='public' AND table_name='tags'").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to check tags table: %v", err)
	}
	if count != 1 {
		t.Error("tags table should exist")
	}

	// Run migrations again (should be idempotent)
	if err := db.Migrate(); err != nil {
		t.Fatalf("Failed to run migrations again: %v", err)
	}
}

func TestCascadeDelete(t *testing.T) {
	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	analysis := createTestAnalysis("test-cascade-001")

	if err := db.SaveAnalysis(analysis); err != nil {
		t.Fatalf("Failed to save analysis: %v", err)
	}

	// Verify tags exist (using PostgreSQL placeholder $1)
	var tagCount int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM tags WHERE analysis_id = $1", "test-cascade-001").Scan(&tagCount)
	if err != nil {
		t.Fatalf("Failed to count tags: %v", err)
	}

	if tagCount != len(analysis.Metadata.Tags) {
		t.Errorf("Expected %d tags, got %d", len(analysis.Metadata.Tags), tagCount)
	}

	// Delete the analysis
	if err := db.DeleteAnalysis("test-cascade-001"); err != nil {
		t.Fatalf("Failed to delete analysis: %v", err)
	}

	// Verify tags are deleted (using PostgreSQL placeholder $1)
	err = db.conn.QueryRow("SELECT COUNT(*) FROM tags WHERE analysis_id = $1", "test-cascade-001").Scan(&tagCount)
	if err != nil {
		t.Fatalf("Failed to count tags after delete: %v", err)
	}

	if tagCount != 0 {
		t.Errorf("Expected 0 tags after delete, got %d", tagCount)
	}
}
