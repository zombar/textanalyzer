package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/rs/cors"
	"github.com/zombar/textanalyzer/internal/analyzer"
	"github.com/zombar/textanalyzer/internal/database"
	"github.com/zombar/textanalyzer/internal/models"
)

// Handler handles HTTP requests
type Handler struct {
	db       *database.DB
	analyzer *analyzer.Analyzer
	mux      *http.ServeMux
}

// NewHandler creates a new API handler with CORS support
func NewHandler(db *database.DB, analyzer *analyzer.Analyzer) http.Handler {
	h := &Handler{
		db:       db,
		analyzer: analyzer,
		mux:      http.NewServeMux(),
	}

	h.setupRoutes()

	// Setup CORS
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})

	return c.Handler(h.mux)
}

// setupRoutes configures all API routes
func (h *Handler) setupRoutes() {
	h.mux.HandleFunc("/api/analyze", h.handleAnalyze)
	h.mux.HandleFunc("/api/analyses", h.handleListAnalyses)
	h.mux.HandleFunc("/api/analyses/", h.handleAnalysisOperations)
	h.mux.HandleFunc("/api/search", h.handleSearchByTag)
	h.mux.HandleFunc("/api/search/reference", h.handleSearchByReference)
	h.mux.HandleFunc("/health", h.handleHealth)
}

// handleHealth handles health check requests
func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	})
}

// handleAnalyze handles text analysis requests
func (h *Handler) handleAnalyze(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Text string `json:"text"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Text == "" {
		respondError(w, "Text field is required", http.StatusBadRequest)
		return
	}

	// Perform analysis in a goroutine
	resultChan := make(chan *models.Analysis)
	errorChan := make(chan error)

	go func() {
		analysis := &models.Analysis{
			ID:        generateID(),
			Text:      req.Text,
			Metadata:  h.analyzer.Analyze(req.Text),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		if err := h.db.SaveAnalysis(analysis); err != nil {
			errorChan <- err
			return
		}

		resultChan <- analysis
	}()

	select {
	case analysis := <-resultChan:
		respondJSON(w, analysis, http.StatusCreated)
	case err := <-errorChan:
		respondError(w, err.Error(), http.StatusInternalServerError)
	case <-time.After(400 * time.Second):
		respondError(w, "Analysis timeout", http.StatusRequestTimeout)
	}
}

// handleListAnalyses handles listing all analyses with pagination
func (h *Handler) handleListAnalyses(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	limit := 10
	offset := 0

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	// Fetch analyses in a goroutine
	resultChan := make(chan []*models.Analysis)
	errorChan := make(chan error)

	go func() {
		analyses, err := h.db.ListAnalyses(limit, offset)
		if err != nil {
			errorChan <- err
			return
		}
		resultChan <- analyses
	}()

	select {
	case analyses := <-resultChan:
		respondJSON(w, analyses, http.StatusOK)
	case err := <-errorChan:
		respondError(w, err.Error(), http.StatusInternalServerError)
	case <-time.After(10 * time.Second):
		respondError(w, "Request timeout", http.StatusRequestTimeout)
	}
}

// handleAnalysisOperations handles GET and DELETE for specific analyses
func (h *Handler) handleAnalysisOperations(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/api/analyses/"):]
	if id == "" {
		respondError(w, "Analysis ID is required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getAnalysis(w, r, id)
	case http.MethodDelete:
		h.deleteAnalysis(w, r, id)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// getAnalysis retrieves a specific analysis
func (h *Handler) getAnalysis(w http.ResponseWriter, r *http.Request, id string) {
	resultChan := make(chan *models.Analysis)
	errorChan := make(chan error)

	go func() {
		analysis, err := h.db.GetAnalysis(id)
		if err != nil {
			errorChan <- err
			return
		}
		resultChan <- analysis
	}()

	select {
	case analysis := <-resultChan:
		respondJSON(w, analysis, http.StatusOK)
	case err := <-errorChan:
		if err.Error() == "analysis not found" {
			respondError(w, err.Error(), http.StatusNotFound)
		} else {
			respondError(w, err.Error(), http.StatusInternalServerError)
		}
	case <-time.After(10 * time.Second):
		respondError(w, "Request timeout", http.StatusRequestTimeout)
	}
}

// deleteAnalysis deletes a specific analysis
func (h *Handler) deleteAnalysis(w http.ResponseWriter, r *http.Request, id string) {
	errorChan := make(chan error)
	doneChan := make(chan bool)

	go func() {
		if err := h.db.DeleteAnalysis(id); err != nil {
			errorChan <- err
			return
		}
		doneChan <- true
	}()

	select {
	case <-doneChan:
		w.WriteHeader(http.StatusNoContent)
	case err := <-errorChan:
		if err.Error() == "analysis not found" {
			respondError(w, err.Error(), http.StatusNotFound)
		} else {
			respondError(w, err.Error(), http.StatusInternalServerError)
		}
	case <-time.After(10 * time.Second):
		respondError(w, "Request timeout", http.StatusRequestTimeout)
	}
}

// handleSearchByTag handles searching analyses by tag
func (h *Handler) handleSearchByTag(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	tag := r.URL.Query().Get("tag")
	if tag == "" {
		respondError(w, "Tag parameter is required", http.StatusBadRequest)
		return
	}

	// Search in a goroutine
	resultChan := make(chan []*models.Analysis)
	errorChan := make(chan error)

	go func() {
		analyses, err := h.db.GetAnalysesByTag(tag)
		if err != nil {
			errorChan <- err
			return
		}
		resultChan <- analyses
	}()

	select {
	case analyses := <-resultChan:
		respondJSON(w, analyses, http.StatusOK)
	case err := <-errorChan:
		respondError(w, err.Error(), http.StatusInternalServerError)
	case <-time.After(10 * time.Second):
		respondError(w, "Request timeout", http.StatusRequestTimeout)
	}
}

// handleSearchByReference handles searching analyses by reference text
func (h *Handler) handleSearchByReference(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	reference := r.URL.Query().Get("reference")
	if reference == "" {
		respondError(w, "Reference parameter is required", http.StatusBadRequest)
		return
	}

	// Search in a goroutine
	resultChan := make(chan []*models.Analysis)
	errorChan := make(chan error)

	go func() {
		analyses, err := h.db.GetAnalysesByReference(reference)
		if err != nil {
			errorChan <- err
			return
		}
		resultChan <- analyses
	}()

	select {
	case analyses := <-resultChan:
		respondJSON(w, analyses, http.StatusOK)
	case err := <-errorChan:
		respondError(w, err.Error(), http.StatusInternalServerError)
	case <-time.After(10 * time.Second):
		respondError(w, "Request timeout", http.StatusRequestTimeout)
	}
}

// respondJSON sends a JSON response
func respondJSON(w http.ResponseWriter, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// respondError sends an error response
func respondError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	})
}

// generateID generates a unique ID for an analysis
func generateID() string {
	return time.Now().Format("20060102150405") + "-" + strconv.FormatInt(time.Now().UnixNano()%1000000, 10)
}
