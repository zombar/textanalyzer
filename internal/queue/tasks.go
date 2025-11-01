package queue

import (
	"log/slog"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/hibiken/asynq"
	"github.com/docutag/textanalyzer/internal/models"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// handleProcessDocument processes offline document analysis (Stage 1)
func (w *Worker) handleProcessDocument(ctx context.Context, t *asynq.Task) error {
	// Parse payload
	var payload ProcessDocumentPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		w.logger.Error("failed to unmarshal task payload", "error", err)
		return fmt.Errorf("invalid task payload: %w", err)
	}

	analysisID := payload.AnalysisID
	text := payload.Text
	originalHTML := payload.OriginalHTML
	images := payload.Images

	// Calculate queue wait time
	var queueWaitTime time.Duration
	if payload.EnqueuedAt > 0 {
		enqueuedTime := time.Unix(0, payload.EnqueuedAt)
		queueWaitTime = time.Since(enqueuedTime)
	}

	w.logger.Info("processing document offline",
		"analysis_id", analysisID,
		"text_length", len(text),
		"has_original_html", originalHTML != "",
		"image_count", len(images),
		"queue_wait_seconds", queueWaitTime.Seconds(),
	)

	// Recreate trace context from payload if available
	var span trace.Span
	if payload.TraceID != "" && payload.SpanID != "" {
		// Parse trace ID and span ID from hex strings
		traceID, err := trace.TraceIDFromHex(payload.TraceID)
		if err == nil {
			spanID, err := trace.SpanIDFromHex(payload.SpanID)
			if err == nil {
				// Create span context from stored IDs
				remoteSpanCtx := trace.NewSpanContext(trace.SpanContextConfig{
					TraceID:    traceID,
					SpanID:     spanID,
					TraceFlags: trace.FlagsSampled,
					Remote:     true,
				})

				// Create new context with the remote span context
				ctx = trace.ContextWithRemoteSpanContext(ctx, remoteSpanCtx)

				// Start a new span linked to the enqueue span
				ctx, span = otel.Tracer("textanalyzer").Start(ctx, "asynq.task.process",
					trace.WithSpanKind(trace.SpanKindConsumer),
					trace.WithAttributes(
						attribute.String("task.type", TypeProcessDocument),
						attribute.String("analysis.id", analysisID),
						attribute.Int("text.length", len(text)),
						attribute.Bool("has_original_html", originalHTML != ""),
						attribute.Int("images.count", len(images)),
						attribute.Float64("queue.wait_time_seconds", queueWaitTime.Seconds()),
						attribute.Int64("enqueued_at", payload.EnqueuedAt),
					),
				)
				defer span.End()

				// Record queue wait time event
				span.AddEvent("task_processing_started", trace.WithAttributes(
					attribute.Float64("wait_time_seconds", queueWaitTime.Seconds()),
				))
			}
		}
	} else {
		// No trace context in payload, check current context
		if existingSpan := trace.SpanFromContext(ctx); existingSpan.SpanContext().IsValid() {
			existingSpan.SetAttributes(
				attribute.String("analysis.id", analysisID),
				attribute.Int("text.length", len(text)),
				attribute.Bool("has_original_html", originalHTML != ""),
				attribute.Int("images.count", len(images)),
				attribute.Float64("queue.wait_time_seconds", queueWaitTime.Seconds()),
			)
		}
	}

	// Perform offline analysis (rule-based, no Ollama)
	metadata := w.analyzer.AnalyzeOffline(text)

	// Create analysis record with offline results
	analysis := &models.Analysis{
		ID:           analysisID,
		Text:         text,
		OriginalHTML: originalHTML,
		Metadata:     metadata,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Save offline analysis to database
	if err := w.db.SaveAnalysis(analysis); err != nil {
		return fmt.Errorf("failed to save offline analysis: %w", err)
	}

	w.logger.Info("offline analysis saved", "analysis_id", analysisID)

	// Enqueue AI enrichment tasks if quality threshold is met
	if metadata.QualityScore != nil && metadata.QualityScore.Score >= 0.35 {
		w.logger.Info("quality threshold met, enqueueing AI enrichment",
			"analysis_id", analysisID,
			"quality_score", metadata.QualityScore.Score,
		)

		// Prepare offline cleaned text for enrichment (use CleanedText if available, otherwise use Text)
		offlineText := text
		if metadata.CleanedText != "" {
			offlineText = metadata.CleanedText
		}

		// Enqueue text enrichment (high priority) with offline text and original HTML
		if _, err := w.queueClient.EnqueueEnrichText(ctx, analysisID, text, offlineText, originalHTML); err != nil {
			w.logger.Error("failed to enqueue text enrichment", "error", err)
			// Don't fail the task if enrichment enqueue fails
		}

		// Enqueue image enrichment tasks (low priority)
		for i, imageURL := range images {
			if _, err := w.queueClient.EnqueueEnrichImage(ctx, analysisID, imageURL, i); err != nil {
				w.logger.Error("failed to enqueue image enrichment",
					"error", err,
					"image_index", i,
					"image_url", imageURL,
				)
				// Continue with other images
			}
		}
	} else {
		qualityScore := 0.0
		if metadata.QualityScore != nil {
			qualityScore = metadata.QualityScore.Score
		}
		w.logger.Info("quality threshold not met, skipping AI enrichment",
			"analysis_id", analysisID,
			"quality_score", qualityScore,
		)
	}

	return nil
}

// handleEnrichText processes AI text enrichment via Ollama (Stage 2 - High Priority)
func (w *Worker) handleEnrichText(ctx context.Context, t *asynq.Task) error {
	// Parse payload
	var payload EnrichTextPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		w.logger.Error("failed to unmarshal task payload", "error", err)
		return fmt.Errorf("invalid task payload: %w", err)
	}

	analysisID := payload.AnalysisID
	text := payload.Text
	offlineText := payload.OfflineText
	originalHTML := payload.OriginalHTML

	retryCount, _ := asynq.GetRetryCount(ctx)
	maxRetry, _ := asynq.GetMaxRetry(ctx)

	// Calculate queue wait time
	var queueWaitTime time.Duration
	if payload.EnqueuedAt > 0 {
		enqueuedTime := time.Unix(0, payload.EnqueuedAt)
		queueWaitTime = time.Since(enqueuedTime)
	}

	w.logger.Info("enriching text with AI",
		"analysis_id", analysisID,
		"text_length", len(text),
		"has_offline_text", offlineText != "",
		"has_original_html", originalHTML != "",
		"retry_count", retryCount,
		"max_retries", maxRetry,
		"queue_wait_seconds", queueWaitTime.Seconds(),
	)

	// Recreate trace context from payload if available
	var span trace.Span
	if payload.TraceID != "" && payload.SpanID != "" {
		// Parse trace ID and span ID from hex strings
		traceID, err := trace.TraceIDFromHex(payload.TraceID)
		if err == nil {
			spanID, err := trace.SpanIDFromHex(payload.SpanID)
			if err == nil {
				// Create span context from stored IDs
				remoteSpanCtx := trace.NewSpanContext(trace.SpanContextConfig{
					TraceID:    traceID,
					SpanID:     spanID,
					TraceFlags: trace.FlagsSampled,
					Remote:     true,
				})

				// Create new context with the remote span context
				ctx = trace.ContextWithRemoteSpanContext(ctx, remoteSpanCtx)

				// Start a new span linked to the enqueue span
				ctx, span = otel.Tracer("textanalyzer").Start(ctx, "asynq.task.process",
					trace.WithSpanKind(trace.SpanKindConsumer),
					trace.WithAttributes(
						attribute.String("task.type", TypeEnrichText),
						attribute.String("analysis.id", analysisID),
						attribute.Int("text.length", len(text)),
						attribute.Bool("has_offline_text", offlineText != ""),
						attribute.Bool("has_original_html", originalHTML != ""),
						attribute.Int("retry_count", retryCount),
						attribute.Float64("queue.wait_time_seconds", queueWaitTime.Seconds()),
						attribute.Int64("enqueued_at", payload.EnqueuedAt),
					),
				)
				defer span.End()

				// Record queue wait time event
				span.AddEvent("task_processing_started", trace.WithAttributes(
					attribute.Float64("wait_time_seconds", queueWaitTime.Seconds()),
				))
			}
		}
	} else {
		// No trace context in payload, check current context
		if existingSpan := trace.SpanFromContext(ctx); existingSpan.SpanContext().IsValid() {
			existingSpan.SetAttributes(
				attribute.String("analysis.id", analysisID),
				attribute.Int("text.length", len(text)),
				attribute.Bool("has_offline_text", offlineText != ""),
				attribute.Bool("has_original_html", originalHTML != ""),
				attribute.Int("retry_count", retryCount),
				attribute.Float64("queue.wait_time_seconds", queueWaitTime.Seconds()),
			)
		}
	}

	// Retrieve existing analysis
	analysis, err := w.db.GetAnalysis(analysisID)
	if err != nil {
		return fmt.Errorf("failed to retrieve analysis: %w", err)
	}

	// Start metrics timer for analysis duration with exemplar support
	timer := time.Now()
	var analysisStatus string
	defer func() {
		if analysisStatus != "" {
			duration := time.Since(timer).Seconds()
			// Record duration with exemplar linking to trace ID
			w.businessMetrics.ObserveDurationWithExemplar(ctx, w.businessMetrics.AnalysisDuration, duration, analysisStatus)
			w.businessMetrics.AnalysesTotal.WithLabelValues(analysisStatus).Inc()
		}
	}()

	// Perform AI-powered analysis with Ollama
	// If we have offline text and original HTML, use them for enhanced cleaning
	// Otherwise fall back to standard analysis
	var aiMetadata models.Metadata
	if offlineText != "" && originalHTML != "" {
		// Decompress the original HTML
		decompressedHTML, err := decompressHTML(originalHTML)
		if err != nil {
			w.logger.Warn("failed to decompress HTML, falling back to standard analysis",
				"analysis_id", analysisID,
				"error", err,
			)
			aiMetadata = w.analyzer.AnalyzeWithContext(ctx, text)
		} else {
			// Use enhanced analysis with HTML and offline text as template
			aiMetadata = w.analyzer.AnalyzeWithHTMLContext(ctx, text, offlineText, decompressedHTML)
		}
	} else {
		// Standard AI analysis
		aiMetadata = w.analyzer.AnalyzeWithContext(ctx, text)
	}

	// Merge AI results with existing offline metadata
	analysis.Metadata.Synopsis = aiMetadata.Synopsis
	analysis.Metadata.CleanedText = aiMetadata.CleanedText
	analysis.Metadata.EditorialAnalysis = aiMetadata.EditorialAnalysis
	analysis.Metadata.AIDetection = aiMetadata.AIDetection

	// Update tags with AI-generated tags if available
	if len(aiMetadata.Tags) > 0 {
		analysis.Metadata.Tags = aiMetadata.Tags
	}

	analysis.UpdatedAt = time.Now()

	// Update analysis in database
	if err := w.db.SaveAnalysis(analysis); err != nil {
		analysisStatus = "error"
		// Check if this is a retriable error (connection/timeout)
		if isRetriableOllamaError(err) {
			w.logger.Warn("retriable Ollama error, will retry",
				"analysis_id", analysisID,
				"error", err,
				"retry_count", retryCount,
			)
			return err // Let Asynq retry
		}

		// Permanent error
		w.logger.Error("permanent error enriching text",
			"analysis_id", analysisID,
			"error", err,
		)
		return fmt.Errorf("failed to update enriched analysis: %w", err)
	}

	// Record successful analysis
	analysisStatus = "success"

	// Record tags and synopsis generated
	if len(aiMetadata.Tags) > 0 {
		w.businessMetrics.TagsGeneratedTotal.Add(float64(len(aiMetadata.Tags)))
	}
	if aiMetadata.Synopsis != "" {
		w.businessMetrics.SynopsisGeneratedTotal.Inc()
	}

	w.logger.Info("text enrichment completed",
		"analysis_id", analysisID,
		"retry_count", retryCount,
	)

	return nil
}

// handleEnrichImage processes AI image enrichment via Ollama (Stage 2 - Low Priority)
func (w *Worker) handleEnrichImage(ctx context.Context, t *asynq.Task) error {
	// Parse payload
	var payload EnrichImagePayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		w.logger.Error("failed to unmarshal task payload", "error", err)
		return fmt.Errorf("invalid task payload: %w", err)
	}

	analysisID := payload.AnalysisID
	imageURL := payload.ImageURL

	retryCount, _ := asynq.GetRetryCount(ctx)
	maxRetry, _ := asynq.GetMaxRetry(ctx)

	// Calculate queue wait time
	var queueWaitTime time.Duration
	if payload.EnqueuedAt > 0 {
		enqueuedTime := time.Unix(0, payload.EnqueuedAt)
		queueWaitTime = time.Since(enqueuedTime)
	}

	w.logger.Info("enriching image with AI",
		"analysis_id", analysisID,
		"image_url", imageURL,
		"retry_count", retryCount,
		"max_retries", maxRetry,
		"queue_wait_seconds", queueWaitTime.Seconds(),
	)

	// Recreate trace context from payload if available
	var span trace.Span
	if payload.TraceID != "" && payload.SpanID != "" {
		// Parse trace ID and span ID from hex strings
		traceID, err := trace.TraceIDFromHex(payload.TraceID)
		if err == nil {
			spanID, err := trace.SpanIDFromHex(payload.SpanID)
			if err == nil {
				// Create span context from stored IDs
				remoteSpanCtx := trace.NewSpanContext(trace.SpanContextConfig{
					TraceID:    traceID,
					SpanID:     spanID,
					TraceFlags: trace.FlagsSampled,
					Remote:     true,
				})

				// Create new context with the remote span context
				ctx = trace.ContextWithRemoteSpanContext(ctx, remoteSpanCtx)

				// Start a new span linked to the enqueue span
				ctx, span = otel.Tracer("textanalyzer").Start(ctx, "asynq.task.process",
					trace.WithSpanKind(trace.SpanKindConsumer),
					trace.WithAttributes(
						attribute.String("task.type", TypeEnrichImage),
						attribute.String("analysis.id", analysisID),
						attribute.String("image.url", imageURL),
						attribute.Int("retry_count", retryCount),
						attribute.Float64("queue.wait_time_seconds", queueWaitTime.Seconds()),
						attribute.Int64("enqueued_at", payload.EnqueuedAt),
					),
				)
				defer span.End()

				// Record queue wait time event
				span.AddEvent("task_processing_started", trace.WithAttributes(
					attribute.Float64("wait_time_seconds", queueWaitTime.Seconds()),
				))
			}
		}
	} else {
		// No trace context in payload, check current context
		if existingSpan := trace.SpanFromContext(ctx); existingSpan.SpanContext().IsValid() {
			existingSpan.SetAttributes(
				attribute.String("analysis.id", analysisID),
				attribute.String("image.url", imageURL),
				attribute.Int("retry_count", retryCount),
				attribute.Float64("queue.wait_time_seconds", queueWaitTime.Seconds()),
			)
		}
	}

	// Retrieve existing analysis
	analysis, err := w.db.GetAnalysis(analysisID)
	if err != nil {
		return fmt.Errorf("failed to retrieve analysis: %w", err)
	}

	// Extract offline image metadata
	imageMetadata := w.analyzer.ExtractImageMetadata(imageURL)

	// TODO: When Ollama supports vision models, add AI image analysis here
	// For now, just store the offline metadata

	// Store image metadata in analysis (add to metadata or create image-specific field)
	// This is a placeholder - actual storage structure may need adjustment
	slog.Info("image metadata extracted", "url", imageURL, "metadata", imageMetadata)

	analysis.UpdatedAt = time.Now()

	// Update analysis in database
	if err := w.db.SaveAnalysis(analysis); err != nil {
		// Check if this is a retriable error
		if isRetriableOllamaError(err) {
			w.logger.Warn("retriable error, will retry",
				"analysis_id", analysisID,
				"error", err,
				"retry_count", retryCount,
			)
			return err // Let Asynq retry
		}

		// Permanent error
		w.logger.Error("permanent error enriching image",
			"analysis_id", analysisID,
			"error", err,
		)
		return fmt.Errorf("failed to update enriched analysis: %w", err)
	}

	w.logger.Info("image enrichment completed",
		"analysis_id", analysisID,
		"image_url", imageURL,
		"retry_count", retryCount,
	)

	return nil
}

// isRetriableOllamaError determines if an error is retriable (connection/timeout)
// vs permanent (invalid input)
func isRetriableOllamaError(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())

	// Retriable errors: connection issues, timeouts, temporary failures
	retriablePatterns := []string{
		"connection refused",
		"connection reset",
		"timeout",
		"temporary failure",
		"service unavailable",
		"bad gateway",
		"gateway timeout",
		"too many requests",
		"context deadline exceeded",
		"context canceled",
		"i/o timeout",
		"no such host",
		"network is unreachable",
	}

	for _, pattern := range retriablePatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}

	return false
}

// compressHTML compresses and base64 encodes HTML text
func compressHTML(html string) (string, error) {
	if html == "" {
		return "", nil
	}

	var buf bytes.Buffer
	gzWriter := gzip.NewWriter(&buf)

	if _, err := gzWriter.Write([]byte(html)); err != nil {
		return "", fmt.Errorf("failed to write to gzip: %w", err)
	}

	if err := gzWriter.Close(); err != nil {
		return "", fmt.Errorf("failed to close gzip writer: %w", err)
	}

	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

// decompressHTML decodes base64 and decompresses HTML text
func decompressHTML(encoded string) (string, error) {
	if encoded == "" {
		return "", nil
	}

	// Decode base64
	compressed, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}

	// Decompress gzip
	gzReader, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return "", fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	decompressed, err := io.ReadAll(gzReader)
	if err != nil {
		return "", fmt.Errorf("failed to read decompressed data: %w", err)
	}

	return string(decompressed), nil
}
