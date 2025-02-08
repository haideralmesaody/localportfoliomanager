package main

import (
	"database/sql"
	"fmt"
	"localportfoliomanager/internal/api"
	"localportfoliomanager/internal/migrations"
	"localportfoliomanager/internal/reporting"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func main() {
	// Initialize DB connection
	db, err := api.InitDB()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Run migrations
	if err := migrations.RunMigrations(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Create and configure server
	server := api.NewServer(db)
	router := server.Router()

	// Set up routes
	reportingService := reporting.NewReportingService(db)
	portfolioHandler := api.NewPortfolioHandler(reportingService, server)

	// Add debug logging
	fmt.Println("Setting up performance report route")
	router.HandleFunc("/api/portfolios/{id}/performance", portfolioHandler.GetPerformanceReport).Methods("GET")

	// Print registered routes for debugging
	server.LogRoutes()

	// Start the server
	log.Printf("Starting server on :8080")
	if err := http.ListenAndServe(":8080", router); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

func setupRoutes(router *mux.Router, db *sql.DB) {
	// Portfolio performance reporting
	reportingService := reporting.NewReportingService(db)
	portfolioHandler := api.NewPortfolioHandler(reportingService)
	router.HandleFunc("/api/portfolios/{id}/performance", portfolioHandler.GetPerformanceReport).Methods("GET")
}
