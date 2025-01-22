package api

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

// Stock Endpoints Documentation
//
// Base Path: /api/stocks
//
// Available Endpoints:
// 1. GET /api/stocks
//    - Returns a list of all available stocks in the system
//    - Query Parameters:
//      * limit (optional): Number of stocks to return (default: 50)
//      * offset (optional): Number of stocks to skip (default: 0)
//    - Response: Array of stocks with basic information
//    - Example Response:
//      {
//        "stocks": [
//          {
//            "ticker": "BBOB",
//            "last_price": 0.450,
//            "change": 0.005,
//            "change_percentage": 1.12
//          }
//        ],
//        "total": 57
//      }
//
// 2. GET /api/stocks/{ticker}
//    - Returns detailed information for a specific stock
//    - Parameters:
//      * ticker: Stock symbol (e.g., BBOB, BCOI)
//    - Response: Detailed stock information including latest prices
//    - Example Response:
//      {
//        "ticker": "BBOB",
//        "last_price": 0.450,
//        "open": 0.445,
//        "high": 0.455,
//        "low": 0.440,
//        "volume": 1000000,
//        "change": 0.005,
//        "change_percentage": 1.12,
//        "last_updated": "2024-02-20T15:30:00Z"
//      }
//
// 3. GET /api/stocks/{ticker}/prices
//    - Returns historical price data for a specific stock
//    - Parameters:
//      * ticker: Stock symbol
//    - Query Parameters:
//      * from (optional): Start date (YYYY-MM-DD)
//      * to (optional): End date (YYYY-MM-DD)
//      * interval (optional): Data interval (daily, weekly, monthly)
//    - Response: Array of historical prices
//    - Example Response:
//      {
//        "ticker": "BBOB",
//        "interval": "daily",
//        "prices": [
//          {
//            "date": "2024-02-20",
//            "open": 0.445,
//            "high": 0.455,
//            "low": 0.440,
//            "close": 0.450,
//            "volume": 1000000
//          }
//        ]
//      }

// GetStocks returns a list of all available stocks
func (s *Server) GetStocks(w http.ResponseWriter, r *http.Request) {
	// Get pagination parameters
	limit := 50 // default limit
	offset := 0 // default offset

	// Log the request
	s.logger.Debug("Getting stocks with limit: %d, offset: %d", limit, offset)

	if limitParam := r.URL.Query().Get("limit"); limitParam != "" {
		if parsedLimit, err := strconv.Atoi(limitParam); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}
	if offsetParam := r.URL.Query().Get("offset"); offsetParam != "" {
		if parsedOffset, err := strconv.Atoi(offsetParam); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	// Query to get latest stock prices
	query := `
		WITH LatestPrices AS (
			SELECT DISTINCT ON (ticker)
				ticker,
				date as current_date,
				close_price as current_price,
				LAG(close_price, 1) OVER (
					PARTITION BY ticker 
					ORDER BY date DESC
				) as prev_price,
				LAG(date, 1) OVER (
					PARTITION BY ticker 
					ORDER BY date DESC
				) as prev_date
			FROM daily_stock_prices
			ORDER BY ticker, date DESC
		)
		SELECT 
			lp.ticker,
			lp.current_price as last_price,
			lp.current_date,
			lp.prev_date,
			COALESCE(lp.current_price - lp.prev_price, 0) as change,
			CASE 
				WHEN lp.prev_price > 0 THEN 
					ROUND(((lp.current_price - lp.prev_price) / lp.prev_price) * 100, 2)
				ELSE 0 
			END as change_percentage,
			COUNT(*) OVER() as total_count
		FROM LatestPrices lp
		ORDER BY lp.ticker
		LIMIT $1 OFFSET $2
	`

	// Execute query
	rows, err := s.db.Query(query, limit, offset)
	if err != nil {
		s.logger.Error("Failed to query stocks: %v", err)
		s.respondWithError(w, http.StatusInternalServerError, "Failed to fetch stocks")
		return
	}
	defer rows.Close()

	var stocks []StockResponse
	var total int

	// Log before scanning rows
	s.logger.Debug("Starting to scan rows")

	// Iterate through results
	for rows.Next() {
		var stock StockResponse
		var currentDate, prevDate sql.NullTime
		if err := rows.Scan(
			&stock.Ticker,
			&stock.LastPrice,
			&currentDate,
			&prevDate,
			&stock.Change,
			&stock.ChangePercentage,
			&total,
		); err != nil {
			s.logger.Error("Failed to scan stock row: %v", err)
			s.respondWithError(w, http.StatusInternalServerError, "Failed to process stocks data")
			return
		}

		s.logger.Debug("Stock %s: Current[%v]=$%.2f, Prev[%v], Change=$%.2f (%.2f%%)",
			stock.Ticker,
			currentDate.Time.Format("2006-01-02"),
			stock.LastPrice,
			prevDate.Time.Format("2006-01-02"),
			stock.Change,
			stock.ChangePercentage,
		)

		stocks = append(stocks, stock)
	}

	// Log after scanning rows
	s.logger.Debug("Found %d stocks", len(stocks))

	if err = rows.Err(); err != nil {
		s.logger.Error("Error iterating stock rows: %v", err)
		s.respondWithError(w, http.StatusInternalServerError, "Failed to process stocks data")
		return
	}

	response := StocksListResponse{
		Stocks: stocks,
		Total:  total,
	}

	// Log before sending response
	s.logger.Debug("Sending response with %d stocks", len(stocks))

	s.respondWithJSON(w, http.StatusOK, response)
}

// GetStockByTicker returns detailed information for a specific stock
func (s *Server) GetStockByTicker(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ticker := vars["ticker"]

	query := `
		WITH price_data AS (
			SELECT 
				ticker,
				date,
				open_price as open,
				high_price as high,
				low_price as low,
				close_price as close,
				qty_of_shares_traded as volume
			FROM daily_stock_prices
			WHERE ticker = $1
			ORDER BY date DESC
			LIMIT 2
		),
		latest_prices AS (
			SELECT 
				pd1.ticker,
				pd1.date,
				pd1.open,
				pd1.high,
				pd1.low,
				pd1.close,
				pd1.volume,
				pd1.close as current_price,
				pd2.close as prev_price,
				pd2.date as prev_date
			FROM (
				SELECT * FROM price_data LIMIT 1
			) pd1
			LEFT JOIN (
				SELECT * FROM price_data OFFSET 1 LIMIT 1
			) pd2 ON pd1.ticker = pd2.ticker
		)
		SELECT 
			ticker,
			close as last_price,
			open,
			high,
			low,
			volume,
			COALESCE(current_price - prev_price, 0) as change,
			CASE 
				WHEN prev_price > 0 THEN 
					ROUND(((current_price - prev_price) / prev_price) * 100, 2)
				ELSE 0 
			END as change_percentage,
			date as last_updated,
			prev_date
		FROM latest_prices
	`

	var stock StockDetailResponse
	var prevDate sql.NullTime
	err := s.db.QueryRow(query, ticker).Scan(
		&stock.Ticker,
		&stock.LastPrice,
		&stock.Open,
		&stock.High,
		&stock.Low,
		&stock.Volume,
		&stock.Change,
		&stock.ChangePercentage,
		&stock.LastUpdated,
		&prevDate,
	)

	if err == sql.ErrNoRows {
		s.respondWithError(w, http.StatusNotFound, "Stock not found")
		return
	}
	if err != nil {
		s.logger.Error("Failed to fetch stock details: %v", err)
		s.respondWithError(w, http.StatusInternalServerError, "Failed to fetch stock details")
		return
	}

	s.logger.Debug("Stock %s: Current[%v]=$%.2f, Prev[%v], Change=$%.2f (%.2f%%)",
		stock.Ticker,
		stock.LastUpdated.Format("2006-01-02"),
		stock.LastPrice,
		prevDate.Time.Format("2006-01-02"),
		stock.Change,
		stock.ChangePercentage,
	)

	s.respondWithJSON(w, http.StatusOK, stock)
}

// GetStockPrices returns historical price data for a specific stock
func (s *Server) GetStockPrices(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ticker := vars["ticker"]

	// Get query parameters
	fromDateStr := r.URL.Query().Get("from")
	toDateStr := r.URL.Query().Get("to")
	interval := r.URL.Query().Get("interval")

	// Default to daily interval if not specified
	if interval == "" {
		interval = "daily"
	}

	// Parse dates
	var fromDate, toDate time.Time
	var err error

	if fromDateStr != "" {
		fromDate, err = time.Parse("2006-01-02", fromDateStr)
		if err != nil {
			s.respondWithError(w, http.StatusBadRequest, "Invalid from date format")
			return
		}
	}

	if toDateStr != "" {
		toDate, err = time.Parse("2006-01-02", toDateStr)
		if err != nil {
			s.respondWithError(w, http.StatusBadRequest, "Invalid to date format")
			return
		}
	} else {
		toDate = time.Now()
	}

	// Add date validation
	if toDate.After(time.Now()) {
		toDate = time.Now()
	}

	// Ensure single day queries work
	if !fromDate.IsZero() && fromDate.Equal(toDate) {
		toDate = toDate.Add(24 * time.Hour)
	}

	// Build query with proper change calculation
	query := `
		WITH prices AS (
			SELECT 
				date,
				open_price as open,
				high_price as high,
				low_price as low,
				close_price as close,
				qty_of_shares_traded as volume,
				LAG(close_price) OVER (ORDER BY date DESC) as prev_close
			FROM daily_stock_prices
			WHERE ticker = $1
			AND date >= COALESCE($2, date - INTERVAL '1 year')
			AND date <= $3
		)
		SELECT 
			date, 
			open, 
			high, 
			low, 
			close, 
			volume,
			COALESCE(close - prev_close, 0) as change,
			CASE 
				WHEN prev_close > 0 THEN ROUND(((close - prev_close) / prev_close) * 100, 2)
				ELSE 0 
			END as change_percentage
		FROM prices
		ORDER BY date DESC
	`

	// Execute query with proper date handling
	var queryArgs []interface{}
	queryArgs = append(queryArgs, ticker)
	if !fromDate.IsZero() {
		queryArgs = append(queryArgs, fromDate)
	} else {
		queryArgs = append(queryArgs, nil)
	}
	queryArgs = append(queryArgs, toDate)

	rows, err := s.db.Query(query, queryArgs...)
	if err != nil {
		s.logger.Error("Failed to query stock prices: %v", err)
		s.respondWithError(w, http.StatusInternalServerError, "Failed to fetch stock prices")
		return
	}
	defer rows.Close()

	var prices []StockPriceData
	for rows.Next() {
		var price StockPriceData
		if err := rows.Scan(
			&price.Date,
			&price.Open,
			&price.High,
			&price.Low,
			&price.Close,
			&price.Volume,
			&price.Change,
			&price.ChangePercentage,
		); err != nil {
			s.logger.Error("Failed to scan price row: %v", err)
			s.respondWithError(w, http.StatusInternalServerError, "Failed to process price data")
			return
		}
		prices = append(prices, price)
	}

	response := StockPricesResponse{
		Ticker:   ticker,
		Interval: interval,
		Prices:   prices,
	}

	s.respondWithJSON(w, http.StatusOK, response)
}

// isTradeDay checks if the given date is a trading day
func isTradeDay(date time.Time) bool {
	// Check if it's a weekend
	if date.Weekday() == time.Saturday || date.Weekday() == time.Sunday {
		return false
	}

	// Add holiday checks here
	holidays := map[string]bool{
		"2024-01-01": true, // New Year
		"2024-01-06": true, // Epiphany
		"2024-03-31": true, // Easter
		"2024-05-01": true, // Labor Day
		"2024-12-25": true, // Christmas
	}

	return !holidays[date.Format("2006-01-02")]
}
