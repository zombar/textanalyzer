package queue

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/hibiken/asynq"
	"github.com/docutag/platform/pkg/metrics"
	"github.com/docutag/textanalyzer/internal/analyzer"
	"github.com/docutag/textanalyzer/internal/database"
)

// Worker wraps the Asynq server for processing tasks
type Worker struct {
	server          *asynq.Server
	mux             *asynq.ServeMux
	db              *database.DB
	analyzer        *analyzer.Analyzer
	queueClient     *Client
	concurrency     int
	maxRetries      int
	logger          *slog.Logger
	businessMetrics *metrics.BusinessMetrics
}

// WorkerConfig contains configuration for the queue worker
type WorkerConfig struct {
	RedisAddr   string
	Concurrency int
	MaxRetries  int
}

// NewWorker creates a new queue worker
func NewWorker(
	cfg WorkerConfig,
	db *database.DB,
	analyzer *analyzer.Analyzer,
	queueClient *Client,
) *Worker {
	redisOpt := asynq.RedisClientOpt{
		Addr: cfg.RedisAddr,
	}

	serverCfg := asynq.Config{
		// Concurrency determines how many tasks can be processed simultaneously
		Concurrency: cfg.Concurrency,

		// Queue priority: higher value = higher priority
		// Named queues for clarity: text enrichment gets highest priority, then offline processing, then images
		Queues: map[string]int{
			"text-enrichment":     7, // AI text enrichment with Ollama (highest priority)
			"offline-processing":  5, // Offline rule-based document processing (medium priority)
			"image-enrichment":    3, // AI image enrichment with Ollama (lowest priority)
		},

		// StrictPriority: false means queues are processed proportionally
		// true would mean text-enrichment queue must be empty before processing offline-processing
		StrictPriority: false,

		// Retry configuration with aggressive backoff for Ollama tasks
		RetryDelayFunc: func(n int, err error, task *asynq.Task) time.Duration {
			// Check if this is an Ollama enrichment task
			if task.Type() == TypeEnrichText || task.Type() == TypeEnrichImage {
				// Exponential backoff with jitter for Ollama tasks
				// 30s, 1m, 2m, 5m, 10m, 20m, 30m, 1h, 2h, 4h
				// Total retry window: ~7.5 hours
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
				if n < len(delays) {
					return delays[n]
				}
				return delays[len(delays)-1]
			}

			// Standard retry for offline processing tasks: 1m, 5m, 15m
			delays := []time.Duration{
				1 * time.Minute,
				5 * time.Minute,
				15 * time.Minute,
			}
			if n < len(delays) {
				return delays[n]
			}
			return delays[len(delays)-1]
		},

		// Graceful shutdown timeout
		ShutdownTimeout: 30 * time.Second,

		// Error handler for logging
		ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
			retried, _ := asynq.GetRetryCount(ctx)
			maxRetry, _ := asynq.GetMaxRetry(ctx)

			slog.Error("task processing error",
				"task_type", task.Type(),
				"error", err,
				"retry_count", retried,
				"max_retries", maxRetry,
			)
		}),
	}

	server := asynq.NewServer(redisOpt, serverCfg)
	mux := asynq.NewServeMux()

	// Initialize business metrics
	businessMetrics := metrics.NewBusinessMetrics("textanalyzer")

	w := &Worker{
		server:          server,
		mux:             mux,
		db:              db,
		analyzer:        analyzer,
		queueClient:     queueClient,
		concurrency:     cfg.Concurrency,
		maxRetries:      cfg.MaxRetries,
		logger:          slog.Default(),
		businessMetrics: businessMetrics,
	}

	// Register task handlers
	w.registerHandlers()

	return w
}

// registerHandlers registers all task handlers with the worker
func (w *Worker) registerHandlers() {
	w.mux.HandleFunc(TypeProcessDocument, w.handleProcessDocument)
	w.mux.HandleFunc(TypeEnrichText, w.handleEnrichText)
	w.mux.HandleFunc(TypeEnrichImage, w.handleEnrichImage)
}

// Start starts the worker to begin processing tasks
func (w *Worker) Start() error {
	w.logger.Info("starting asynq worker",
		"concurrency", w.concurrency,
		"queues", map[string]int{"text-enrichment": 7, "offline-processing": 5, "image-enrichment": 3},
		"ollama_max_retries", w.maxRetries,
	)

	// Run is blocking - starts processing tasks
	if err := w.server.Run(w.mux); err != nil {
		return fmt.Errorf("asynq server error: %w", err)
	}

	return nil
}

// Shutdown gracefully shuts down the worker
func (w *Worker) Shutdown() {
	w.logger.Info("shutting down asynq worker")
	w.server.Shutdown()
}

// Server returns the underlying Asynq server (for testing)
func (w *Worker) Server() *asynq.Server {
	return w.server
}

// getRetryDelayFunc returns the retry delay function (for testing)
func (w *Worker) getRetryDelayFunc() func(n int, err error, task *asynq.Task) time.Duration {
	return func(n int, err error, task *asynq.Task) time.Duration {
		// Check if this is an Ollama enrichment task
		if task.Type() == TypeEnrichText || task.Type() == TypeEnrichImage {
			// Exponential backoff for Ollama tasks
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
			if n < len(delays) {
				return delays[n]
			}
			return delays[len(delays)-1]
		}

		// Standard retry for other tasks
		delays := []time.Duration{
			1 * time.Minute,
			5 * time.Minute,
			15 * time.Minute,
		}
		if n < len(delays) {
			return delays[n]
		}
		return delays[len(delays)-1]
	}
}
