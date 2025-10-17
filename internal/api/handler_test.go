package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/zombar/textanalyzer/internal/analyzer"
	"github.com/zombar/textanalyzer/internal/database"
	"github.com/zombar/textanalyzer/internal/models"
)

func setupTestHandler(t *testing.T) (*Handler, *database.DB, func()) {
	dbPath := "test_api_" + time.Now().Format("20060102150405") + ".db"

	db, err := database.New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	if err := db.Migrate(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	a := analyzer.New()
	_ = NewHandler(db, a)

	// Create internal handler for testing
	handler := &Handler{
		db:       db,
		analyzer: a,
		mux:      http.NewServeMux(),
	}
	handler.setupRoutes()

	cleanup := func() {
		db.Close()
		os.Remove(dbPath)
	}

	return handler, db, cleanup
}

func TestHealthEndpoint(t *testing.T) {
	handler, _, cleanup := setupTestHandler(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	handler.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]string
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["status"] != "ok" {
		t.Errorf("Expected status 'ok', got '%s'", response["status"])
	}
}

func TestAnalyzeEndpoint(t *testing.T) {
	handler, _, cleanup := setupTestHandler(t)
	defer cleanup()

	reqBody := map[string]string{
		"text": "This is a test text for analysis. It contains multiple sentences. The analysis should extract metadata.",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/analyze", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", w.Code)
	}

	var response models.Analysis
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.ID == "" {
		t.Error("Expected analysis ID to be set")
	}

	if response.Text != reqBody["text"] {
		t.Error("Expected text to match request")
	}

	if response.Metadata.WordCount == 0 {
		t.Error("Expected word count to be greater than 0")
	}
}

func TestAnalyzeEndpointEmptyText(t *testing.T) {
	handler, _, cleanup := setupTestHandler(t)
	defer cleanup()

	reqBody := map[string]string{
		"text": "",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/analyze", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestAnalyzeEndpointInvalidMethod(t *testing.T) {
	handler, _, cleanup := setupTestHandler(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/analyze", nil)
	w := httptest.NewRecorder()

	handler.mux.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

func TestGetAnalysisEndpoint(t *testing.T) {
	handler, db, cleanup := setupTestHandler(t)
	defer cleanup()

	// Create a test analysis
	analysis := &models.Analysis{
		ID:   "test-get-001",
		Text: "Test text",
		Metadata: models.Metadata{
			WordCount: 2,
			Tags:      []string{"test"},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := db.SaveAnalysis(analysis); err != nil {
		t.Fatalf("Failed to save test analysis: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/analyses/test-get-001", nil)
	w := httptest.NewRecorder()

	handler.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response models.Analysis
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.ID != "test-get-001" {
		t.Errorf("Expected ID 'test-get-001', got '%s'", response.ID)
	}
}

func TestGetAnalysisNotFound(t *testing.T) {
	handler, _, cleanup := setupTestHandler(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/analyses/nonexistent", nil)
	w := httptest.NewRecorder()

	handler.mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestListAnalysesEndpoint(t *testing.T) {
	handler, db, cleanup := setupTestHandler(t)
	defer cleanup()

	// Create multiple test analyses
	for i := 1; i <= 5; i++ {
		analysis := &models.Analysis{
			ID:   "test-list-" + string(rune('0'+i)),
			Text: "Test text",
			Metadata: models.Metadata{
				WordCount: 2,
				Tags:      []string{"test"},
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := db.SaveAnalysis(analysis); err != nil {
			t.Fatalf("Failed to save test analysis: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/analyses?limit=3&offset=0", nil)
	w := httptest.NewRecorder()

	handler.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response []*models.Analysis
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(response) != 3 {
		t.Errorf("Expected 3 analyses, got %d", len(response))
	}
}

func TestDeleteAnalysisEndpoint(t *testing.T) {
	handler, db, cleanup := setupTestHandler(t)
	defer cleanup()

	// Create a test analysis
	analysis := &models.Analysis{
		ID:   "test-delete-001",
		Text: "Test text",
		Metadata: models.Metadata{
			WordCount: 2,
			Tags:      []string{"test"},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := db.SaveAnalysis(analysis); err != nil {
		t.Fatalf("Failed to save test analysis: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/analyses/test-delete-001", nil)
	w := httptest.NewRecorder()

	handler.mux.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", w.Code)
	}

	// Verify it's deleted
	_, err := db.GetAnalysis("test-delete-001")
	if err == nil {
		t.Error("Expected analysis to be deleted")
	}
}

func TestSearchByTagEndpoint(t *testing.T) {
	handler, db, cleanup := setupTestHandler(t)
	defer cleanup()

	// Create test analyses with tags
	analysis1 := &models.Analysis{
		ID:   "test-search-001",
		Text: "Test text 1",
		Metadata: models.Metadata{
			WordCount: 2,
			Tags:      []string{"positive", "long"},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	analysis2 := &models.Analysis{
		ID:   "test-search-002",
		Text: "Test text 2",
		Metadata: models.Metadata{
			WordCount: 2,
			Tags:      []string{"positive", "short"},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := db.SaveAnalysis(analysis1); err != nil {
		t.Fatalf("Failed to save test analysis 1: %v", err)
	}
	if err := db.SaveAnalysis(analysis2); err != nil {
		t.Fatalf("Failed to save test analysis 2: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/search?tag=positive", nil)
	w := httptest.NewRecorder()

	handler.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response []*models.Analysis
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(response) != 2 {
		t.Errorf("Expected 2 analyses with 'positive' tag, got %d", len(response))
	}
}

func TestSearchByTagMissingParameter(t *testing.T) {
	handler, _, cleanup := setupTestHandler(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/search", nil)
	w := httptest.NewRecorder()

	handler.mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestGenerateID(t *testing.T) {
	id1 := generateID()
	time.Sleep(1 * time.Millisecond)
	id2 := generateID()

	if id1 == id2 {
		t.Error("Generated IDs should be unique")
	}

	if len(id1) == 0 {
		t.Error("Generated ID should not be empty")
	}

	// Verify UUID format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
	if len(id1) != 36 {
		t.Errorf("Expected UUID length 36, got %d", len(id1))
	}

	// Check for proper UUID format with hyphens
	if id1[8] != '-' || id1[13] != '-' || id1[18] != '-' || id1[23] != '-' {
		t.Errorf("Generated ID does not match UUID format: %s", id1)
	}
}

func TestGetAnalysisByUUID(t *testing.T) {
	handler, db, cleanup := setupTestHandler(t)
	defer cleanup()

	// Create a test analysis with UUID
	uuid := generateID()
	analysis := &models.Analysis{
		ID:   uuid,
		Text: "Test text for UUID",
		Metadata: models.Metadata{
			WordCount: 4,
			Tags:      []string{"test", "uuid"},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := db.SaveAnalysis(analysis); err != nil {
		t.Fatalf("Failed to save test analysis: %v", err)
	}

	// Test GET /api/uuid/{uuid}
	req := httptest.NewRequest(http.MethodGet, "/api/uuid/"+uuid, nil)
	w := httptest.NewRecorder()

	handler.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response models.Analysis
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.ID != uuid {
		t.Errorf("Expected ID '%s', got '%s'", uuid, response.ID)
	}

	if response.Text != "Test text for UUID" {
		t.Errorf("Expected text to match, got '%s'", response.Text)
	}
}

func TestDeleteAnalysisByUUID(t *testing.T) {
	handler, db, cleanup := setupTestHandler(t)
	defer cleanup()

	// Create a test analysis with UUID
	uuid := generateID()
	analysis := &models.Analysis{
		ID:   uuid,
		Text: "Test text to delete",
		Metadata: models.Metadata{
			WordCount: 4,
			Tags:      []string{"test"},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := db.SaveAnalysis(analysis); err != nil {
		t.Fatalf("Failed to save test analysis: %v", err)
	}

	// Test DELETE /api/uuid/{uuid}
	req := httptest.NewRequest(http.MethodDelete, "/api/uuid/"+uuid, nil)
	w := httptest.NewRecorder()

	handler.mux.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", w.Code)
	}

	// Verify it's deleted
	_, err := db.GetAnalysisByUUID(uuid)
	if err == nil {
		t.Error("Expected analysis to be deleted")
	}
}

func TestGetAnalysisByUUIDNotFound(t *testing.T) {
	handler, _, cleanup := setupTestHandler(t)
	defer cleanup()

	// Generate a UUID that doesn't exist
	uuid := generateID()

	req := httptest.NewRequest(http.MethodGet, "/api/uuid/"+uuid, nil)
	w := httptest.NewRecorder()

	handler.mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestDeleteAnalysisByUUIDNotFound(t *testing.T) {
	handler, _, cleanup := setupTestHandler(t)
	defer cleanup()

	// Generate a UUID that doesn't exist
	uuid := generateID()

	req := httptest.NewRequest(http.MethodDelete, "/api/uuid/"+uuid, nil)
	w := httptest.NewRecorder()

	handler.mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}
