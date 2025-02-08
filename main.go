package main

import (
	"context"
	"database/sql"
	"fmt"

	"localportfoliomanager/internal/api"
	"localportfoliomanager/internal/utils"
	"localportfoliomanager/scraper"

	"github.com/chromedp/chromedp"
	_ "github.com/lib/pq"
)

func main() {
	// Initialize logger
	logger := utils.NewAppLogger()

	// Load configuration from new location
	config, err := utils.LoadConfig("configs")
	if err != nil {
		fmt.Println("Error loading config:", err)
	}

	// Initialize ChromeDP context
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	// Initialize the scraper
	scraper := scraper.NewScraper(logger, ctx, cancel, config)

	// Initialize database connection
	db, err := sql.Open("postgres", config.Database.DSN)
	if err != nil {
		//print the error
		fmt.Println("Error connecting to database:", err)
	}
	defer db.Close()

	// Test database connection
	err = db.Ping()
	if err != nil {
		//print the error
		fmt.Println("Error pinging database:", err)
	}

	logger.Info("Connected to database successfully")

	// Create and start the server with the scraper instance
	server := api.NewServer(logger, config, db, scraper)
	if err := server.Start(); err != nil {
		//print the error
		fmt.Println("Error starting server:", err)
	}
}
