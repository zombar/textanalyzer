package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

// TestAnalyzeTracing tests that the analyze handler creates proper tracing spans
func TestAnalyzeTracing(t *testing.T) {
	// Setup trace exporter
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)
	otel.SetTracerProvider(tp)
	defer otel.SetTracerProvider(trace.NewNoopTracerProvider())

	// Setup handler
	handler, _, cleanup := setupTestHandler(t)
	defer cleanup()

	// Create request
	reqBody := `{"text":"This is a test article about artificial intelligence and machine learning. It provides detailed information about AI technology and its applications."}`
	req := httptest.NewRequest(http.MethodPost, "/api/analyze", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	// Add trace context to request
	ctx, span := tp.Tracer("test").Start(context.Background(), "test-request")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	// Execute handler
	handler.handleAnalyze(w, req)
	span.End()

	// Force flush to ensure all spans are recorded
	tp.ForceFlush(context.Background())

	// Get recorded spans
	spans := exporter.GetSpans()

	// Verify we have spans
	if len(spans) == 0 {
		t.Fatal("No spans were recorded")
	}

	// Test 1: Verify ai.analyze_text span exists
	var aiSpan *tracetest.SpanStub
	for i := range spans {
		if spans[i].Name == "ai.analyze_text" {
			aiSpan = &spans[i]
			break
		}
	}

	if aiSpan == nil {
		t.Error("ai.analyze_text span not found")
		t.Logf("Available spans: %v", getSpanNames(spans))
	} else {
		// Verify analysis attributes exist
		attrs := aiSpan.Attributes
		hasAnalysisID := false
		hasTextLength := false

		for _, attr := range attrs {
			if string(attr.Key) == "analysis.id" {
				hasAnalysisID = true
			}
			if string(attr.Key) == "text.length" {
				hasTextLength = true
			}
		}

		if !hasAnalysisID {
			t.Error("analysis.id attribute not found on ai.analyze_text span")
		}
		if !hasTextLength {
			t.Error("text.length attribute not found on ai.analyze_text span")
		}
	}

	// Test 2: Verify database.save_analysis span exists
	var saveSpan *tracetest.SpanStub
	for i := range spans {
		if spans[i].Name == "database.save_analysis" {
			saveSpan = &spans[i]
			break
		}
	}

	if saveSpan == nil {
		t.Error("database.save_analysis span not found")
	} else {
		// Verify analysis ID attribute exists
		attrs := saveSpan.Attributes
		hasAnalysisID := false

		for _, attr := range attrs {
			if string(attr.Key) == "analysis.id" {
				hasAnalysisID = true
			}
		}

		if !hasAnalysisID {
			t.Error("analysis.id attribute not found on database.save_analysis span")
		}
	}

	// Verify response was successful
	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d: %s", w.Code, w.Body.String())
	}
}

// getSpanNames returns a list of span names for debugging
func getSpanNames(spans tracetest.SpanStubs) []string {
	names := make([]string, len(spans))
	for i, span := range spans {
		names[i] = span.Name
	}
	return names
}
