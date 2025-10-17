package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zombar/textanalyzer/internal/analyzer"
	"github.com/zombar/textanalyzer/internal/api"
	"github.com/zombar/textanalyzer/internal/database"
	"github.com/zombar/textanalyzer/internal/ollama"
)

func main() {
	var (
		port       = flag.String("port", "8080", "Server port")
		dbPath     = flag.String("db", "textanalyzer.db", "Database file path")
		ollamaURL  = flag.String("ollama-url", "http://honker:11434", "Ollama API URL")
		ollamaModel = flag.String("ollama-model", "gpt-oss:20b", "Ollama model to use")
		useOllama  = flag.Bool("use-ollama", true, "Enable Ollama for AI-powered analysis")
	)
	flag.Parse()

	// Initialize database
	db, err := database.New(*dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Run migrations
	if err := db.Migrate(); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Initialize analyzer
	var textAnalyzer *analyzer.Analyzer
	if *useOllama {
		ollamaClient, err := ollama.New(*ollamaURL, *ollamaModel)
		if err != nil {
			log.Printf("Warning: Failed to initialize Ollama client: %v. Falling back to rule-based analysis.", err)
			textAnalyzer = analyzer.New()
		} else {
			log.Printf("Ollama client initialized with model: %s", *ollamaModel)
			textAnalyzer = analyzer.NewWithOllama(ollamaClient)
		}
	} else {
		log.Println("Ollama disabled, using rule-based analysis")
		textAnalyzer = analyzer.New()
	}

	// Initialize API handler
	handler := api.NewHandler(db, textAnalyzer)

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
		log.Printf("Server starting on port %s", *port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped")
}
