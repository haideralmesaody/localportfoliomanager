package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"localportfoliomanager/internal/reporting"
	"net/http"
	"strconv"
	"time"

	"localportfoliomanager/internal/utils"

	"github.com/gorilla/mux"
)

// Portfolio types for request/response
type Portfolio struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CreatePortfolioRequest struct {
	Name        string `json:"name" validate:"required"`
	Description string `json:"description"`
}

type PortfolioHandler struct {
	reportingService *reporting.ReportingService
	server           *Server
	logger           *utils.AppLogger
}

func NewPortfolioHandler(reportingService *reporting.ReportingService, server *Server, logger *utils.AppLogger) *PortfolioHandler {
	return &PortfolioHandler{
		reportingService: reportingService,
		server:           server,
		logger:           logger,
	}
}

// CreatePortfolio handles the creation of a new portfolio
func (s *Server) CreatePortfolio(w http.ResponseWriter, r *http.Request) {
	var req CreatePortfolioRequest
	var err error
	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		s.respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Validate required fields
	if req.Name == "" {
		s.respondWithError(w, http.StatusBadRequest, "Portfolio name is required")
		return
	}

	// Insert into database
	query := `
		INSERT INTO portfolios (name, description)
		VALUES ($1, $2)
		RETURNING id, name, description, created_at, updated_at
	`

	var portfolio Portfolio
	err = s.db.QueryRow(
		query,
		req.Name,
		req.Description,
	).Scan(
		&portfolio.ID,
		&portfolio.Name,
		&portfolio.Description,
		&portfolio.CreatedAt,
		&portfolio.UpdatedAt,
	)

	if err != nil {
		s.logger.Error("Failed to create portfolio: %v", err)
		s.respondWithError(w, http.StatusInternalServerError, "Failed to create portfolio")
		return
	}

	s.respondWithJSON(w, http.StatusCreated, portfolio)
}

// ListPortfolios returns all portfolios
func (s *Server) ListPortfolios(w http.ResponseWriter, r *http.Request) {
	query := `
		SELECT id, name, description, created_at, updated_at
		FROM portfolios
		ORDER BY created_at DESC
	`

	rows, err := s.db.Query(query)
	if err != nil {
		s.logger.Error("Failed to query portfolios: %v", err)
		s.respondWithError(w, http.StatusInternalServerError, "Failed to fetch portfolios")
		return
	}
	defer rows.Close()

	var portfolios []Portfolio
	for rows.Next() {
		var p Portfolio
		err = rows.Scan(
			&p.ID,
			&p.Name,
			&p.Description,
			&p.CreatedAt,
			&p.UpdatedAt,
		)
		if err != nil {
			s.logger.Error("Failed to scan portfolio row: %v", err)
			s.respondWithError(w, http.StatusInternalServerError, "Failed to process portfolio data")
			return
		}
		portfolios = append(portfolios, p)
	}

	s.respondWithJSON(w, http.StatusOK, portfolios)
}

// GetPortfolio returns a specific portfolio by ID
func (s *Server) GetPortfolio(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		s.respondWithError(w, http.StatusBadRequest, "Invalid portfolio ID")
		return
	}

	query := `
		SELECT id, name, description, created_at, updated_at
		FROM portfolios
		WHERE id = $1
	`

	var portfolio Portfolio
	err = s.db.QueryRow(query, id).Scan(
		&portfolio.ID,
		&portfolio.Name,
		&portfolio.Description,
		&portfolio.CreatedAt,
		&portfolio.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		s.respondWithError(w, http.StatusNotFound, "Portfolio not found")
		return
	}
	if err != nil {
		s.logger.Error("Failed to fetch portfolio: %v", err)
		s.respondWithError(w, http.StatusInternalServerError, "Failed to fetch portfolio")
		return
	}

	s.respondWithJSON(w, http.StatusOK, portfolio)
}

// DeletePortfolio deletes a portfolio
func (s *Server) DeletePortfolio(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		s.logger.Error("Invalid portfolio ID: %v", err)
		s.respondWithError(w, http.StatusBadRequest, "Invalid portfolio ID")
		return
	}

	s.logger.Info("Attempting to delete portfolio with ID: %d", id)

	// Start transaction
	tx, err := s.db.Begin()
	if err != nil {
		s.logger.Error("Failed to start transaction: %v", err)
		s.respondWithError(w, http.StatusInternalServerError, "Failed to start delete operation")
		return
	}
	defer tx.Rollback()

	// First, delete any related records (if they exist)
	deleteRelatedQuery := `DELETE FROM portfolio_transactions WHERE portfolio_id = $1`
	_, err = tx.Exec(deleteRelatedQuery, id)
	if err != nil {
		s.logger.Error("Failed to delete related transactions: %v", err)
		s.respondWithError(w, http.StatusInternalServerError, "Failed to delete related records")
		return
	}

	// Then delete the portfolio
	query := `DELETE FROM portfolios WHERE id = $1`
	result, err := tx.Exec(query, id)
	if err != nil {
		s.logger.Error("Failed to delete portfolio: %v", err)
		s.respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to delete portfolio: %v", err))
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		s.logger.Error("Failed to get rows affected: %v", err)
		s.respondWithError(w, http.StatusInternalServerError, "Failed to confirm deletion")
		return
	}

	if rowsAffected == 0 {
		s.logger.Info("No portfolio found with ID: %d", id)
		s.respondWithError(w, http.StatusNotFound, "Portfolio not found")
		return
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		s.logger.Error("Failed to commit transaction: %v", err)
		s.respondWithError(w, http.StatusInternalServerError, "Failed to complete deletion")
		return
	}

	s.logger.Info("Successfully deleted portfolio with ID: %d", id)
	s.respondWithJSON(w, http.StatusOK, map[string]string{
		"message": fmt.Sprintf("Portfolio %d deleted successfully", id),
	})
}

// RenamePortfolio updates the name and description of an existing portfolio
func (s *Server) RenamePortfolio(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		s.respondWithError(w, http.StatusBadRequest, "Invalid portfolio ID")
		return
	}

	// Decode the request payload
	var req struct {
		NewName     string `json:"new_name"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if req.NewName == "" {
		s.respondWithError(w, http.StatusBadRequest, "New portfolio name is required")
		return
	}

	query := `
		UPDATE portfolios
		SET name = $1, description = $2, updated_at = NOW()
		WHERE id = $3
		RETURNING id, name, description, created_at, updated_at
	`

	var portfolio Portfolio
	err = s.db.QueryRow(query, req.NewName, req.Description, id).Scan(
		&portfolio.ID,
		&portfolio.Name,
		&portfolio.Description,
		&portfolio.CreatedAt,
		&portfolio.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		s.respondWithError(w, http.StatusNotFound, "Portfolio not found")
		return
	}
	if err != nil {
		s.logger.Error("Failed to rename portfolio: %v", err)
		s.respondWithError(w, http.StatusInternalServerError, "Failed to rename portfolio")
		return
	}

	s.respondWithJSON(w, http.StatusOK, portfolio)
}

// GetPortfolioHoldings returns all holdings for a specific portfolio
func (s *Server) GetPortfolioHoldings(w http.ResponseWriter, r *http.Request) {
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

	// Initialize holdings if needed
	err = s.initializePortfolioHoldings(portfolioID, tx)
	if err != nil {
		s.respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to initialize holdings: %v", err))
		return
	}

	// Rest of the existing query...
	query := `
		SELECT 
			h.id,
			h.portfolio_id,
			h.ticker,
			COALESCE(h.shares, 0) as shares,
			COALESCE(h.purchase_cost_average, 0) as purchase_cost_average,
			COALESCE(h.purchase_cost_fifo, 0) as purchase_cost_fifo,
			COALESCE(h.current_price, 0) as current_price,
			COALESCE(h.price_last_date, CURRENT_TIMESTAMP) as price_last_date,
			COALESCE(h.position_cost_average, 0) as position_cost_average,
			COALESCE(h.position_cost_fifo, 0) as position_cost_fifo,
			COALESCE(h.unrealized_gain_average, 0) as unrealized_gain_average,
			COALESCE(h.unrealized_gain_fifo, 0) as unrealized_gain_fifo,
			COALESCE(h.target_percentage, 0) as target_percentage,
			COALESCE(h.current_percentage, 0) as current_percentage,
			COALESCE(h.adjustment_percentage, 0) as adjustment_percentage,
			COALESCE(h.adjustment_value, 0) as adjustment_value,
			COALESCE(h.adjustment_quantity, 0) as adjustment_quantity,
			h.created_at,
			h.updated_at
		FROM portfolio_holdings h
		WHERE h.portfolio_id = $1
		ORDER BY h.ticker ASC`

	rows, err := tx.Query(query, portfolioID)
	if err != nil {
		s.respondWithError(w, http.StatusInternalServerError, "Failed to fetch holdings")
		return
	}
	defer rows.Close()

	var holdings []Holding
	for rows.Next() {
		var h Holding
		err := rows.Scan(
			&h.ID, &h.PortfolioID, &h.Ticker, &h.Shares,
			&h.PurchaseCostAverage, &h.PurchaseCostFIFO,
			&h.CurrentPrice, &h.PriceLastDate,
			&h.PositionCostAverage, &h.PositionCostFIFO,
			&h.UnrealizedGainAverage, &h.UnrealizedGainFIFO,
			&h.TargetPercentage, &h.CurrentPercentage,
			&h.AdjustmentPercentage, &h.AdjustmentValue,
			&h.AdjustmentQuantity, &h.CreatedAt, &h.UpdatedAt,
		)
		if err != nil {
			s.logger.Error("Error scanning holding: %v", err)
			s.respondWithError(w, http.StatusInternalServerError, "Error scanning holding")
			return
		}
		holdings = append(holdings, h)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		s.respondWithError(w, http.StatusInternalServerError, "Failed to commit transaction")
		return
	}

	s.respondWithJSON(w, http.StatusOK, holdings)
}

// GetHoldings returns all holdings for a portfolio with calculated values
func (s *Server) GetHoldings(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	portfolioID, err := strconv.Atoi(vars["id"])
	if err != nil {
		s.respondWithError(w, http.StatusBadRequest, "Invalid portfolio ID")
		return
	}

	query := `
		SELECT 
			h.id,
			h.portfolio_id,
			h.ticker,
			COALESCE(h.shares, 0) as shares,
			COALESCE(h.purchase_cost_average, 0) as purchase_cost_average,
			COALESCE(h.purchase_cost_fifo, 0) as purchase_cost_fifo,
			COALESCE(h.current_price, 0) as current_price,
			COALESCE(h.price_last_date, CURRENT_TIMESTAMP) as price_last_date,
			COALESCE(h.position_cost_average, 0) as position_cost_average,
			COALESCE(h.position_cost_fifo, 0) as position_cost_fifo,
			COALESCE(h.unrealized_gain_average, 0) as unrealized_gain_average,
			COALESCE(h.unrealized_gain_fifo, 0) as unrealized_gain_fifo,
			COALESCE(h.target_percentage, 0) as target_percentage,
			COALESCE(h.current_percentage, 0) as current_percentage,
			COALESCE(h.adjustment_percentage, 0) as adjustment_percentage,
			COALESCE(h.adjustment_value, 0) as adjustment_value,
			COALESCE(h.adjustment_quantity, 0) as adjustment_quantity,
			h.created_at,
			h.updated_at
		FROM portfolio_holdings h
		WHERE h.portfolio_id = $1
		ORDER BY h.ticker ASC`

	rows, err := s.db.Query(query, portfolioID)
	if err != nil {
		s.logger.Error("Failed to fetch holdings: %v", err)
		s.respondWithError(w, http.StatusInternalServerError, "Failed to fetch holdings")
		return
	}
	defer rows.Close()

	var holdings []Holding
	for rows.Next() {
		var h Holding
		err := rows.Scan(
			&h.ID, &h.PortfolioID, &h.Ticker, &h.Shares,
			&h.PurchaseCostAverage, &h.PurchaseCostFIFO,
			&h.CurrentPrice, &h.PriceLastDate,
			&h.PositionCostAverage, &h.PositionCostFIFO,
			&h.UnrealizedGainAverage, &h.UnrealizedGainFIFO,
			&h.TargetPercentage, &h.CurrentPercentage,
			&h.AdjustmentPercentage, &h.AdjustmentValue,
			&h.AdjustmentQuantity, &h.CreatedAt, &h.UpdatedAt,
		)
		if err != nil {
			s.logger.Error("Error scanning holding: %v", err)
			s.respondWithError(w, http.StatusInternalServerError, "Error scanning holding")
			return
		}
		holdings = append(holdings, h)
	}

	s.respondWithJSON(w, http.StatusOK, holdings)
}

// GetPortfolioSummary returns summary information for a portfolio
func (s *Server) GetPortfolioSummary(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	portfolioID, err := strconv.Atoi(vars["id"])
	if err != nil {
		s.respondWithError(w, http.StatusBadRequest, "Invalid portfolio ID")
		return
	}

	// Updated query to include realized gains from transactions
	query := `
		WITH portfolio_totals AS (
			SELECT 
				SUM(CASE 
					WHEN ticker = 'CASH' THEN shares 
					ELSE shares * COALESCE(current_price, purchase_cost_average)
				END) as total_value,
				SUM(CASE 
					WHEN ticker = 'CASH' THEN 0 
					ELSE shares * purchase_cost_average
				END) as total_cost_average,
				SUM(CASE 
					WHEN ticker = 'CASH' THEN 0 
					ELSE (
						SELECT COALESCE(SUM(l.remaining_shares * l.purchase_price) / NULLIF(SUM(l.remaining_shares), 0), 0)
						FROM portfolio_stock_lots l
						WHERE l.portfolio_id = h.portfolio_id AND l.ticker = h.ticker
					) * shares
				END) as total_cost_fifo
			FROM portfolio_holdings h
			WHERE h.portfolio_id = $1
		),
		realized_gains AS (
			SELECT 
				COALESCE(SUM(realized_gain_avg), 0) as total_realized_gain_avg,
				COALESCE(SUM(realized_gain_fifo), 0) as total_realized_gain_fifo
			FROM portfolio_transactions
			WHERE portfolio_id = $1 AND type = 'SELL'
		)
		SELECT 
			p.name,
			p.description,
			COALESCE(pt.total_value, 0) as total_value,
			COALESCE(pt.total_cost_average, 0) as total_cost_average,
			COALESCE(pt.total_cost_fifo, 0) as total_cost_fifo,
			COALESCE(pt.total_value - pt.total_cost_average + rg.total_realized_gain_avg, 0) as total_gain_average,
			COALESCE(pt.total_value - pt.total_cost_fifo + rg.total_realized_gain_fifo, 0) as total_gain_fifo,
			COALESCE(rg.total_realized_gain_avg, 0) as realized_gain_average,
			COALESCE(rg.total_realized_gain_fifo, 0) as realized_gain_fifo,
			p.created_at,
			p.updated_at
		FROM portfolios p
		LEFT JOIN portfolio_totals pt ON true
		LEFT JOIN realized_gains rg ON true
		WHERE p.id = $1`

	var summary PortfolioSummary
	err = s.db.QueryRow(query, portfolioID).Scan(
		&summary.Name,
		&summary.Description,
		&summary.TotalValue,
		&summary.TotalCostAverage,
		&summary.TotalCostFIFO,
		&summary.TotalGainAverage,
		&summary.TotalGainFIFO,
		&summary.RealizedGainAverage,
		&summary.RealizedGainFIFO,
		&summary.CreatedAt,
		&summary.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		s.respondWithError(w, http.StatusNotFound, "Portfolio not found")
		return
	}
	if err != nil {
		s.respondWithError(w, http.StatusInternalServerError, "Error fetching portfolio summary")
		return
	}

	s.respondWithJSON(w, http.StatusOK, summary)
}

// GetLots returns all FIFO lots for a portfolio
func (s *Server) GetLots(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	portfolioID, err := strconv.Atoi(vars["id"])
	if err != nil {
		s.respondWithError(w, http.StatusBadRequest, "Invalid portfolio ID")
		return
	}

	query := `
		SELECT id, portfolio_id, ticker, shares, remaining_shares,
			   purchase_price, purchase_date, created_at
		FROM portfolio_stock_lots
		WHERE portfolio_id = $1
		ORDER BY purchase_date ASC`

	rows, err := s.db.Query(query, portfolioID)
	if err != nil {
		s.respondWithError(w, http.StatusInternalServerError, "Failed to fetch lots")
		return
	}
	defer rows.Close()

	var lots []StockLot
	for rows.Next() {
		var lot StockLot
		err := rows.Scan(
			&lot.ID, &lot.PortfolioID, &lot.Ticker,
			&lot.Shares, &lot.RemainingShares,
			&lot.PurchasePrice, &lot.PurchaseDate, &lot.CreatedAt,
		)
		if err != nil {
			s.respondWithError(w, http.StatusInternalServerError, "Error scanning lot")
			return
		}
		lots = append(lots, lot)
	}

	s.respondWithJSON(w, http.StatusOK, lots)
}

// initializePortfolioHoldings creates initial CASH holding for a portfolio
func (s *Server) initializePortfolioHoldings(portfolioID int, tx *sql.Tx) error {
	// Check if portfolio exists
	var exists bool
	err := tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM portfolios WHERE id = $1)`, portfolioID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check portfolio existence: %v", err)
	}
	if !exists {
		return fmt.Errorf("portfolio %d does not exist", portfolioID)
	}

	// Initialize CASH holding only
	_, err = tx.Exec(`
		INSERT INTO portfolio_holdings (portfolio_id, ticker, shares)
		VALUES ($1, 'CASH', 0)
		ON CONFLICT (portfolio_id, ticker) DO NOTHING
	`, portfolioID)
	if err != nil {
		return fmt.Errorf("failed to initialize cash holding: %v", err)
	}

	return nil
}

// initializeTickerHolding creates a holding for a specific ticker if it doesn't exist
func (s *Server) initializeTickerHolding(portfolioID int, ticker string, tx *sql.Tx) error {
	_, err := tx.Exec(`
		INSERT INTO portfolio_holdings (portfolio_id, ticker, shares)
		VALUES ($1, $2, 0)
		ON CONFLICT (portfolio_id, ticker) DO NOTHING
	`, portfolioID, ticker)

	if err != nil {
		return fmt.Errorf("failed to initialize holding for %s: %v", ticker, err)
	}
	return nil
}

func (h *PortfolioHandler) GetPerformanceReport(w http.ResponseWriter, r *http.Request) {
	fmt.Println("GetPerformanceReport called")

	vars := mux.Vars(r)
	portfolioID, err := strconv.Atoi(vars["id"])
	if err != nil {
		fmt.Printf("Invalid portfolio ID: %v\n", err)
		http.Error(w, "Invalid portfolio ID", http.StatusBadRequest)
		return
	}

	period := r.URL.Query().Get("period")
	if period == "" {
		period = "ALL"
	}

	fmt.Printf("Generating report for portfolio %d, period: %s\n", portfolioID, period)

	// Check if reportingService is nil
	if h.reportingService == nil {
		fmt.Println("Error: reportingService is nil")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	report, err := h.reportingService.GeneratePerformanceReport(portfolioID, period)
	if err != nil {
		fmt.Printf("Error generating report: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Printf("Report generated successfully: %+v\n", report)

	// Create response object
	response := struct {
		Status string `json:"status"`
		Data   struct {
			PortfolioID     int                            `json:"portfolio_id"`
			Name            string                         `json:"name"`
			CurrentValue    float64                        `json:"current_value"`
			CashBalance     float64                        `json:"cash_balance"`
			StocksValue     float64                        `json:"stocks_value"`
			RealizedGains   float64                        `json:"realized_gains"`
			UnrealizedGains float64                        `json:"unrealized_gains"`
			DividendIncome  float64                        `json:"dividend_income"`
			TotalReturn     float64                        `json:"total_return"`
			ReturnPercent   float64                        `json:"return_percent"`
			Holdings        []reporting.HoldingPerformance `json:"holdings"`
		} `json:"data"`
	}{
		Status: "success",
		Data: struct {
			PortfolioID     int                            `json:"portfolio_id"`
			Name            string                         `json:"name"`
			CurrentValue    float64                        `json:"current_value"`
			CashBalance     float64                        `json:"cash_balance"`
			StocksValue     float64                        `json:"stocks_value"`
			RealizedGains   float64                        `json:"realized_gains"`
			UnrealizedGains float64                        `json:"unrealized_gains"`
			DividendIncome  float64                        `json:"dividend_income"`
			TotalReturn     float64                        `json:"total_return"`
			ReturnPercent   float64                        `json:"return_percent"`
			Holdings        []reporting.HoldingPerformance `json:"holdings"`
		}{
			PortfolioID:     report.PortfolioID,
			Name:            report.Name,
			CurrentValue:    report.CurrentValue,
			CashBalance:     report.CashBalance,
			StocksValue:     report.StocksValue,
			RealizedGains:   report.RealizedGains,
			UnrealizedGains: report.UnrealizedGains,
			DividendIncome:  report.DividendIncome,
			TotalReturn:     report.TotalReturn,
			ReturnPercent:   report.ReturnPercent,
			Holdings:        report.Holdings,
		},
	}

	// Add debug logging before response
	responseBytes, err := json.MarshalIndent(response, "", "    ")
	if err != nil {
		fmt.Printf("Error marshaling response: %v\n", err)
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
		return
	}
	fmt.Printf("Response to be sent: %s\n", string(responseBytes))

	// Use direct response if server is nil
	if h.server == nil {
		fmt.Println("Warning: server is nil, using direct response")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(responseBytes)
		return
	}

	h.server.respondWithJSON(w, http.StatusOK, response)
}
