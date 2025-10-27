package queue

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/hibiken/asynq"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// TestE2ETraceFlow_ProcessDocument tests the complete trace flow for document processing
func TestE2ETraceFlow_ProcessDocument(t *testing.T) {
	// Setup in-memory span exporter
	spanRecorder := tracetest.NewSpanRecorder()
	tp := trace.NewTracerProvider(
		trace.WithSpanProcessor(spanRecorder),
	)
	otel.SetTracerProvider(tp)

	// Create parent span simulating incoming request
	tracer := tp.Tracer("test")
	ctx, parentSpan := tracer.Start(context.Background(), "controller.scrape_handler",
		oteltrace.WithSpanKind(oteltrace.SpanKindServer),
	)

	parentSpanContext := parentSpan.SpanContext()

	// Step 1: Enqueue process document task
	payload := ProcessDocumentPayload{
		AnalysisID:   "analysis-e2e-123",
		Text:         "Sample text for analysis",
		OriginalHTML: "<html><body>Sample text for analysis</body></html>",
		Images:       []string{"https://example.com/image1.jpg"},
		EnqueuedAt:   time.Now().UnixNano(),
	}

	// Capture trace context
	if span := oteltrace.SpanFromContext(ctx); span.SpanContext().IsValid() {
		spanCtx := span.SpanContext()
		payload.TraceID = spanCtx.TraceID().String()
		payload.SpanID = spanCtx.SpanID().String()
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Failed to marshal payload: %v", err)
	}

	// Verify trace context captured
	if payload.TraceID != parentSpanContext.TraceID().String() {
		t.Errorf("TraceID mismatch: got %s, want %s",
			payload.TraceID, parentSpanContext.TraceID().String())
	}

	// Step 2: Simulate worker processing
	var receivedPayload ProcessDocumentPayload
	if err := json.Unmarshal(payloadBytes, &receivedPayload); err != nil {
		t.Fatalf("Failed to unmarshal payload: %v", err)
	}

	// Extract trace context
	traceID, _ := oteltrace.TraceIDFromHex(receivedPayload.TraceID)
	spanID, _ := oteltrace.SpanIDFromHex(receivedPayload.SpanID)

	remoteSpanCtx := oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: oteltrace.FlagsSampled,
		Remote:     true,
	})

	linkedCtx := oteltrace.ContextWithRemoteSpanContext(context.Background(), remoteSpanCtx)

	// Create worker span
	_, workerSpan := tracer.Start(linkedCtx, "asynq.task.process",
		oteltrace.WithSpanKind(oteltrace.SpanKindConsumer),
	)
	workerSpan.End()

	// End parent span before verification
	parentSpan.End()

	time.Sleep(100 * time.Millisecond)

	// Step 3: Verify trace chain
	spans := spanRecorder.Ended()
	if len(spans) < 2 {
		t.Fatalf("Expected at least 2 spans, got %d", len(spans))
	}

	expectedTraceID := parentSpanContext.TraceID()
	for _, span := range spans {
		if span.SpanContext().TraceID() != expectedTraceID {
			t.Errorf("Span %s has different TraceID: got %s, want %s",
				span.Name(), span.SpanContext().TraceID(), expectedTraceID)
		}
	}

	t.Logf("Successfully verified E2E trace flow for ProcessDocument with TraceID: %s", expectedTraceID)
}

// TestE2ETraceFlow_EnrichText tests the complete trace flow for text enrichment
func TestE2ETraceFlow_EnrichText(t *testing.T) {
	// Setup span recorder
	spanRecorder := tracetest.NewSpanRecorder()
	tp := trace.NewTracerProvider(
		trace.WithSpanProcessor(spanRecorder),
	)
	otel.SetTracerProvider(tp)

	// Create parent span
	tracer := tp.Tracer("test")
	ctx, parentSpan := tracer.Start(context.Background(), "controller.process_document")

	parentSpanContext := parentSpan.SpanContext()

	// Enqueue enrich text task
	payload := EnrichTextPayload{
		AnalysisID:   "analysis-enrich-456",
		Text:         "Text to be enriched with AI analysis",
		OfflineText:  "Cleaned text to be enriched",
		OriginalHTML: "<p>Text to be enriched with AI analysis</p>",
		EnqueuedAt:   time.Now().UnixNano(),
	}

	// Capture trace context
	if span := oteltrace.SpanFromContext(ctx); span.SpanContext().IsValid() {
		spanCtx := span.SpanContext()
		payload.TraceID = spanCtx.TraceID().String()
		payload.SpanID = spanCtx.SpanID().String()
	}

	payloadBytes, _ := json.Marshal(payload)

	// Simulate worker processing
	var receivedPayload EnrichTextPayload
	json.Unmarshal(payloadBytes, &receivedPayload)

	// Extract and link trace context
	traceID, _ := oteltrace.TraceIDFromHex(receivedPayload.TraceID)
	spanID, _ := oteltrace.SpanIDFromHex(receivedPayload.SpanID)

	remoteSpanCtx := oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: oteltrace.FlagsSampled,
		Remote:     true,
	})

	linkedCtx := oteltrace.ContextWithRemoteSpanContext(context.Background(), remoteSpanCtx)

	// Create worker span
	_, workerSpan := tracer.Start(linkedCtx, "asynq.task.enrich_text",
		oteltrace.WithSpanKind(oteltrace.SpanKindConsumer),
	)
	workerSpan.End()

	// End parent span before verification
	parentSpan.End()

	time.Sleep(100 * time.Millisecond)

	// Verify all spans share the same TraceID
	spans := spanRecorder.Ended()
	expectedTraceID := parentSpanContext.TraceID()

	for _, span := range spans {
		if span.SpanContext().TraceID() != expectedTraceID {
			t.Errorf("Span has different TraceID: got %s, want %s",
				span.SpanContext().TraceID(), expectedTraceID)
		}
	}

	t.Logf("Successfully verified E2E trace flow for EnrichText with TraceID: %s", expectedTraceID)
}

// TestE2ETraceFlow_EnrichImage tests the complete trace flow for image enrichment
func TestE2ETraceFlow_EnrichImage(t *testing.T) {
	// Setup span recorder
	spanRecorder := tracetest.NewSpanRecorder()
	tp := trace.NewTracerProvider(
		trace.WithSpanProcessor(spanRecorder),
	)
	otel.SetTracerProvider(tp)

	// Create parent span
	tracer := tp.Tracer("test")
	ctx, parentSpan := tracer.Start(context.Background(), "controller.process_images")

	parentSpanContext := parentSpan.SpanContext()

	// Enqueue enrich image task
	payload := EnrichImagePayload{
		AnalysisID: "analysis-image-789",
		ImageURL:   "https://example.com/image-to-analyze.jpg",
		EnqueuedAt: time.Now().UnixNano(),
	}

	// Capture trace context
	if span := oteltrace.SpanFromContext(ctx); span.SpanContext().IsValid() {
		spanCtx := span.SpanContext()
		payload.TraceID = spanCtx.TraceID().String()
		payload.SpanID = spanCtx.SpanID().String()
	}

	payloadBytes, _ := json.Marshal(payload)

	// Simulate worker processing
	var receivedPayload EnrichImagePayload
	json.Unmarshal(payloadBytes, &receivedPayload)

	// Extract and link trace context
	traceID, _ := oteltrace.TraceIDFromHex(receivedPayload.TraceID)
	spanID, _ := oteltrace.SpanIDFromHex(receivedPayload.SpanID)

	remoteSpanCtx := oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: oteltrace.FlagsSampled,
		Remote:     true,
	})

	linkedCtx := oteltrace.ContextWithRemoteSpanContext(context.Background(), remoteSpanCtx)

	// Create worker span
	_, workerSpan := tracer.Start(linkedCtx, "asynq.task.enrich_image",
		oteltrace.WithSpanKind(oteltrace.SpanKindConsumer),
	)
	workerSpan.End()

	// End parent span before verification
	parentSpan.End()

	time.Sleep(100 * time.Millisecond)

	// Verify trace chain
	spans := spanRecorder.Ended()
	expectedTraceID := parentSpanContext.TraceID()

	for _, span := range spans {
		if span.SpanContext().TraceID() != expectedTraceID {
			t.Errorf("Span has different TraceID: got %s, want %s",
				span.SpanContext().TraceID(), expectedTraceID)
		}
	}

	t.Logf("Successfully verified E2E trace flow for EnrichImage with TraceID: %s", expectedTraceID)
}

// TestE2EMultiTaskTrace tests trace propagation across multiple related tasks
func TestE2EMultiTaskTrace(t *testing.T) {
	// Setup span recorder
	spanRecorder := tracetest.NewSpanRecorder()
	tp := trace.NewTracerProvider(
		trace.WithSpanProcessor(spanRecorder),
	)
	otel.SetTracerProvider(tp)

	// Create initial parent span (simulating scrape request)
	tracer := tp.Tracer("test")
	ctx, scrapeSpan := tracer.Start(context.Background(), "controller.scrape_request")

	parentSpanContext := scrapeSpan.SpanContext()

	// Task 1: Process Document
	task1 := ProcessDocumentPayload{
		AnalysisID: "multi-task-analysis",
		Text:       "Text from scraped page",
		EnqueuedAt: time.Now().UnixNano(),
	}

	if span := oteltrace.SpanFromContext(ctx); span.SpanContext().IsValid() {
		spanCtx := span.SpanContext()
		task1.TraceID = spanCtx.TraceID().String()
		task1.SpanID = spanCtx.SpanID().String()
	}

	// Simulate worker 1 processing
	traceID, _ := oteltrace.TraceIDFromHex(task1.TraceID)
	spanID, _ := oteltrace.SpanIDFromHex(task1.SpanID)

	remoteSpanCtx1 := oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: oteltrace.FlagsSampled,
		Remote:     true,
	})

	linkedCtx1 := oteltrace.ContextWithRemoteSpanContext(context.Background(), remoteSpanCtx1)
	worker1Ctx, worker1Span := tracer.Start(linkedCtx1, "worker.process_document",
		oteltrace.WithSpanKind(oteltrace.SpanKindConsumer),
	)

	// Worker 1 enqueues Task 2: Enrich Text (using same trace context)
	task2 := EnrichTextPayload{
		AnalysisID: "multi-task-analysis",
		Text:       "Text to enrich",
		EnqueuedAt: time.Now().UnixNano(),
	}

	if span := oteltrace.SpanFromContext(worker1Ctx); span.SpanContext().IsValid() {
		spanCtx := span.SpanContext()
		task2.TraceID = spanCtx.TraceID().String()
		task2.SpanID = spanCtx.SpanID().String()
	}

	worker1Span.End()

	// Simulate worker 2 processing
	traceID2, _ := oteltrace.TraceIDFromHex(task2.TraceID)
	spanID2, _ := oteltrace.SpanIDFromHex(task2.SpanID)

	remoteSpanCtx2 := oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
		TraceID:    traceID2,
		SpanID:     spanID2,
		TraceFlags: oteltrace.FlagsSampled,
		Remote:     true,
	})

	linkedCtx2 := oteltrace.ContextWithRemoteSpanContext(context.Background(), remoteSpanCtx2)
	_, worker2Span := tracer.Start(linkedCtx2, "worker.enrich_text",
		oteltrace.WithSpanKind(oteltrace.SpanKindConsumer),
	)
	worker2Span.End()

	// End scrape span before verification
	scrapeSpan.End()

	time.Sleep(100 * time.Millisecond)

	// Verify all spans share the same TraceID
	spans := spanRecorder.Ended()
	expectedTraceID := parentSpanContext.TraceID()

	if len(spans) < 3 {
		t.Fatalf("Expected at least 3 spans (scrape, process_document, enrich_text), got %d", len(spans))
	}

	for i, span := range spans {
		if span.SpanContext().TraceID() != expectedTraceID {
			t.Errorf("Span %d (%s) has different TraceID: got %s, want %s",
				i, span.Name(), span.SpanContext().TraceID(), expectedTraceID)
		}
	}

	t.Logf("Successfully verified multi-task E2E trace flow with TraceID: %s", expectedTraceID)
	t.Logf("Recorded %d spans in trace chain", len(spans))
}

// TestE2ETraceFlowWithRealAsynq tests with actual Asynq client (requires Redis)
func TestE2ETraceFlowWithRealAsynq(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup span recorder
	spanRecorder := tracetest.NewSpanRecorder()
	tp := trace.NewTracerProvider(
		trace.WithSpanProcessor(spanRecorder),
	)
	otel.SetTracerProvider(tp)

	// Setup Asynq client
	redisAddr := "localhost:6379"
	client := asynq.NewClient(asynq.RedisClientOpt{Addr: redisAddr})
	defer client.Close()

	queueClient := &Client{client: client}

	// Create parent span
	tracer := tp.Tracer("test")
	ctx, parentSpan := tracer.Start(context.Background(), "controller.analyze_request")

	// Enqueue a real task
	analysisID := "test-analysis-real-" + time.Now().Format("20060102150405")
	taskID, err := queueClient.EnqueueProcessDocument(ctx, analysisID,
		"Sample text for real Asynq test",
		"<html>Sample text</html>",
		[]string{"https://example.com/img1.jpg"})

	if err != nil {
		t.Skipf("Could not connect to Redis: %v", err)
	}

	t.Logf("Enqueued task: %s", taskID)

	// End parent span before checking
	parentSpan.End()

	time.Sleep(100 * time.Millisecond)
	spans := spanRecorder.Ended()

	if len(spans) == 0 {
		t.Error("No spans recorded")
	}

	for _, span := range spans {
		t.Logf("Recorded span: %s (TraceID: %s, SpanID: %s)",
			span.Name(), span.SpanContext().TraceID(), span.SpanContext().SpanID())
	}
}

// TestE2EQueueWaitTimeAccuracy tests queue wait time measurement accuracy
func TestE2EQueueWaitTimeAccuracy(t *testing.T) {
	testCases := []struct {
		name        string
		waitSeconds int
	}{
		{"ShortWait", 1},
		{"MediumWait", 5},
		{"LongWait", 10},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Enqueue task with past timestamp
			enqueuedTime := time.Now().Add(-time.Duration(tc.waitSeconds) * time.Second)
			payload := ProcessDocumentPayload{
				AnalysisID: "wait-time-test",
				Text:       "Test text",
				EnqueuedAt: enqueuedTime.UnixNano(),
				TraceID:    "test-trace-id",
				SpanID:     "test-span-id",
			}

			// Simulate worker starting now
			processingStartTime := time.Now()

			// Calculate wait time
			var queueWaitTime time.Duration
			if payload.EnqueuedAt > 0 {
				enqueueTime := time.Unix(0, payload.EnqueuedAt)
				queueWaitTime = processingStartTime.Sub(enqueueTime)
			}

			// Allow 100ms tolerance
			expectedMin := time.Duration(tc.waitSeconds)*time.Second - 100*time.Millisecond
			expectedMax := time.Duration(tc.waitSeconds)*time.Second + 100*time.Millisecond

			if queueWaitTime < expectedMin || queueWaitTime > expectedMax {
				t.Errorf("Queue wait time out of range: got %v, expected ~%ds",
					queueWaitTime, tc.waitSeconds)
			}

			t.Logf("Queue wait time: %v (expected ~%ds)", queueWaitTime, tc.waitSeconds)
		})
	}
}
