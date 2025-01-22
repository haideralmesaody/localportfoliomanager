package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

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

// UpdatePortfolio updates an existing portfolio
func (s *Server) UpdatePortfolio(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		s.respondWithError(w, http.StatusBadRequest, "Invalid portfolio ID")
		return
	}

	var req CreatePortfolioRequest
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

	query := `
		UPDATE portfolios 
		SET name = $1, description = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $3
		RETURNING id, name, description, created_at, updated_at
	`

	var portfolio Portfolio
	err = s.db.QueryRow(
		query,
		req.Name,
		req.Description,
		id,
	).Scan(
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
		s.logger.Error("Failed to update portfolio: %v", err)
		s.respondWithError(w, http.StatusInternalServerError, "Failed to update portfolio")
		return
	}

	s.respondWithJSON(w, http.StatusOK, portfolio)
}

// DeletePortfolio deletes a portfolio
func (s *Server) DeletePortfolio(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		s.respondWithError(w, http.StatusBadRequest, "Invalid portfolio ID")
		return
	}

	query := `DELETE FROM portfolios WHERE id = $1`
	result, err := s.db.Exec(query, id)
	if err != nil {
		s.logger.Error("Failed to delete portfolio: %v", err)
		s.respondWithError(w, http.StatusInternalServerError, "Failed to delete portfolio")
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		s.logger.Error("Failed to get rows affected: %v", err)
		s.respondWithError(w, http.StatusInternalServerError, "Failed to confirm portfolio deletion")
		return
	}

	if rowsAffected == 0 {
		s.respondWithError(w, http.StatusNotFound, "Portfolio not found")
		return
	}

	s.respondWithJSON(w, http.StatusOK, map[string]string{"message": "Portfolio deleted successfully"})
}

// GetPortfolioPerformance calculates portfolio performance metrics
func (s *Server) GetPortfolioPerformance(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	portfolioID, err := strconv.Atoi(vars["id"])
	if err != nil {
		s.respondWithError(w, http.StatusBadRequest, "Invalid portfolio ID")
		return
	}

	// Start a transaction for consistent calculations
	tx, err := s.db.Begin()
	if err != nil {
		s.respondWithError(w, http.StatusInternalServerError, "Failed to start transaction")
		return
	}
	defer tx.Rollback()

	// Get portfolio date range
	query := `
		SELECT 
			MIN(transaction_at) as start_date,
			MAX(transaction_at) as end_date
		FROM portfolio_transactions
		WHERE portfolio_id = $1
	`
	var perf PortfolioPerformance
	err = tx.QueryRow(query, portfolioID).Scan(&perf.StartDate, &perf.EndDate)
	if err != nil {
		s.respondWithError(w, http.StatusInternalServerError, "Failed to get date range")
		return
	}

	// Calculate start and end values
	perf.StartValue, err = s.getPortfolioValueAtDate(portfolioID, perf.StartDate, tx)
	if err != nil {
		s.respondWithError(w, http.StatusInternalServerError, "Failed to get start value")
		return
	}

	perf.EndValue, err = s.getPortfolioValueAtDate(portfolioID, perf.EndDate, tx)
	if err != nil {
		s.respondWithError(w, http.StatusInternalServerError, "Failed to get end value")
		return
	}

	// Calculate net contributions (deposits - withdrawals)
	contributionsQuery := `
		SELECT 
			COALESCE(SUM(CASE WHEN type = 'DEPOSIT' THEN amount ELSE -amount END), 0)
		FROM portfolio_transactions
		WHERE portfolio_id = $1
		AND type IN ('DEPOSIT', 'WITHDRAW')
	`
	err = tx.QueryRow(contributionsQuery, portfolioID).Scan(&perf.NetContributions)
	if err != nil {
		s.respondWithError(w, http.StatusInternalServerError, "Failed to calculate contributions")
		return
	}

	// Calculate realized gains
	realizedGainsQuery := `
		SELECT COALESCE(
			SUM(
				CASE 
					WHEN type = 'SELL' THEN (price - average_cost_before) * shares
					ELSE 0 
				END
			),
			0
		) as realized_gains
		FROM portfolio_transactions
		WHERE portfolio_id = $1
	`
	err = tx.QueryRow(realizedGainsQuery, portfolioID).Scan(&perf.RealizedGains)
	if err != nil {
		s.logger.Error("Failed to calculate realized gains: %v", err)
	}

	// Calculate dividend income
	dividendQuery := `
		SELECT COALESCE(
			SUM(amount),
			0
		) as dividend_income
		FROM portfolio_transactions
		WHERE portfolio_id = $1
		AND type = 'DIVIDEND'
	`
	err = tx.QueryRow(dividendQuery, portfolioID).Scan(&perf.DividendIncome)
	if err != nil {
		s.logger.Error("Failed to calculate dividend income: %v", err)
	}

	// Calculate unrealized gains
	perf.UnrealizedGains = perf.EndValue - perf.StartValue -
		perf.NetContributions - perf.RealizedGains - perf.DividendIncome

	// Calculate TWR and MWR
	perf.TWR = calculateTWR(perf.StartValue, perf.EndValue, perf.NetContributions)
	perf.MWR = calculateMWR(perf.StartValue, perf.EndValue, perf.NetContributions)

	tx.Commit()
	s.respondWithJSON(w, http.StatusOK, perf)
}

// Helper functions for performance calculations
func calculateTWR(startValue, endValue, netContributions float64) float64 {
	if startValue == 0 {
		return 0
	}
	// Adjust for contributions
	adjustedEndValue := endValue - netContributions
	return (adjustedEndValue / startValue) - 1
}

func calculateMWR(startValue, endValue, netContributions float64) float64 {
	if startValue == 0 {
		return 0
	}
	// Modified Dietz method
	return (endValue - startValue - netContributions) /
		(startValue + (netContributions / 2))
}

// Add this new type to types.go
type PortfolioBalance struct {
	Cash            float64 `json:"cash"`
	StockValue      float64 `json:"stock_value"`
	TotalValue      float64 `json:"total_value"`
	UnrealizedGains float64 `json:"unrealized_gains"`
}

// GetPortfolioBalance returns the current balance of a portfolio
func (s *Server) GetPortfolioBalance(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	portfolioID, err := strconv.Atoi(vars["id"])
	if err != nil {
		s.logger.Error("Invalid portfolio ID: %v", err)
		s.respondWithError(w, http.StatusBadRequest, "Invalid portfolio ID")
		return
	}

	// First validate if portfolio exists
	if err := s.validatePortfolio(portfolioID); err != nil {
		s.logger.Error("Portfolio not found: %v", err)
		s.respondWithError(w, http.StatusNotFound, "Portfolio not found")
		return
	}

	query := `
		WITH latest_cash AS (
			SELECT cash_balance_after as cash_balance
			FROM portfolio_transactions
			WHERE portfolio_id = $1
			ORDER BY transaction_at DESC, id DESC
			LIMIT 1
		),
		stock_positions AS (
			SELECT 
				ticker,
				shares_count_after as shares,
				price,
				average_cost_after as cost_basis
			FROM portfolio_transactions
			WHERE portfolio_id = $1
				AND type IN ('BUY', 'SELL')
				AND shares_count_after > 0
			ORDER BY transaction_at DESC, id DESC
		)
		SELECT 
			COALESCE((SELECT cash_balance FROM latest_cash), 0) as cash_balance,
			COALESCE(SUM(shares * price), 0) as stock_value,
			COALESCE(SUM(shares * (price - cost_basis)), 0) as unrealized_gains
		FROM stock_positions
	`

	s.logger.Debug("Executing balance query for portfolio %d", portfolioID)

	var balance PortfolioBalance
	err = s.db.QueryRow(query, portfolioID).Scan(
		&balance.Cash,
		&balance.StockValue,
		&balance.UnrealizedGains,
	)
	if err != nil {
		s.logger.Error("Failed to get portfolio balance: %v", err)
		s.respondWithError(w, http.StatusInternalServerError, "Failed to calculate portfolio balance")
		return
	}

	balance.TotalValue = balance.Cash + balance.StockValue
	s.respondWithJSON(w, http.StatusOK, balance)
}

// Helper function to get portfolio value at a specific date
func (s *Server) getPortfolioValueAtDate(portfolioID int, date time.Time, tx *sql.Tx) (float64, error) {
	query := `
		WITH latest_positions AS (
			SELECT DISTINCT ON (ticker)
				ticker,
				shares_count_after as shares,
				price,
				cash_balance_after as cash_balance
		FROM portfolio_transactions
		WHERE portfolio_id = $1
		AND transaction_at <= $2
			ORDER BY ticker, transaction_at DESC, id DESC
		)
		SELECT 
			COALESCE(MAX(cash_balance), 0) as cash_balance,
			COALESCE(SUM(shares * price), 0) as stock_value
		FROM latest_positions
	`

	var cashBalance, stockValue float64
	err := tx.QueryRow(query, portfolioID, date).Scan(&cashBalance, &stockValue)
	if err != nil {
		return 0, fmt.Errorf("failed to get portfolio value: %v", err)
	}

	return cashBalance + stockValue, nil
}

// GetPortfolioHoldings returns the current holdings for a portfolio
func (s *Server) GetPortfolioHoldings(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	portfolioID, err := strconv.Atoi(vars["id"])
	if err != nil {
		s.respondWithError(w, http.StatusBadRequest, "Invalid portfolio ID")
		return
	}

	s.logger.Debug("Getting holdings for portfolio %d", portfolioID)

	// Get latest positions with their average cost
	query := `
		WITH latest_transactions AS (
			SELECT 
				ticker,
				transaction_at,
				shares_count_after as shares,
				average_cost_after as average_cost,
				ROW_NUMBER() OVER (PARTITION BY ticker ORDER BY transaction_at DESC, id DESC) as rn
			FROM portfolio_transactions
			WHERE portfolio_id = $1
			AND type IN ('BUY', 'SELL')
			AND shares_count_after > 0
		)
		SELECT 
			ticker, 
			shares, 
			average_cost
		FROM latest_transactions
		WHERE rn = 1
	`

	rows, err := s.db.Query(query, portfolioID)
	if err != nil {
		s.logger.Error("Failed to get holdings: %v", err)
		s.respondWithError(w, http.StatusInternalServerError, "Failed to get holdings")
		return
	}
	defer rows.Close()

	var holdings []PortfolioHolding
	var totalCost, totalValue float64

	for rows.Next() {
		var h PortfolioHolding
		err := rows.Scan(&h.Ticker, &h.Shares, &h.AverageCost)
		if err != nil {
			s.logger.Error("Failed to scan holding: %v", err)
			s.respondWithError(w, http.StatusInternalServerError, "Failed to scan holding")
			return
		}

		// Get realized gains for this ticker
		realizedGainsQuery := `
			SELECT COALESCE(
				SUM(
					CASE 
						WHEN type = 'SELL' THEN (price - average_cost_before) * shares
						ELSE 0 
					END
				),
				0
			) as realized_gains
			FROM portfolio_transactions
			WHERE portfolio_id = $1
			AND ticker = $2
			AND type IN ('SELL')
		`
		err = s.db.QueryRow(realizedGainsQuery, portfolioID, h.Ticker).Scan(&h.RealizedGain)
		if err != nil {
			s.logger.Error("Failed to get realized gains for %s: %v", h.Ticker, err)
			h.RealizedGain = 0
		}

		// Get current price and calculate other metrics
		priceQuery := `
			SELECT price 
			FROM portfolio_transactions 
			WHERE portfolio_id = $1 
			AND ticker = $2 
			AND type IN ('BUY', 'SELL')
			ORDER BY transaction_at DESC, id DESC 
			LIMIT 1
		`
		err = s.db.QueryRow(priceQuery, portfolioID, h.Ticker).Scan(&h.CurrentPrice)
		if err != nil {
			s.logger.Error("Failed to get current price: %v", err)
			h.CurrentPrice = h.AverageCost
		}

		h.CostBasis = h.Shares * h.AverageCost
		h.MarketValue = h.Shares * h.CurrentPrice
		h.UnrealizedGain = h.MarketValue - h.CostBasis

		totalCost += h.CostBasis
		totalValue += h.MarketValue
		holdings = append(holdings, h)
	}

	// Calculate total realized gains
	totalRealizedGainsQuery := `
		SELECT COALESCE(
			SUM(
				CASE 
					WHEN type = 'SELL' THEN (price - average_cost_before) * shares
					ELSE 0 
				END
			),
			0
		) as total_realized_gains
		FROM portfolio_transactions
		WHERE portfolio_id = $1
		AND type IN ('SELL')
	`
	var totalRealizedGains float64
	err = s.db.QueryRow(totalRealizedGainsQuery, portfolioID).Scan(&totalRealizedGains)
	if err != nil {
		s.logger.Error("Failed to get total realized gains: %v", err)
		totalRealizedGains = 0
	}

	totalUnrealizedGains := totalValue - totalCost

	response := PortfolioHoldingsResponse{
		Holdings:       holdings,
		TotalCost:      totalCost,
		TotalValue:     totalValue,
		UnrealizedGain: totalUnrealizedGains,
		RealizedGain:   totalRealizedGains,
		TotalGain:      totalUnrealizedGains + totalRealizedGains,
	}

	s.logger.Debug("Found %d holdings", len(holdings))
	for _, h := range holdings {
		s.logger.Debug("Holding: %s - %.2f shares @ %.2f (avg cost: %.2f, realized gain: %.2f)",
			h.Ticker, h.Shares, h.CurrentPrice, h.AverageCost, h.RealizedGain)
	}

	s.respondWithJSON(w, http.StatusOK, response)
}

// GetTransactionHistory returns the transaction history for a portfolio
func (s *Server) GetTransactionHistory(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	portfolioID, err := strconv.Atoi(vars["id"])
	if err != nil {
		s.respondWithError(w, http.StatusBadRequest, "Invalid portfolio ID")
		return
	}

	query := `
		SELECT 
			id,
			type,
			ticker,
			shares,
			price,
			amount,
			fee,
			notes,
			transaction_at,
			cash_balance_after,
			shares_count_after,
			CASE 
				WHEN type = 'SELL' THEN (price - average_cost_before) * shares
				ELSE 0 
			END as realized_gain
		FROM portfolio_transactions
		WHERE portfolio_id = $1
		ORDER BY transaction_at DESC, id DESC
	`

	rows, err := s.db.Query(query, portfolioID)
	if err != nil {
		s.logger.Error("Failed to get transactions: %v", err)
		s.respondWithError(w, http.StatusInternalServerError, "Failed to get transactions")
		return
	}
	defer rows.Close()

	var transactions []TransactionHistoryItem
	for rows.Next() {
		var t TransactionHistoryItem
		var ticker, notes sql.NullString
		var shares, price sql.NullFloat64

		err := rows.Scan(
			&t.ID,
			&t.Type,
			&ticker,
			&shares,
			&price,
			&t.Amount,
			&t.Fee,
			&notes,
			&t.TransactionAt,
			&t.CashBalance,
			&t.SharesCount,
			&t.RealizedGain,
		)
		if err != nil {
			s.logger.Error("Failed to scan transaction: %v", err)
			continue
		}

		if ticker.Valid {
			t.Ticker = ticker.String
		}
		if shares.Valid {
			t.Shares = shares.Float64
		}
		if price.Valid {
			t.Price = price.Float64
		}
		if notes.Valid {
			t.Notes = notes.String
		}

		transactions = append(transactions, t)
	}

	s.respondWithJSON(w, http.StatusOK, transactions)
}

// DailyPortfolioValue represents a daily portfolio snapshot
type DailyPortfolioValue struct {
	Date            time.Time  `json:"date"`
	CashBalance     float64    `json:"cash_balance"`
	StockValue      float64    `json:"stock_value"`
	TotalValue      float64    `json:"total_value"`
	Deposits        float64    `json:"deposits"`
	Withdrawals     float64    `json:"withdrawals"`
	DailyChange     float64    `json:"daily_change"`
	AdjustedChange  float64    `json:"adjusted_change"`
	DailyChangePerc float64    `json:"daily_change_percentage"`
	Positions       []Position `json:"positions"`
}

// Position represents a stock position
type Position struct {
	Ticker    string  `json:"ticker"`
	Shares    float64 `json:"shares"`
	Price     float64 `json:"price"`
	Value     float64 `json:"value"`
	CostBasis float64 `json:"cost_basis"`
}

// DailyValuesRequest represents the request parameters for daily values
type DailyValuesRequest struct {
	StartDate string `json:"start_date"` // Format: YYYY-MM-DD
	EndDate   string `json:"end_date"`   // Format: YYYY-MM-DD
}

// GetPortfolioDailyValues returns daily portfolio values
func (s *Server) GetPortfolioDailyValues(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	portfolioID, err := strconv.Atoi(vars["id"])
	if err != nil {
		s.logger.Error("Invalid portfolio ID: %v", err)
		s.respondWithError(w, http.StatusBadRequest, "Invalid portfolio ID")
		return
	}

	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")

	// Validate dates are provided
	if startDate == "" || endDate == "" {
		s.respondWithError(w, http.StatusBadRequest, "Both start_date and end_date are required")
		return
	}

	// Parse dates
	parsedStartDate, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		s.respondWithError(w, http.StatusBadRequest, "Invalid start_date format. Use YYYY-MM-DD")
		return
	}

	parsedEndDate, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		s.respondWithError(w, http.StatusBadRequest, "Invalid end_date format. Use YYYY-MM-DD")
		return
	}

	// Validate end date doesn't exceed today
	today := time.Now().Truncate(24 * time.Hour)
	if parsedEndDate.After(today) {
		s.logger.Debug("End date %s exceeds today's date %s, using today instead",
			parsedEndDate.Format("2006-01-02"), today.Format("2006-01-02"))
		endDate = today.Format("2006-01-02")
	}

	// Validate start date is not after end date
	if parsedStartDate.After(parsedEndDate) {
		s.respondWithError(w, http.StatusBadRequest, "Start date cannot be after end date")
		return
	}

	// Simpler query for debugging
	query := `
		WITH dates AS (
			SELECT generate_series(
				$2::date,
				$3::date,
				'1 day'::interval
			)::date as date
		),
		latest_positions AS (
			SELECT DISTINCT ON (d.date)
				d.date,
				t.cash_balance_after as cash_balance,
				(
					SELECT COALESCE(SUM(amount), 0)
					FROM portfolio_transactions
					WHERE portfolio_id = $1
					AND type = 'DEPOSIT'
					AND DATE(transaction_at) = d.date
				) as deposits,
				(
					SELECT COALESCE(SUM(amount), 0)
					FROM portfolio_transactions
					WHERE portfolio_id = $1
					AND type = 'WITHDRAW'
					AND DATE(transaction_at) = d.date
				) as withdrawals
			FROM dates d
			LEFT JOIN LATERAL (
				SELECT cash_balance_after
				FROM portfolio_transactions
				WHERE portfolio_id = $1
				AND DATE(transaction_at) <= d.date
				ORDER BY transaction_at DESC, id DESC
				LIMIT 1
			) t ON true
		),
		daily_positions AS (
			SELECT 
				d.date,
				jsonb_agg(
					CASE WHEN p.ticker IS NOT NULL THEN
						jsonb_build_object(
							'ticker', p.ticker,
							'shares', p.shares,
							'price', p.price,
							'value', p.shares * p.price,
							'cost_basis', p.cost_basis
						)
					ELSE NULL END
				) FILTER (WHERE p.ticker IS NOT NULL) as positions
			FROM dates d
			LEFT JOIN LATERAL (
				SELECT DISTINCT ON (ticker)
					ticker,
					shares_count_after as shares,
					price,
					average_cost_after as cost_basis
				FROM portfolio_transactions
				WHERE portfolio_id = $1
				AND DATE(transaction_at) <= d.date
				AND type IN ('BUY', 'SELL')
				AND shares_count_after > 0
				ORDER BY ticker, transaction_at DESC, id DESC
			) p ON true
			GROUP BY d.date
		)
		SELECT 
			lp.date,
			COALESCE(lp.cash_balance, 0) as cash_balance,
			COALESCE(lp.deposits, 0) as deposits,
			COALESCE(lp.withdrawals, 0) as withdrawals,
			COALESCE(dp.positions, '[]'::jsonb) as positions
		FROM latest_positions lp
		LEFT JOIN daily_positions dp ON dp.date = lp.date
		ORDER BY lp.date
	`

	s.logger.Debug("Executing query with params: portfolioID=%d, startDate=%s, endDate=%s",
		portfolioID, startDate, endDate)

	rows, err := s.db.Query(query, portfolioID, startDate, endDate)
	if err != nil {
		s.logger.Error("Query error: %v", err) // Log the actual error
		s.respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get daily values: %v", err))
		return
	}
	defer rows.Close()

	var dailyValues []DailyPortfolioValue
	var previousValue float64

	for rows.Next() {
		var dv DailyPortfolioValue
		var positionsJSON []byte

		err := rows.Scan(
			&dv.Date,
			&dv.CashBalance,
			&dv.Deposits,
			&dv.Withdrawals,
			&positionsJSON,
		)
		if err != nil {
			s.logger.Error("Row scan error: %v", err) // Log scan errors
			continue
		}

		s.logger.Debug("Processing row: date=%v, cash=%v, deposits=%v, withdrawals=%v",
			dv.Date, dv.CashBalance, dv.Deposits, dv.Withdrawals)

		if positionsJSON != nil {
			if err := json.Unmarshal(positionsJSON, &dv.Positions); err != nil {
				s.logger.Error("JSON unmarshal error: %v", err) // Log unmarshal errors
				continue
			}
		}

		dv.StockValue = 0
		for _, pos := range dv.Positions {
			dv.StockValue += pos.Value
		}

		dv.TotalValue = dv.CashBalance + dv.StockValue

		if len(dailyValues) > 0 {
			dv.DailyChange = dv.TotalValue - previousValue
			dv.AdjustedChange = dv.DailyChange - dv.Deposits + dv.Withdrawals
			if previousValue != 0 {
				dv.DailyChangePerc = (dv.AdjustedChange / previousValue) * 100
			}
		}

		previousValue = dv.TotalValue
		dailyValues = append(dailyValues, dv)
	}

	if err = rows.Err(); err != nil {
		s.logger.Error("Rows iteration error: %v", err) // Log iteration errors
		s.respondWithError(w, http.StatusInternalServerError, "Error processing results")
		return
	}

	if len(dailyValues) == 0 {
		s.logger.Debug("No daily values found for the given date range")
		s.respondWithJSON(w, http.StatusOK, []DailyPortfolioValue{})
		return
	}

	s.respondWithJSON(w, http.StatusOK, dailyValues)
}

// GetPortfolioValue returns the current value of a portfolio and any error
func (s *Server) GetPortfolioValue(portfolioID int) (float64, error) {
	var cashBalance float64
	var stockValue float64

	// Get latest cash balance
	err := s.db.QueryRow(`
		SELECT COALESCE(cash_balance_after, 0)
		FROM portfolio_transactions
		WHERE portfolio_id = $1
		ORDER BY transaction_at DESC, id DESC
		LIMIT 1
	`, portfolioID).Scan(&cashBalance)
	if err != nil && err != sql.ErrNoRows {
		return 0, fmt.Errorf("failed to get cash balance: %v", err)
	}

	// Get stock positions value
	rows, err := s.db.Query(`
		WITH latest_positions AS (
			SELECT DISTINCT ON (ticker)
				ticker,
				shares_count_after as shares,
				price
			FROM portfolio_transactions
			WHERE portfolio_id = $1
			AND type IN ('BUY', 'SELL')
			ORDER BY ticker, transaction_at DESC, id DESC
		)
		SELECT COALESCE(SUM(shares * price), 0)
		FROM latest_positions
		WHERE shares > 0
	`, portfolioID)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	if rows.Next() {
		err = rows.Scan(&stockValue)
		if err != nil {
			return 0, err
		}
	}

	return cashBalance + stockValue, nil
}
