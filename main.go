package main

import (
	"context"
	"database/sql"
	"localportfoliomanager/internal/api"
	"localportfoliomanager/internal/utils"
	"localportfoliomanager/scraper"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/chromedp/chromedp"
	_ "github.com/lib/pq"
)

func main() {
	// Initialize logger
	logger := utils.NewAppLogger()

	// Load configuration from new location
	config, err := utils.LoadConfig("configs")
	if err != nil {
		logger.Error("Error loading config: %v", err)
		os.Exit(1)
	}

	// Initialize ChromeDP context
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	// Initialize the scraper
	scraper := scraper.NewScraper(logger, ctx, cancel, config)

	// Initialize database connection
	db, err := sql.Open("postgres", config.Database.DSN)
	if err != nil {
		logger.Error("Error connecting to database: %v", err)
		os.Exit(1)
	}
	defer db.Close()

	// Test database connection
	err = db.Ping()
	if err != nil {
		logger.Error("Error pinging database: %v", err)
		os.Exit(1)
	}

	logger.Info("Connected to database successfully")

	// Create and start the server with the scraper instance
	server := api.NewServer(logger, config, db, scraper)

	// Create channel to listen for interrupt signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Start server in a goroutine
	go func() {
		if err := server.Start(); err != nil && err != http.ErrServerClosed {
			logger.Error("Error starting server: %v", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	<-stop
	logger.Info("Shutting down server...")

	// Create shutdown context with timeout
	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Error during server shutdown: %v", err)
	}

	logger.Info("Server stopped")
}
