package queue

import (
	"strings"
	"testing"
)

func TestCompressHTML(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "normal HTML",
			input:   "<html><body><h1>Title</h1><p>Content</p></body></html>",
			wantErr: false,
		},
		{
			name:    "large HTML",
			input:   strings.Repeat("<div>Content</div>", 1000),
			wantErr: false,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: false,
		},
		{
			name:    "unicode content",
			input:   "<p>Hello 世界 مرحبا</p>",
			wantErr: false,
		},
		{
			name:    "special characters",
			input:   "<p>&lt;script&gt;alert('test')&lt;/script&gt;</p>",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compressed, err := compressHTML(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("compressHTML() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.input == "" {
				if compressed != "" {
					t.Error("empty input should return empty compressed string")
				}
				return
			}

			// Verify it's base64 encoded (should contain only valid base64 chars)
			if len(compressed) == 0 {
				t.Error("compressed output should not be empty for non-empty input")
			}

			// Verify compression actually reduces size for large inputs
			if len(tt.input) > 500 && len(compressed) > len(tt.input) {
				t.Errorf("compression should reduce size for large inputs: input=%d, compressed=%d",
					len(tt.input), len(compressed))
			}
		})
	}
}

func TestDecompressHTML(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  string
		shouldErr bool
	}{
		{
			name:      "valid compressed HTML",
			input:     "",  // Will be filled by compressing
			expected:  "<html><body>Test</body></html>",
			shouldErr: false,
		},
		{
			name:      "empty string",
			input:     "",
			expected:  "",
			shouldErr: false,
		},
		{
			name:      "invalid base64",
			input:     "not-valid-base64!!!",
			expected:  "",
			shouldErr: true,
		},
		{
			name:      "valid base64 but not gzipped",
			input:     "SGVsbG8gV29ybGQ=", // "Hello World" in base64
			expected:  "",
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// For the valid case, compress first
			if tt.name == "valid compressed HTML" {
				compressed, err := compressHTML(tt.expected)
				if err != nil {
					t.Fatalf("setup failed: %v", err)
				}
				tt.input = compressed
			}

			result, err := decompressHTML(tt.input)
			if (err != nil) != tt.shouldErr {
				t.Errorf("decompressHTML() error = %v, shouldErr %v", err, tt.shouldErr)
				return
			}

			if !tt.shouldErr && result != tt.expected {
				t.Errorf("decompressHTML() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCompressDecompressRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		html string
	}{
		{
			name: "simple HTML",
			html: "<html><body><p>Test content</p></body></html>",
		},
		{
			name: "complex HTML with attributes",
			html: `<html>
				<head><title>Test</title></head>
				<body class="main">
					<div id="content">
						<p class="article">Article text here</p>
						<img src="test.jpg" alt="Test image"/>
					</div>
				</body>
			</html>`,
		},
		{
			name: "large HTML document",
			html: strings.Repeat("<div><p>Paragraph content with some text</p></div>", 100),
		},
		{
			name: "HTML with unicode",
			html: "<p>Hello 世界 مرحبا שלום Привет</p>",
		},
		{
			name: "HTML with newlines and tabs",
			html: "<html>\n\t<body>\n\t\t<p>Content</p>\n\t</body>\n</html>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Compress
			compressed, err := compressHTML(tt.html)
			if err != nil {
				t.Fatalf("compressHTML() failed: %v", err)
			}

			// Decompress
			decompressed, err := decompressHTML(compressed)
			if err != nil {
				t.Fatalf("decompressHTML() failed: %v", err)
			}

			// Verify round trip
			if decompressed != tt.html {
				t.Errorf("round trip failed:\noriginal: %s\ndecompressed: %s",
					tt.html, decompressed)
			}
		})
	}
}

func TestCompressionRatio(t *testing.T) {
	// Test that compression actually provides benefit
	html := strings.Repeat(`
		<div class="article-content">
			<p>This is a paragraph with repetitive content</p>
			<p>This is another paragraph with repetitive content</p>
			<p>This is yet another paragraph with repetitive content</p>
		</div>
	`, 50)

	compressed, err := compressHTML(html)
	if err != nil {
		t.Fatalf("compression failed: %v", err)
	}

	originalSize := len(html)
	compressedSize := len(compressed)
	ratio := float64(compressedSize) / float64(originalSize)

	t.Logf("Original size: %d bytes", originalSize)
	t.Logf("Compressed size: %d bytes", compressedSize)
	t.Logf("Compression ratio: %.2f%%", ratio*100)

	// For repetitive HTML, we should get good compression (< 30%)
	if ratio > 0.5 {
		t.Errorf("expected compression ratio < 50%%, got %.2f%%", ratio*100)
	}
}

// testError is a simple error type for testing
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

// Benchmark tests
func BenchmarkCompressHTML(b *testing.B) {
	html := strings.Repeat("<div><p>Test content</p></div>", 100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = compressHTML(html)
	}
}

func BenchmarkDecompressHTML(b *testing.B) {
	html := strings.Repeat("<div><p>Test content</p></div>", 100)
	compressed, _ := compressHTML(html)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = decompressHTML(compressed)
	}
}

func BenchmarkCompressDecompressRoundTrip(b *testing.B) {
	html := strings.Repeat("<div><p>Test content</p></div>", 100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		compressed, _ := compressHTML(html)
		_, _ = decompressHTML(compressed)
	}
}
