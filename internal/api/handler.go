package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/rs/cors"
	"github.com/zombar/purpletab/pkg/metrics"
	"github.com/zombar/purpletab/pkg/tracing"
	"github.com/zombar/textanalyzer/internal/analyzer"
	"github.com/zombar/textanalyzer/internal/database"
	"github.com/zombar/textanalyzer/internal/models"
	"go.opentelemetry.io/otel/attribute"
)

// Handler handles HTTP requests
type Handler struct {
	db          *database.DB
	analyzer    *analyzer.Analyzer
	mux         *http.ServeMux
	httpMetrics *metrics.HTTPMetrics
	dbMetrics   *metrics.DatabaseMetrics
}

// NewHandler creates a new API handler with CORS support and metrics
func NewHandler(db *database.DB, analyzer *analyzer.Analyzer) http.Handler {
	// Initialize Prometheus metrics
	httpMetrics := metrics.NewHTTPMetrics("textanalyzer")
	dbMetrics := metrics.NewDatabaseMetrics("textanalyzer")

	h := &Handler{
		db:          db,
		analyzer:    analyzer,
		mux:         http.NewServeMux(),
		httpMetrics: httpMetrics,
		dbMetrics:   dbMetrics,
	}

	h.setupRoutes()

	// Start periodic database stats collection
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			dbMetrics.UpdateDBStats(db.Conn())
		}
	}()

	// Setup CORS
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})

	// Wrap with metrics middleware
	return httpMetrics.HTTPMiddleware(c.Handler(h.mux))
}

// setupRoutes configures all API routes
func (h *Handler) setupRoutes() {
	h.mux.Handle("/metrics", metrics.Handler()) // Prometheus metrics endpoint
	h.mux.HandleFunc("/api/analyze", h.handleAnalyze)
	h.mux.HandleFunc("/api/analyses", h.handleListAnalyses)
	h.mux.HandleFunc("/api/analyses/", h.handleAnalysisOperations)
	h.mux.HandleFunc("/api/uuid/", h.handleUUIDOperations)
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

	// Add text length to span
	tracing.SetSpanAttributes(r.Context(),
		attribute.Int("text.length", len(req.Text)))

	// Perform analysis in a goroutine
	resultChan := make(chan *models.Analysis)
	errorChan := make(chan error)

	// Pass context to goroutine for tracing
	ctx := r.Context()

	go func() {
		analysisID := generateID()

		// Create span for AI analysis
		ctx, analyzeSpan := tracing.StartSpan(ctx, "ai.analyze_text")
		analyzeSpan.SetAttributes(
			attribute.String("analysis.id", analysisID),
			attribute.Int("text.length", len(req.Text)))

		metadata := h.analyzer.Analyze(req.Text)

		// Add result metrics
		if len(metadata.Tags) > 0 {
			analyzeSpan.SetAttributes(attribute.StringSlice("analysis.tags", metadata.Tags))
		}
		if metadata.Synopsis != "" {
			analyzeSpan.SetAttributes(attribute.Int("synopsis.length", len(metadata.Synopsis)))
		}
		analyzeSpan.End()

		analysis := &models.Analysis{
			ID:        analysisID,
			Text:      req.Text,
			Metadata:  metadata,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		// Create span for database save
		ctx, saveSpan := tracing.StartSpan(ctx, "database.save_analysis")
		saveSpan.SetAttributes(attribute.String("analysis.id", analysisID))

		if err := h.db.SaveAnalysis(analysis); err != nil {
			tracing.RecordError(ctx, err)
			saveSpan.End()
			errorChan <- err
			return
		}

		tracing.AddEvent(ctx, "analysis_saved",
			attribute.String("analysis.id", analysisID))
		saveSpan.End()

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
	case <-time.After(30 * time.Second):
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
	case <-time.After(30 * time.Second):
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
	case <-time.After(30 * time.Second):
		respondError(w, "Request timeout", http.StatusRequestTimeout)
	}
}

// handleUUIDOperations handles GET and DELETE for analyses by UUID
func (h *Handler) handleUUIDOperations(w http.ResponseWriter, r *http.Request) {
	uuid := r.URL.Path[len("/api/uuid/"):]
	if uuid == "" {
		respondError(w, "UUID is required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getAnalysisByUUID(w, uuid)
	case http.MethodDelete:
		h.deleteAnalysisByUUID(w, uuid)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// getAnalysisByUUID retrieves an analysis by UUID
func (h *Handler) getAnalysisByUUID(w http.ResponseWriter, uuid string) {
	resultChan := make(chan *models.Analysis)
	errorChan := make(chan error)

	go func() {
		analysis, err := h.db.GetAnalysisByUUID(uuid)
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
	case <-time.After(30 * time.Second):
		respondError(w, "Request timeout", http.StatusRequestTimeout)
	}
}

// deleteAnalysisByUUID deletes an analysis by UUID
func (h *Handler) deleteAnalysisByUUID(w http.ResponseWriter, uuid string) {
	errorChan := make(chan error)
	doneChan := make(chan bool)

	go func() {
		if err := h.db.DeleteAnalysisByUUID(uuid); err != nil {
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
	case <-time.After(30 * time.Second):
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
	case <-time.After(30 * time.Second):
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
	case <-time.After(30 * time.Second):
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

// generateID generates a UUID for an analysis
func generateID() string {
	uuid := make([]byte, 16)
	_, err := rand.Read(uuid)
	if err != nil {
		// Fallback to timestamp-based ID if random generation fails
		return time.Now().Format("20060102150405") + "-" + strconv.FormatInt(time.Now().UnixNano()%1000000, 10)
	}

	// Set version (4) and variant bits according to RFC 4122
	uuid[6] = (uuid[6] & 0x0f) | 0x40 // Version 4
	uuid[8] = (uuid[8] & 0x3f) | 0x80 // Variant bits

	// Format as standard UUID string: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		hex.EncodeToString(uuid[0:4]),
		hex.EncodeToString(uuid[4:6]),
		hex.EncodeToString(uuid[6:8]),
		hex.EncodeToString(uuid[8:10]),
		hex.EncodeToString(uuid[10:16]))
}
