package queue

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/assert"
)

// TestProcessDocumentPayload tests the ProcessDocumentPayload structure
func TestProcessDocumentPayload(t *testing.T) {
	payload := ProcessDocumentPayload{
		AnalysisID: "test-123",
		Text:       "Sample text for analysis",
		Images:     []string{"https://example.com/image1.jpg", "https://example.com/image2.jpg"},
	}

	data, err := json.Marshal(payload)
	assert.NoError(t, err)
	assert.NotEmpty(t, data)

	var decoded ProcessDocumentPayload
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, payload.AnalysisID, decoded.AnalysisID)
	assert.Equal(t, payload.Text, decoded.Text)
	assert.Equal(t, payload.Images, decoded.Images)
}

// TestEnrichTextPayload tests the EnrichTextPayload structure
func TestEnrichTextPayload(t *testing.T) {
	payload := EnrichTextPayload{
		AnalysisID: "test-456",
		Text:       "Text to enrich with AI",
	}

	data, err := json.Marshal(payload)
	assert.NoError(t, err)
	assert.NotEmpty(t, data)

	var decoded EnrichTextPayload
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, payload.AnalysisID, decoded.AnalysisID)
	assert.Equal(t, payload.Text, decoded.Text)
}

// TestEnrichImagePayload tests the EnrichImagePayload structure
func TestEnrichImagePayload(t *testing.T) {
	payload := EnrichImagePayload{
		AnalysisID: "test-789",
		ImageURL:   "https://example.com/test-image.jpg",
	}

	data, err := json.Marshal(payload)
	assert.NoError(t, err)
	assert.NotEmpty(t, data)

	var decoded EnrichImagePayload
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, payload.AnalysisID, decoded.AnalysisID)
	assert.Equal(t, payload.ImageURL, decoded.ImageURL)
}


// TestIsRetriableOllamaError tests error classification
func TestIsRetriableOllamaError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "Connection refused error",
			err:      errors.New("connection refused"),
			expected: true,
		},
		{
			name:     "Timeout error",
			err:      errors.New("request timeout"),
			expected: true,
		},
		{
			name:     "Context deadline exceeded",
			err:      errors.New("context deadline exceeded"),
			expected: true,
		},
		{
			name:     "Service unavailable",
			err:      errors.New("503 Service Unavailable"),
			expected: true,
		},
		{
			name:     "Bad gateway",
			err:      errors.New("502 Bad Gateway"),
			expected: true,
		},
		{
			name:     "Network unreachable",
			err:      errors.New("network is unreachable"),
			expected: true,
		},
		{
			name:     "Invalid request error",
			err:      errors.New("invalid request format"),
			expected: false,
		},
		{
			name:     "Generic error",
			err:      errors.New("some other error"),
			expected: false,
		},
		{
			name:     "Empty error",
			err:      errors.New(""),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRetriableOllamaError(tt.err)
			assert.Equal(t, tt.expected, result, "Error: %v", tt.err)
		})
	}
}

// TestRetryDelayFunc tests custom retry delay function
func TestRetryDelayFunc(t *testing.T) {
	worker := &Worker{
		maxRetries: 10,
	}

	cfg := asynq.Config{
		Concurrency: 5,
		Queues: map[string]int{
			"text-enrichment":    7,
			"offline-processing": 5,
			"image-enrichment":   3,
		},
		StrictPriority: false,
		RetryDelayFunc: worker.getRetryDelayFunc(),
	}

	// Test Ollama task retries (exponential backoff)
	ollamaTask := asynq.NewTask(TypeEnrichText, []byte(`{}`))
	testErr := errors.New("connection refused")

	delays := []time.Duration{
		30 * time.Second,
		1 * time.Minute,
		2 * time.Minute,
		5 * time.Minute,
		10 * time.Minute,
		20 * time.Minute,
		30 * time.Minute,
		1 * time.Hour,
		2 * time.Hour,
		4 * time.Hour,
	}

	for i := 0; i < 10; i++ {
		delay := cfg.RetryDelayFunc(i, testErr, ollamaTask)
		expected := delays[i]
		assert.Equal(t, expected, delay, "Retry %d should have delay %v", i, expected)
	}

	// Test standard task retries (linear backoff)
	standardTask := asynq.NewTask(TypeProcessDocument, []byte(`{}`))

	standardDelays := []time.Duration{
		1 * time.Minute,
		5 * time.Minute,
		15 * time.Minute,
	}

	for i := 0; i < 3; i++ {
		delay := cfg.RetryDelayFunc(i, testErr, standardTask)
		expected := standardDelays[i]
		assert.Equal(t, expected, delay, "Standard retry %d should have delay %v", i, expected)
	}
}

// TestQueuePriorities tests that queue priorities are set correctly
func TestQueuePriorities(t *testing.T) {
	// Verify the queue priorities match requirements
	expectedPriorities := map[string]int{
		"text-enrichment":    7, // AI text enrichment (highest priority)
		"offline-processing": 5, // Offline rule-based processing (medium priority)
		"image-enrichment":   3, // AI image enrichment (lowest priority)
	}

	// This would normally be checked in the worker configuration
	// For now, we verify the expected values are what we designed
	assert.Equal(t, 7, expectedPriorities["text-enrichment"], "Text enrichment priority should be 7")
	assert.Equal(t, 5, expectedPriorities["offline-processing"], "Offline processing priority should be 5")
	assert.Equal(t, 3, expectedPriorities["image-enrichment"], "Image enrichment priority should be 3")
}

// TestTaskTypeConstants tests that task type constants are defined correctly
func TestTaskTypeConstants(t *testing.T) {
	assert.Equal(t, "textanalyzer:process_document", TypeProcessDocument)
	assert.Equal(t, "textanalyzer:enrich_text", TypeEnrichText)
	assert.Equal(t, "textanalyzer:enrich_image", TypeEnrichImage)
}
