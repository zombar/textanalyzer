package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/cors"
	"github.com/docutag/platform/pkg/tracing"
	"github.com/docutag/textanalyzer/internal/analyzer"
	"github.com/docutag/textanalyzer/internal/database"
	"github.com/docutag/textanalyzer/internal/models"
	"go.opentelemetry.io/otel/attribute"
)

// Handler handles HTTP requests
type Handler struct {
	db          *database.DB
	analyzer    *analyzer.Analyzer
	queueClient interface {
		EnqueueProcessDocument(ctx context.Context, analysisID, text, originalHTML string, images []string) (string, error)
	}
	mux         *http.ServeMux
}

// NewHandler creates a new API handler with CORS support and metrics
func NewHandler(db *database.DB, analyzer *analyzer.Analyzer, queueClient interface {
	EnqueueProcessDocument(ctx context.Context, analysisID, text, originalHTML string, images []string) (string, error)
}) http.Handler {
	// Initialize Prometheus metrics

	h := &Handler{
		db:          db,
		analyzer:    analyzer,
		queueClient: queueClient,
		mux:         http.NewServeMux(),
	}

	h.setupRoutes()


	// Setup CORS
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})

	// Wrap with CORS
	return c.Handler(h.mux)
}

// setupRoutes configures all API routes
func (h *Handler) setupRoutes() {
	h.mux.Handle("/metrics", promhttp.Handler()) // Prometheus metrics endpoint
	h.mux.HandleFunc("/api/analyze", h.handleAnalyze)
	h.mux.HandleFunc("/api/jobs/", h.handleJobStatus)
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

// handleAnalyze handles text analysis requests - now queue-based
func (h *Handler) handleAnalyze(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Text         string   `json:"text"`
		OriginalHTML string   `json:"original_html,omitempty"` // Compressed + base64 encoded original HTML/raw text
		Images       []string `json:"images,omitempty"`
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
		attribute.Int("text.length", len(req.Text)),
		attribute.Int("images.count", len(req.Images)))

	// Generate analysis ID
	analysisID := generateID()

	// Enqueue document processing task
	ctx := r.Context()
	taskID, err := h.queueClient.EnqueueProcessDocument(ctx, analysisID, req.Text, req.OriginalHTML, req.Images)
	if err != nil {
		respondError(w, fmt.Sprintf("Failed to enqueue analysis: %v", err), http.StatusInternalServerError)
		return
	}

	// Return job ID immediately
	respondJSON(w, map[string]interface{}{
		"job_id":   analysisID,
		"task_id":  taskID,
		"status":   "queued",
		"message":  "Analysis queued for processing",
	}, http.StatusAccepted)
}

// handleJobStatus handles job status requests
func (h *Handler) handleJobStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract job ID from path
	jobID := r.URL.Path[len("/api/jobs/"):]
	if idx := strings.Index(jobID, "/"); idx != -1 {
		jobID = jobID[:idx]
	}

	if jobID == "" {
		respondError(w, "Job ID is required", http.StatusBadRequest)
		return
	}

	// Try to retrieve the analysis
	analysis, err := h.db.GetAnalysis(jobID)
	if err != nil {
		if err.Error() == "analysis not found" {
			respondJSON(w, map[string]interface{}{
				"job_id": jobID,
				"status": "not_found",
				"message": "Analysis not found - it may still be queued or has expired",
			}, http.StatusNotFound)
			return
		}
		respondError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Determine status based on analysis metadata
	status := "completed"
	if analysis.Metadata.Synopsis == "" && analysis.Metadata.CleanedText == "" {
		// No AI enrichment yet
		if analysis.Metadata.QualityScore != nil && analysis.Metadata.QualityScore.Score < 0.35 {
			status = "completed_offline_only" // Below threshold, won't be enriched
		} else {
			status = "processing" // Offline complete, AI enrichment pending/in progress
		}
	}

	response := map[string]interface{}{
		"job_id":     jobID,
		"status":     status,
		"created_at": analysis.CreatedAt,
		"updated_at": analysis.UpdatedAt,
	}

	// Include analysis if completed
	if status == "completed" || status == "completed_offline_only" {
		response["analysis"] = analysis
	}

	respondJSON(w, response, http.StatusOK)
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
