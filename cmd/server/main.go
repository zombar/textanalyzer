package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/docutag/platform/pkg/metrics"
	"github.com/docutag/platform/pkg/tracing"
	"github.com/docutag/textanalyzer/internal/analyzer"
	"github.com/docutag/textanalyzer/internal/api"
	"github.com/docutag/textanalyzer/internal/database"
	"github.com/docutag/textanalyzer/internal/ollama"
	"github.com/docutag/textanalyzer/internal/queue"
	"github.com/docutag/textanalyzer/pkg/logging"
)

func main() {
	// Setup structured logging with JSON output
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	logger.Info("textanalyzer service initializing", "version", "1.0.0")

	// Initialize tracing
	tp, err := tracing.InitTracer("docutab-textanalyzer")
	if err != nil {
		logger.Warn("failed to initialize tracer, continuing without tracing", "error", err)
	} else {
		defer func() {
			if err := tp.Shutdown(context.Background()); err != nil {
				logger.Error("error shutting down tracer", "error", err)
			}
		}()
		logger.Info("tracing initialized successfully")
	}

	// Get default values from environment variables, with fallbacks
	portDefault := getEnv("PORT", "8080")
	ollamaURLDefault := getEnv("OLLAMA_URL", "http://localhost:11434")
	ollamaModelDefault := getEnv("OLLAMA_MODEL", "gpt-oss:20b")
	useOllamaDefault := getEnvBool("USE_OLLAMA", true)
	redisAddrDefault := getEnv("REDIS_ADDR", "localhost:6379")
	workerConcurrencyDefault := getEnvInt("WORKER_CONCURRENCY", 5)
	ollamaMaxRetriesDefault := getEnvInt("OLLAMA_MAX_RETRIES", 10)

	// PostgreSQL environment variables
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "docutab")
	dbPassword := getEnv("DB_PASSWORD", "docutab_dev_pass")
	dbName := getEnv("DB_NAME", "docutab")

	var (
		port              = flag.String("port", portDefault, "Server port (env: PORT)")
		ollamaURL         = flag.String("ollama-url", ollamaURLDefault, "Ollama API URL (env: OLLAMA_URL)")
		ollamaModel       = flag.String("ollama-model", ollamaModelDefault, "Ollama model to use (env: OLLAMA_MODEL)")
		useOllama         = flag.Bool("use-ollama", useOllamaDefault, "Enable Ollama for AI-powered analysis (env: USE_OLLAMA)")
		redisAddr         = flag.String("redis-addr", redisAddrDefault, "Redis address for queue (env: REDIS_ADDR)")
		workerConcurrency = flag.Int("worker-concurrency", workerConcurrencyDefault, "Worker concurrency (env: WORKER_CONCURRENCY)")
		ollamaMaxRetries  = flag.Int("ollama-max-retries", ollamaMaxRetriesDefault, "Max retries for Ollama tasks (env: OLLAMA_MAX_RETRIES)")
	)
	flag.Parse()

	// Construct PostgreSQL connection string
	dbConnStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)
	logger.Info("using PostgreSQL database", "host", dbHost, "port", dbPort, "database", dbName)

	// Initialize database
	db, err := database.New(dbConnStr)
	if err != nil {
		logger.Error("failed to initialize database", "error", err, "connection_string", dbConnStr)
		os.Exit(1)
	}
	defer db.Close()

	// Run migrations
	if err := db.Migrate(); err != nil {
		logger.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	// Initialize database metrics
	dbMetrics := metrics.NewDatabaseMetrics("textanalyzer")
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			dbMetrics.UpdateDBStats(db.Conn())
		}
	}()
	logger.Info("database metrics initialized")

	// Initialize analyzer
	var textAnalyzer *analyzer.Analyzer
	if *useOllama {
		ollamaClient, err := ollama.New(*ollamaURL, *ollamaModel)
		if err != nil {
			logger.Warn("failed to initialize Ollama client, falling back to rule-based analysis",
				"error", err,
				"ollama_url", *ollamaURL,
				"ollama_model", *ollamaModel,
			)
			textAnalyzer = analyzer.New()
		} else {
			logger.Info("Ollama client initialized", "model", *ollamaModel, "url", *ollamaURL)
			textAnalyzer = analyzer.NewWithOllama(ollamaClient)
		}
	} else {
		logger.Info("Ollama disabled, using rule-based analysis")
		textAnalyzer = analyzer.New()
	}

	// Initialize queue client
	queueClient := queue.NewClient(queue.ClientConfig{
		RedisAddr: *redisAddr,
	})
	logger.Info("queue client initialized", "redis_addr", *redisAddr)

	// Initialize queue worker
	queueWorker := queue.NewWorker(
		queue.WorkerConfig{
			RedisAddr:   *redisAddr,
			Concurrency: *workerConcurrency,
			MaxRetries:  *ollamaMaxRetries,
		},
		db,
		textAnalyzer,
		queueClient,
	)

	// Start worker in a goroutine
	go func() {
		logger.Info("starting queue worker")
		if err := queueWorker.Start(); err != nil {
			logger.Error("queue worker error", "error", err)
			os.Exit(1)
		}
	}()

	// Initialize API handler with queue client
	apiHandler := api.NewHandler(db, textAnalyzer, queueClient)

	// Setup server with middleware chain (applied bottom-up, executes top-down):
	// Execution order: tracing -> metrics -> logging -> handlers
	// This ensures tracing creates span BEFORE logging tries to read trace context
	var handler http.Handler = apiHandler

	// Add HTTP request logging (innermost, executes last)
	handler = logging.HTTPLoggingMiddleware(logger)(handler)

	// Add HTTP metrics middleware
	handler = metrics.HTTPMiddleware("textanalyzer")(handler)

	// Wrap with tracing middleware (outermost, executes first)
	handler = tracing.HTTPMiddleware("textanalyzer")(handler)

	// Create server with extended timeouts for AI processing
	srv := &http.Server{
		Addr:         ":" + *port,
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 420 * time.Second, // 7 minutes for AI analysis
		IdleTimeout:  120 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		logger.Info("textanalyzer service starting",
			"port", *port,
			"db_host", dbHost,
			"db_name", dbName,
			"ollama_enabled", *useOllama,
			"ollama_url", *ollamaURL,
			"ollama_model", *ollamaModel,
		)

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server failed to start", "error", err)
			os.Exit(1)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server and worker")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown queue worker
	queueWorker.Shutdown()
	logger.Info("queue worker stopped")

	// Close queue client
	if err := queueClient.Close(); err != nil {
		logger.Error("error closing queue client", "error", err)
	}

	// Shutdown HTTP server
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("server forced to shutdown", "error", err)
		os.Exit(1)
	}

	logger.Info("server stopped")
}

// getEnv retrieves an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvBool retrieves a boolean environment variable or returns a default value
func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return value == "true" || value == "1" || value == "yes"
	}
	return defaultValue
}

// getEnvInt retrieves an integer environment variable or returns a default value
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
