package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zombar/purpletab/pkg/metrics"
	"github.com/zombar/purpletab/pkg/tracing"
	"github.com/zombar/textanalyzer/internal/analyzer"
	"github.com/zombar/textanalyzer/internal/api"
	"github.com/zombar/textanalyzer/internal/database"
	"github.com/zombar/textanalyzer/internal/ollama"
	"github.com/zombar/textanalyzer/pkg/logging"
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
	dbPathDefault := getEnv("DB_PATH", "textanalyzer.db")
	ollamaURLDefault := getEnv("OLLAMA_URL", "http://localhost:11434")
	ollamaModelDefault := getEnv("OLLAMA_MODEL", "gpt-oss:20b")
	useOllamaDefault := getEnvBool("USE_OLLAMA", true)

	var (
		port        = flag.String("port", portDefault, "Server port (env: PORT)")
		dbPath      = flag.String("db", dbPathDefault, "Database file path (env: DB_PATH)")
		ollamaURL   = flag.String("ollama-url", ollamaURLDefault, "Ollama API URL (env: OLLAMA_URL)")
		ollamaModel = flag.String("ollama-model", ollamaModelDefault, "Ollama model to use (env: OLLAMA_MODEL)")
		useOllama   = flag.Bool("use-ollama", useOllamaDefault, "Enable Ollama for AI-powered analysis (env: USE_OLLAMA)")
	)
	flag.Parse()

	// Initialize database
	db, err := database.New(*dbPath)
	if err != nil {
		logger.Error("failed to initialize database", "error", err, "database_path", *dbPath)
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

	// Initialize API handler
	apiHandler := api.NewHandler(db, textAnalyzer)

	// Wrap handler with middleware chain: HTTP logging -> tracing -> handlers
	handler := logging.HTTPLoggingMiddleware(logger)(
		tracing.HTTPMiddleware("textanalyzer")(apiHandler),
	)

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
			"database", *dbPath,
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

	logger.Info("shutting down server")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

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
