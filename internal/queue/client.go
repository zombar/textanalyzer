package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Task type constants
const (
	TypeProcessDocument = "textanalyzer:process_document"
	TypeEnrichText      = "textanalyzer:enrich_text"
	TypeEnrichImage     = "textanalyzer:enrich_image"
)

// ProcessDocumentPayload represents the payload for offline document processing
type ProcessDocumentPayload struct {
	AnalysisID   string   `json:"analysis_id"`
	Text         string   `json:"text"`
	OriginalHTML string   `json:"original_html,omitempty"` // Compressed + base64 encoded original HTML/raw text
	Images       []string `json:"images,omitempty"`
	// Tracing and timing fields
	TraceID    string `json:"trace_id,omitempty"`
	SpanID     string `json:"span_id,omitempty"`
	EnqueuedAt int64  `json:"enqueued_at"` // Unix timestamp in nanoseconds
}

// EnrichTextPayload represents the payload for AI text enrichment
type EnrichTextPayload struct {
	AnalysisID   string `json:"analysis_id"`
	Text         string `json:"text"`
	OfflineText  string `json:"offline_text,omitempty"`  // Offline analysis text to use as template
	OriginalHTML string `json:"original_html,omitempty"` // Compressed + base64 encoded original HTML/raw text
	// Tracing and timing fields
	TraceID    string `json:"trace_id,omitempty"`
	SpanID     string `json:"span_id,omitempty"`
	EnqueuedAt int64  `json:"enqueued_at"` // Unix timestamp in nanoseconds
}

// EnrichImagePayload represents the payload for AI image enrichment
type EnrichImagePayload struct {
	AnalysisID string `json:"analysis_id"`
	ImageURL   string `json:"image_url"`
	// Tracing and timing fields
	TraceID    string `json:"trace_id,omitempty"`
	SpanID     string `json:"span_id,omitempty"`
	EnqueuedAt int64  `json:"enqueued_at"` // Unix timestamp in nanoseconds
}

// Client wraps the Asynq client for enqueueing tasks
type Client struct {
	client *asynq.Client
}

// ClientConfig contains configuration for the queue client
type ClientConfig struct {
	RedisAddr string
}

// NewClient creates a new queue client
func NewClient(cfg ClientConfig) *Client {
	redisOpt := asynq.RedisClientOpt{
		Addr: cfg.RedisAddr,
	}

	client := asynq.NewClient(redisOpt)

	return &Client{
		client: client,
	}
}

// EnqueueProcessDocument enqueues an offline document processing task
func (c *Client) EnqueueProcessDocument(ctx context.Context, analysisID, text, originalHTML string, images []string) (string, error) {
	payload := ProcessDocumentPayload{
		AnalysisID:   analysisID,
		Text:         text,
		OriginalHTML: originalHTML,
		Images:       images,
		EnqueuedAt:   time.Now().UnixNano(), // Record enqueue time for queue wait metrics
	}

	// Add tracing context if available
	if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
		spanCtx := span.SpanContext()
		payload.TraceID = spanCtx.TraceID().String()
		payload.SpanID = spanCtx.SpanID().String()

		// Record enqueue event
		span.AddEvent("task_enqueued", trace.WithAttributes(
			attribute.String("task.type", TypeProcessDocument),
			attribute.String("task.id", analysisID),
			attribute.String("analysis_id", analysisID),
			attribute.Int64("enqueued_at", payload.EnqueuedAt),
		))
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal task payload: %w", err)
	}

	task := asynq.NewTask(TypeProcessDocument, payloadBytes, asynq.TaskID(analysisID))

	opts := []asynq.Option{
		asynq.MaxRetry(3),                   // Standard retry for offline processing
		asynq.Timeout(5 * time.Minute),      // 5 minute timeout
		asynq.Queue("offline-processing"),   // Offline processing queue (medium priority)
		asynq.Retention(7 * 24 * time.Hour), // Keep completed tasks for 7 days
	}

	info, err := c.client.Enqueue(task, opts...)
	if err != nil {
		return "", fmt.Errorf("failed to enqueue process document task: %w", err)
	}

	return info.ID, nil
}

// EnqueueEnrichText enqueues a high-priority AI text enrichment task
func (c *Client) EnqueueEnrichText(ctx context.Context, analysisID, text, offlineText, originalHTML string) (string, error) {
	payload := EnrichTextPayload{
		AnalysisID:   analysisID,
		Text:         text,
		OfflineText:  offlineText,
		OriginalHTML: originalHTML,
		EnqueuedAt:   time.Now().UnixNano(),
	}

	// Add tracing context if available
	if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
		spanCtx := span.SpanContext()
		payload.TraceID = spanCtx.TraceID().String()
		payload.SpanID = spanCtx.SpanID().String()

		// Record enqueue event
		span.AddEvent("task_enqueued", trace.WithAttributes(
			attribute.String("task.type", TypeEnrichText),
			attribute.String("task.id", analysisID+"-text-enrich"),
			attribute.String("analysis_id", analysisID),
			attribute.Int64("enqueued_at", payload.EnqueuedAt),
		))
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal task payload: %w", err)
	}

	taskID := analysisID + "-text-enrich"
	task := asynq.NewTask(TypeEnrichText, payloadBytes, asynq.TaskID(taskID))

	opts := []asynq.Option{
		asynq.MaxRetry(10),                    // High retry tolerance for Ollama
		asynq.Timeout(10 * time.Minute),       // 10 minute timeout for AI processing
		asynq.Queue("text-enrichment"),        // Text enrichment queue (highest priority)
		asynq.Retention(7 * 24 * time.Hour),   // Keep completed tasks for 7 days
	}

	info, err := c.client.Enqueue(task, opts...)
	if err != nil {
		return "", fmt.Errorf("failed to enqueue enrich text task: %w", err)
	}

	return info.ID, nil
}

// EnqueueEnrichImage enqueues a low-priority AI image enrichment task
func (c *Client) EnqueueEnrichImage(ctx context.Context, analysisID, imageURL string, imageIndex int) (string, error) {
	payload := EnrichImagePayload{
		AnalysisID: analysisID,
		ImageURL:   imageURL,
		EnqueuedAt: time.Now().UnixNano(),
	}

	// Add tracing context if available
	if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
		spanCtx := span.SpanContext()
		payload.TraceID = spanCtx.TraceID().String()
		payload.SpanID = spanCtx.SpanID().String()

		// Record enqueue event
		span.AddEvent("task_enqueued", trace.WithAttributes(
			attribute.String("task.type", TypeEnrichImage),
			attribute.String("task.id", fmt.Sprintf("%s-image-enrich-%d", analysisID, imageIndex)),
			attribute.String("analysis_id", analysisID),
			attribute.String("image_url", imageURL),
			attribute.Int("image_index", imageIndex),
			attribute.Int64("enqueued_at", payload.EnqueuedAt),
		))
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal task payload: %w", err)
	}

	taskID := fmt.Sprintf("%s-image-enrich-%d", analysisID, imageIndex)
	task := asynq.NewTask(TypeEnrichImage, payloadBytes, asynq.TaskID(taskID))

	opts := []asynq.Option{
		asynq.MaxRetry(10),                    // High retry tolerance for Ollama
		asynq.Timeout(15 * time.Minute),       // 15 minute timeout for image AI processing
		asynq.Queue("image-enrichment"),       // Image enrichment queue (lowest priority)
		asynq.Retention(7 * 24 * time.Hour),   // Keep completed tasks for 7 days
	}

	info, err := c.client.Enqueue(task, opts...)
	if err != nil {
		return "", fmt.Errorf("failed to enqueue enrich image task: %w", err)
	}

	return info.ID, nil
}

// Close closes the client connection
func (c *Client) Close() error {
	return c.client.Close()
}
