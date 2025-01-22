// Package api provides the HTTP server and API endpoints for the Local Portfolio Manager
package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"localportfoliomanager/internal/utils"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

// Server represents the API server instance
// It handles HTTP requests and manages connections to the database
type Server struct {
	router *mux.Router   // HTTP request router
	logger *utils.Logger // Application logger
	config *utils.Config // Application configuration
	db     *sql.DB       // Database connection
}

// NewServer creates and initializes a new API server instance
// It sets up the database connection and configures the HTTP router
//
// Parameters:
//   - logger: Application logger for recording server activities
//   - config: Application configuration including database and server settings
//
// Returns:
//   - *Server: Initialized server instance
//   - The function will call logger.Fatal if database connection fails
func NewServer(logger *utils.Logger, config *utils.Config) *Server {
	// Initialize database connection
	db, err := sql.Open("postgres", config.Database.DSN)
	if err != nil {
		logger.Fatal("Failed to connect to database: %v", err)
	}

	server := &Server{
		router: mux.NewRouter(),
		logger: logger,
		config: config,
		db:     db,
	}

	server.setupRoutes()
	return server
}

// setupRoutes configures all API endpoints for the server
// This includes routes for stocks, portfolios, and transactions
//
// API Endpoints:
// Stock Data:
//   - GET /api/stocks                     - List all stocks with latest prices
//   - GET /api/stocks/{ticker}            - Get detailed information for a specific stock
//   - GET /api/stocks/{ticker}/prices     - Get historical prices for a stock
//
// Portfolio Management:
//   - POST   /api/portfolios             - Create a new portfolio
//   - GET    /api/portfolios             - List all portfolios
//   - GET    /api/portfolios/{id}        - Get portfolio details
//   - PUT    /api/portfolios/{id}        - Update portfolio information
//   - DELETE /api/portfolios/{id}        - Delete a portfolio
//   - GET    /api/portfolios/{id}/performance - Get portfolio performance metrics
//
// Transaction Management:
//   - POST   /api/portfolios/{id}/transactions    - Record a new transaction
//   - GET    /api/portfolios/{id}/transactions    - List portfolio transactions
//   - GET    /api/portfolios/{id}/transactions/{txId} - Get transaction details
func (s *Server) setupRoutes() {
	// Add debug logging
	s.logger.Debug("Setting up routes...")

	// Add a test/debug endpoint
	s.router.HandleFunc("/api/test", func(w http.ResponseWriter, r *http.Request) {
		routes := []string{}
		s.router.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
			path, _ := route.GetPathTemplate()
			methods, _ := route.GetMethods()
			routes = append(routes, fmt.Sprintf("%v %v", methods, path))
			return nil
		})
		s.respondWithJSON(w, http.StatusOK, map[string]interface{}{
			"status": "Server is running",
			"routes": routes,
		})
	}).Methods("GET")

	// Stock data endpoints
	s.router.HandleFunc("/api/stocks", s.GetStocks).Methods("GET")
	s.router.HandleFunc("/api/stocks/{ticker}", s.GetStockByTicker).Methods("GET")
	s.router.HandleFunc("/api/stocks/{ticker}/prices", s.GetStockPrices).Methods("GET")

	// Portfolio endpoints
	s.router.HandleFunc("/api/portfolios", s.CreatePortfolio).Methods("POST")
	s.router.HandleFunc("/api/portfolios", s.ListPortfolios).Methods("GET")
	s.router.HandleFunc("/api/portfolios/{id}", s.GetPortfolio).Methods("GET")
	s.router.HandleFunc("/api/portfolios/{id}", s.UpdatePortfolio).Methods("PUT")
	s.router.HandleFunc("/api/portfolios/{id}", s.DeletePortfolio).Methods("DELETE")
	s.router.HandleFunc("/api/portfolios/{id}/performance", s.GetPortfolioPerformance).Methods("GET")

	// Transaction endpoints
	s.router.HandleFunc("/api/portfolios/{id}/transactions/history", s.GetTransactionHistory).Methods("GET")
	s.router.HandleFunc("/api/portfolios/{id}/transactions/{txId}", s.GetTransaction).Methods("GET")
	s.router.HandleFunc("/api/portfolios/{id}/transactions", s.ListTransactions).Methods("GET")

	// New route for getting portfolio balance
	s.router.HandleFunc("/api/portfolios/{id}/balance", s.GetPortfolioBalance).Methods("GET")

	// Add this to setupRoutes
	s.router.HandleFunc("/api/test/parse-date", func(w http.ResponseWriter, r *http.Request) {
		dateStr := r.URL.Query().Get("date")
		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			s.respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Failed to parse date: %v", err))
			return
		}

		s.respondWithJSON(w, http.StatusOK, map[string]interface{}{
			"input":     dateStr,
			"parsed":    date,
			"formatted": date.Format(time.RFC3339),
			"type":      fmt.Sprintf("%T", date),
		})
	}).Methods("GET")

	// Add reset endpoint
	s.router.HandleFunc("/api/portfolios/{id}/reset", s.ResetPortfolio).Methods("POST")

	// Add portfolio holdings endpoint
	s.router.HandleFunc("/api/portfolios/{id}/holdings", s.GetPortfolioHoldings).Methods("GET")

	// Add new routes
	s.router.HandleFunc("/api/portfolios/{id}/performance", s.GetPortfolioPerformance).Methods("GET")

	// Add daily portfolio values endpoint
	s.router.HandleFunc("/api/portfolios/{id}/daily-values", s.GetPortfolioDailyValues).Methods("GET")
}

// Start begins listening for HTTP requests on the configured port
// This is a blocking call that will run until the server is shut down
//
// Returns:
//   - error: Any error that occurs while running the server
//
// Example Usage:
//
//	server := NewServer(logger, config)
//	if err := server.Start(); err != nil {
//	    log.Fatal(err)
//	}
func (s *Server) Start() error {
	s.logger.Info("Starting API server on port %s", s.config.Server.Port)
	return http.ListenAndServe(":"+s.config.Server.Port, s.router)
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
	response, err := json.Marshal(payload)
	if err != nil {
		s.logger.Error("Failed to marshal JSON response: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}
