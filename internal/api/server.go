package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"localportfoliomanager/internal/reporting"
	"localportfoliomanager/internal/utils"
	"localportfoliomanager/scraper"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gorilla/mux"
)

// Server represents the API server instance
// It handles HTTP requests and manages connections to the database
type Server struct {
	router  *mux.Router      // HTTP request router
	logger  *utils.AppLogger // Application logger
	config  *utils.Config    // Application configuration
	db      *sql.DB          // Database connection
	scraper *scraper.Scraper
	ctx     context.Context
}

// NewServer creates and initializes a new API server instance
// It sets up the database connection and configures the HTTP router
//
// Parameters:
//   - logger: Application logger for recording server activities
//   - config: Application configuration including database and server settings
//   - db: Database connection
//   - scraper: Scraper instance for stock data scraping
//
// Returns:
//   - *Server: Initialized server instance
//   - The function will call logger.Fatal if database connection fails
func NewServer(logger *utils.AppLogger, config *utils.Config, db *sql.DB, scraper *scraper.Scraper) *Server {
	server := &Server{
		router:  mux.NewRouter(),
		logger:  logger,
		config:  config,
		db:      db,
		scraper: scraper,
		ctx:     context.Background(),
	}

	// Create reporting service and handler
	reportingService := reporting.NewReportingService(db)
	reportingHandler := reporting.NewReportingHandler(reportingService)

	server.setupRouter()
	server.setupRoutes(reportingHandler)
	server.verifyRoutes()
	server.startStockUpdater()
	return server
}

// setupRoutes configures APIs for the server.
func (s *Server) setupRoutes(reportingHandler *reporting.ReportingHandler) {
	s.logger.Debug("Setting up routes...")

	// Create API subrouter
	apiRouter := s.router.PathPrefix("/api").Subrouter()

	// Test endpoint
	apiRouter.HandleFunc("/test", s.TestConnection).Methods("GET")
	s.logger.Debug("Registered route: GET /api/test")

	// Create stocks subrouter with better path handling
	stocksRouter := apiRouter.PathPrefix("/stocks").Subrouter()

	// Define routes with explicit paths
	routes := []struct {
		path    string
		handler http.HandlerFunc
		methods []string
	}{
		{"/latest", s.GetLatestStockPrices, []string{"GET"}},
		{"/{ticker}/prices", s.GetStockPrices, []string{"GET"}},
		{"/{ticker}/sparkline", s.GetStockSparkline, []string{"GET"}},
		{"/{ticker}/chart", s.GetStockChartData, []string{"GET"}},
		{"/{ticker}", s.GetStockDetails, []string{"GET"}},
		{"/", s.GetStocks, []string{"GET"}},
	}

	// Register routes and log them
	for _, route := range routes {
		stocksRouter.HandleFunc(route.path, route.handler).Methods(route.methods...)
		s.logger.Debug("Registered route: %s /api/stocks%s", route.methods[0], route.path)
	}

	// Portfolio routes
	portfolioRouter := apiRouter.PathPrefix("/portfolios").Subrouter()
	portfolioRouter.HandleFunc("", s.ListPortfolios).Methods("GET")
	portfolioRouter.HandleFunc("", s.CreatePortfolio).Methods("POST")
	portfolioRouter.HandleFunc("/{id}", s.GetPortfolio).Methods("GET")
	portfolioRouter.HandleFunc("/{id}", s.DeletePortfolio).Methods("DELETE")
	portfolioRouter.HandleFunc("/{id}/rename", s.RenamePortfolio).Methods("PUT")
	portfolioRouter.HandleFunc("/{id}/holdings", s.GetPortfolioHoldings).Methods("GET")

	// Add these transaction routes
	portfolioRouter.HandleFunc("/{id}/transactions", s.GetTransactions).Methods("GET")
	portfolioRouter.HandleFunc("/{id}/transactions", s.CreateTransaction).Methods("POST")

	s.logger.Debug("Registered route: GET /api/portfolios/{id}/transactions")
	s.logger.Debug("Registered route: POST /api/portfolios/{id}/transactions")

	s.logger.Info("Portfolio routes registered")

	// Add CORS middleware
	s.router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	})

	s.logger.Info("Routes setup completed")

	// Add logging middleware
	s.router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			s.logger.Debug("Request started: %s %s", r.Method, r.URL.Path)
			next.ServeHTTP(w, r)
			s.logger.Debug("Request completed: %s %s (%v)", r.Method, r.URL.Path, time.Since(start))
		})
	})

	// Add new routes for FIFO tracking
	s.router.HandleFunc("/api/portfolios/{id}/lots", s.GetLots).Methods("GET")
	s.router.HandleFunc("/api/portfolios/{id}/summary", s.GetPortfolioSummary).Methods("GET")

	// Add reporting routes
	s.router.HandleFunc("/api/portfolios/{id}/performance", reportingHandler.GetPortfolioPerformance).Methods("GET")

	// Stock routes
	s.router.HandleFunc("/api/stocks", s.GetStocks).Methods("GET")
	s.router.HandleFunc("/api/stocks/{ticker}/details", s.GetStockDetails).Methods("GET")
	s.router.HandleFunc("/api/stocks/{ticker}/sparkline", s.GetStockSparkline).Methods("GET")
	s.router.HandleFunc("/api/stocks/{ticker}/chart", s.GetStockChartData).Methods("GET")
}

// setupRouter configures middleware for the server.
func (s *Server) setupRouter() {
	s.router.HandleFunc("/health", s.healthCheck).Methods("GET")
	s.router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	})
}

// Start begins listening for HTTP requests.
func (s *Server) Start() error {
	// Initial startup message
	s.logger.Info("Starting API server on port %s", s.config.Server.Port)

	// Create HTTP server with proper configuration
	srv := &http.Server{
		Addr:         ":" + s.config.Server.Port,
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Channel for server errors
	errChan := make(chan error, 1)

	// Start server in a goroutine
	go func() {
		s.logger.Info("HTTP server starting on http://localhost:%s", s.config.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("HTTP server error: %v", err)
			errChan <- err
		}
	}()

	// Wait a moment for the server to start
	time.Sleep(100 * time.Millisecond)

	// Clear startup message
	s.logger.Info("===========================================")
	s.logger.Info("🚀 Server is ready at http://localhost:%s", s.config.Server.Port)
	s.logger.Info("Available endpoints:")
	s.logger.Info("  GET /api/test")
	s.logger.Info("  GET /api/stocks")
	s.logger.Info("  GET /api/stocks/latest")
	s.logger.Info("  GET /api/stocks/{ticker}")
	s.logger.Info("  GET /api/stocks/{ticker}/sparkline")
	s.logger.Info("===========================================")

	// Wait for interrupt signal
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Wait for either error or interrupt
	select {
	case err := <-errChan:
		return fmt.Errorf("server error: %w", err)
	case <-stop:
		s.logger.Info("Shutdown signal received")
	}

	// Graceful shutdown
	s.logger.Info("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		s.logger.Error("Server shutdown failed: %v", err)
		return err
	}

	s.logger.Info("Server stopped gracefully")
	return nil
}

// ResetPortfolio resets a portfolio by deleting all transactions associated with it
func (s *Server) ResetPortfolio(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	portfolioID, err := strconv.Atoi(vars["id"])
	if err != nil {
		s.respondWithError(w, http.StatusBadRequest, "Invalid portfolio ID")
		return
	}

	// Start transaction
	tx, err := s.db.Begin()
	if err != nil {
		s.respondWithError(w, http.StatusInternalServerError, "Failed to start transaction")
		return
	}
	defer tx.Rollback()

	// Delete all transactions
	query := `DELETE FROM portfolio_transactions WHERE portfolio_id = $1`
	_, err = tx.Exec(query, portfolioID)
	if err != nil {
		s.respondWithError(w, http.StatusInternalServerError, "Failed to reset portfolio")
		return
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		s.respondWithError(w, http.StatusInternalServerError, "Failed to commit transaction")
		return
	}

	s.respondWithJSON(w, http.StatusOK, map[string]string{"message": "Portfolio reset successful"})
}

// respondWithError sends an error response with the specified status code and message
func (s *Server) respondWithError(w http.ResponseWriter, code int, message string) {
	s.respondWithJSON(w, code, map[string]string{"error": message})
}

// respondWithJSON sends a JSON response with the specified status code and payload
func (s *Server) respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(payload)
}

// TestConnection handler
func (s *Server) TestConnection(w http.ResponseWriter, r *http.Request) {
	s.logger.Info("Test connection endpoint hit")
	s.respondWithJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"message": "Server is running",
	})
}

func (s *Server) startStockUpdater() {
	// Run initial update in background
	go func() {
		s.logger.Info("Initial stock update running...")
		if err := s.scraper.ScrapeStockPrices(); err != nil {
			s.logger.Error("Initial stock update failed: %v", err)
		} else {
			s.logger.Info("Initial stock update completed successfully")
		}
	}()

	// Set up hourly updates
	ticker := time.NewTicker(1 * time.Hour)
	go func() {
		for {
			select {
			case <-ticker.C:
				s.logger.Info("Running hourly stock update")
				if err := s.scraper.ScrapeStockPrices(); err != nil {
					s.logger.Error("Failed to update stocks: %v", err)
				} else {
					s.logger.Info("Hourly stock update completed successfully")
				}
			case <-s.ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()
}

func (s *Server) verifyRoutes() {
	s.logger.Debug("Verifying registered routes:")
	s.router.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		pathTemplate, _ := route.GetPathTemplate()
		methods, _ := route.GetMethods()
		s.logger.Debug("Route: %s [%v]", pathTemplate, methods)
		return nil
	})
}

func (s *Server) Routes() {
	// Portfolio routes
	s.router.HandleFunc("/api/portfolios/{id}/transactions", s.GetTransactions).Methods("GET")
	s.router.HandleFunc("/api/portfolios/{id}/transactions", s.CreateTransaction).Methods("POST")
	s.router.HandleFunc("/api/portfolios/{id}/holdings", s.GetPortfolioHoldings).Methods("GET")
	s.router.HandleFunc("/api/portfolios/{id}/lots", s.GetLots).Methods("GET")
	s.router.HandleFunc("/api/portfolios/{id}/summary", s.GetPortfolioSummary).Methods("GET")
}

// Add if not present
func (s *Server) Router() http.Handler {
	return s.router
}

// validateTicker checks if a ticker is valid (either 'CASH' or exists in tickers table)
func (s *Server) validateTicker(ticker string, tx *sql.Tx) error {
	if ticker == "CASH" {
		return nil
	}

	var exists bool
	err := tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM tickers WHERE ticker = $1)`, ticker).Scan(&exists)
	if err != nil {
		return fmt.Errorf("error checking ticker existence: %v", err)
	}
	if !exists {
		return fmt.Errorf("invalid ticker: %s", ticker)
	}
	return nil
}

func (s *Server) healthCheck(w http.ResponseWriter, r *http.Request) {
	// Check database connection
	err := s.db.Ping()
	if err != nil {
		s.respondWithError(w, http.StatusServiceUnavailable, "Database unavailable")
		return
	}

	s.respondWithJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"version": "1.0.0",
	})
}
