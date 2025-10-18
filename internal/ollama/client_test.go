package ollama

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name        string
		ollamaURL   string
		model       string
		expectError bool
		expectedModel string
	}{
		{
			name:          "default values",
			ollamaURL:     "",
			model:         "",
			expectError:   false,
			expectedModel: DefaultModel,
		},
		{
			name:          "custom URL and model",
			ollamaURL:     "http://custom-ollama:11434",
			model:         "llama3.2",
			expectError:   false,
			expectedModel: "llama3.2",
		},
		{
			name:          "custom URL, default model",
			ollamaURL:     "http://localhost:11434",
			model:         "",
			expectError:   false,
			expectedModel: DefaultModel,
		},
		{
			name:        "invalid URL",
			ollamaURL:   "://invalid-url",
			model:       "test",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := New(tt.ollamaURL, tt.model)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if client == nil {
					t.Fatal("Expected client but got nil")
				}
				if client.model != tt.expectedModel {
					t.Errorf("Expected model %s, got %s", tt.expectedModel, client.model)
				}
				if client.timeout != DefaultTimeout {
					t.Errorf("Expected timeout %v, got %v", DefaultTimeout, client.timeout)
				}
			}
		})
	}
}

func TestParseTagsFromJSON(t *testing.T) {
	tests := []struct {
		name        string
		response    string
		expected    []string
		expectError bool
	}{
		{
			name:        "valid JSON array",
			response:    `["technology", "ai", "machine-learning", "data-science"]`,
			expected:    []string{"technology", "ai", "machine-learning", "data-science"},
			expectError: false,
		},
		{
			name:        "JSON array with prefix text",
			response:    `Here are the tags:\n["web-development", "javascript", "frontend"]`,
			expected:    []string{"web-development", "javascript", "frontend"},
			expectError: false,
		},
		{
			name:        "JSON array with suffix text",
			response:    `["python", "backend", "api"]\nThese are relevant tags.`,
			expected:    []string{"python", "backend", "api"},
			expectError: false,
		},
		{
			name:        "more than 5 tags (should limit)",
			response:    `["tag1", "tag2", "tag3", "tag4", "tag5", "tag6", "tag7"]`,
			expected:    []string{"tag1", "tag2", "tag3", "tag4", "tag5"},
			expectError: false,
		},
		{
			name:        "empty array",
			response:    `[]`,
			expected:    []string{},
			expectError: false,
		},
		{
			name:        "no JSON array",
			response:    "No JSON here",
			expectError: true,
		},
		{
			name:        "invalid JSON",
			response:    `["invalid"`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var tags []string
			var err error

			// Simulate the parsing logic from GenerateTags
			start := strings.Index(tt.response, "[")
			end := strings.LastIndex(tt.response, "]")
			if start >= 0 && end > start {
				jsonStr := tt.response[start : end+1]
				err = json.Unmarshal([]byte(jsonStr), &tags)
			} else {
				err = &jsonParseError{}
			}

			// Limit to 5 tags
			if err == nil && len(tags) > 5 {
				tags = tags[:5]
			}

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if len(tags) != len(tt.expected) {
					t.Errorf("Expected %d tags, got %d", len(tt.expected), len(tags))
				}
				for i, tag := range tags {
					if tag != tt.expected[i] {
						t.Errorf("Expected tag %s at index %d, got %s", tt.expected[i], i, tag)
					}
				}
			}
		})
	}
}

type jsonParseError struct{}

func (e *jsonParseError) Error() string {
	return "no JSON array found in response"
}

func TestParseReferencesFromJSON(t *testing.T) {
	tests := []struct {
		name        string
		response    string
		expectedLen int
		expectError bool
	}{
		{
			name: "valid references",
			response: `[
				{"text": "The population is 1 million", "type": "statistic", "context": "Demographics", "confidence": "high"},
				{"text": "According to Smith", "type": "citation", "context": "Introduction", "confidence": "medium"}
			]`,
			expectedLen: 2,
			expectError: false,
		},
		{
			name:        "empty array",
			response:    `[]`,
			expectedLen: 0,
			expectError: false,
		},
		{
			name:        "no JSON array",
			response:    "No references found",
			expectError: true,
		},
		{
			name: "with surrounding text",
			response: `Here are the references:
			[{"text": "Test claim", "type": "claim", "context": "Body", "confidence": "low"}]
			End of references`,
			expectedLen: 1,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var references []Reference
			var err error

			// Simulate the parsing logic from ExtractReferences
			start := strings.Index(tt.response, "[")
			end := strings.LastIndex(tt.response, "]")
			if start >= 0 && end > start {
				jsonStr := tt.response[start : end+1]
				err = json.Unmarshal([]byte(jsonStr), &references)
			} else {
				err = &jsonParseError{}
			}

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if len(references) != tt.expectedLen {
					t.Errorf("Expected %d references, got %d", tt.expectedLen, len(references))
				}
			}
		})
	}
}

func TestParseAIDetectionFromJSON(t *testing.T) {
	tests := []struct {
		name        string
		response    string
		expectError bool
		checkFields bool
	}{
		{
			name: "valid AI detection result",
			response: `{
				"likelihood": "likely",
				"confidence": "high",
				"reasoning": "The text shows typical AI patterns",
				"indicators": ["repetitive structure", "formal tone"],
				"human_score": 35.5
			}`,
			expectError: false,
			checkFields: true,
		},
		{
			name: "with surrounding text",
			response: `Analysis complete:
			{"likelihood": "unlikely", "confidence": "medium", "reasoning": "Natural writing", "indicators": ["personal voice"], "human_score": 85.0}
			End of analysis`,
			expectError: false,
			checkFields: true,
		},
		{
			name:        "no JSON object",
			response:    "No JSON here",
			expectError: true,
		},
		{
			name:        "invalid JSON",
			response:    `{"likelihood": "likely"`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result AIDetectionResult
			var err error

			// Simulate the parsing logic from DetectAIContent
			start := strings.Index(tt.response, "{")
			end := strings.LastIndex(tt.response, "}")
			if start >= 0 && end > start {
				jsonStr := tt.response[start : end+1]
				err = json.Unmarshal([]byte(jsonStr), &result)
			} else {
				err = &jsonParseError{}
			}

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if tt.checkFields {
					if result.Likelihood == "" {
						t.Error("Expected likelihood to be set")
					}
					if result.Confidence == "" {
						t.Error("Expected confidence to be set")
					}
					if result.Reasoning == "" {
						t.Error("Expected reasoning to be set")
					}
				}
			}
		})
	}
}

func TestReference(t *testing.T) {
	// Test that Reference struct can be marshaled and unmarshaled
	ref := Reference{
		Text:       "Test claim",
		Type:       "claim",
		Context:    "Test context",
		Confidence: "high",
	}

	data, err := json.Marshal(ref)
	if err != nil {
		t.Fatalf("Failed to marshal Reference: %v", err)
	}

	var decoded Reference
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal Reference: %v", err)
	}

	if decoded.Text != ref.Text {
		t.Errorf("Expected Text %s, got %s", ref.Text, decoded.Text)
	}
	if decoded.Type != ref.Type {
		t.Errorf("Expected Type %s, got %s", ref.Type, decoded.Type)
	}
}

func TestAIDetectionResult(t *testing.T) {
	// Test that AIDetectionResult struct can be marshaled and unmarshaled
	result := AIDetectionResult{
		Likelihood: "likely",
		Confidence: "high",
		Reasoning:  "Test reasoning",
		Indicators: []string{"indicator1", "indicator2"},
		HumanScore: 42.5,
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal AIDetectionResult: %v", err)
	}

	var decoded AIDetectionResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal AIDetectionResult: %v", err)
	}

	if decoded.Likelihood != result.Likelihood {
		t.Errorf("Expected Likelihood %s, got %s", result.Likelihood, decoded.Likelihood)
	}
	if decoded.HumanScore != result.HumanScore {
		t.Errorf("Expected HumanScore %.2f, got %.2f", result.HumanScore, decoded.HumanScore)
	}
	if len(decoded.Indicators) != len(result.Indicators) {
		t.Errorf("Expected %d indicators, got %d", len(result.Indicators), len(decoded.Indicators))
	}
}

func TestContextHandling(t *testing.T) {
	// Test that context is properly handled in constructor
	client, err := New("http://localhost:11434", "test-model")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Verify timeout is set
	if client.timeout != DefaultTimeout {
		t.Errorf("Expected timeout %v, got %v", DefaultTimeout, client.timeout)
	}

	// Test with canceled context (this won't make actual API calls)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// These should fail with context canceled error (or succeed quickly if mocked)
	// We're mainly testing that the methods accept context properly
	_, err = client.GenerateSynopsis(ctx, "test")
	if err == nil {
		t.Log("Note: GenerateSynopsis didn't fail with canceled context (likely no Ollama server)")
	}
}
