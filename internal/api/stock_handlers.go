package api

import (
	"database/sql"
	"fmt"
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
				) as prev_date,
				ARRAY_AGG(close_price) OVER (
					PARTITION BY ticker 
					ORDER BY date DESC
					ROWS BETWEEN 9 PRECEDING AND CURRENT ROW
				) as sparkline_prices
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
			lp.sparkline_prices,
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
		var sparklinePrices []float64
		if err := rows.Scan(
			&stock.Ticker,
			&stock.LastPrice,
			&currentDate,
			&prevDate,
			&stock.Change,
			&stock.ChangePercentage,
			&sparklinePrices,
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

		stock.SparklinePrices = sparklinePrices
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

	// Ensure that stocksList is always a slice, even if empty:
	if stocks == nil {
		stocks = []StockResponse{} // or proper slice type
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

	query := `
		SELECT 
			ticker,
			to_char(date, 'YYYY-MM-DD') as date,
			open_price,
			high_price,
			low_price,
			close_price,
			qty_of_shares_traded as shares_traded,
			value_of_shares_traded as value_traded,
			num_trades,
			change,
			change_percentage
		FROM daily_stock_prices
		WHERE ticker = $1
		ORDER BY date DESC
	`

	rows, err := s.db.Query(query, ticker)
	if err != nil {
		s.logger.Error("Failed to fetch stock prices: %v", err)
		s.respondWithError(w, http.StatusInternalServerError, "Failed to fetch stock prices")
		return
	}
	defer rows.Close()

	var prices []map[string]interface{}
	for rows.Next() {
		var (
			ticker        string
			date          string
			openPrice     float64
			highPrice     float64
			lowPrice      float64
			closePrice    float64
			sharesTraded  int64
			valueTraded   float64
			numTrades     int
			change        float64
			changePercent float64
		)

		err := rows.Scan(
			&ticker,
			&date,
			&openPrice,
			&highPrice,
			&lowPrice,
			&closePrice,
			&sharesTraded,
			&valueTraded,
			&numTrades,
			&change,
			&changePercent,
		)
		if err != nil {
			s.logger.Error("Failed to scan price row: %v", err)
			continue
		}

		prices = append(prices, map[string]interface{}{
			"ticker":            ticker,
			"date":              date,
			"open_price":        openPrice,
			"high_price":        highPrice,
			"low_price":         lowPrice,
			"close_price":       closePrice,
			"shares_traded":     sharesTraded,
			"value_traded":      valueTraded,
			"num_trades":        numTrades,
			"change":            change,
			"change_percentage": changePercent,
		})
	}

	if err = rows.Err(); err != nil {
		s.logger.Error("Error iterating price rows: %v", err)
		s.respondWithError(w, http.StatusInternalServerError, "Failed to process price data")
		return
	}

	// Get company name
	var companyName string
	err = s.db.QueryRow("SELECT company_name FROM tickers WHERE ticker = $1", ticker).Scan(&companyName)
	if err != nil {
		s.logger.Error("Failed to fetch company name: %v", err)
		companyName = ticker // fallback to ticker if company name not found
	}

	response := map[string]interface{}{
		"ticker":       ticker,
		"company_name": companyName,
		"prices":       prices,
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

// GetLatestStockPrices returns the latest record by date for each ticker.
func (s *Server) GetLatestStockPrices(w http.ResponseWriter, r *http.Request) {
	query := `
		WITH LatestPrices AS (
			SELECT 
				ticker,
				to_char(date, 'YYYY-MM-DD') as date,
				open_price,
				high_price,
				low_price,
				close_price,
				qty_of_shares_traded as shares_traded,
				value_of_shares_traded as value_traded,
				num_trades,
				change,
				change_percentage
			FROM daily_stock_prices dsp1
			WHERE date = (
				SELECT MAX(date)
				FROM daily_stock_prices dsp2
				WHERE dsp2.ticker = dsp1.ticker
			)
		)
		SELECT * FROM LatestPrices
		ORDER BY ticker
	`

	rows, err := s.db.Query(query)
	if err != nil {
		s.logger.Error("Failed to fetch stock prices: %v", err)
		s.respondWithError(w, http.StatusInternalServerError, "Failed to fetch stock prices")
		return
	}
	defer rows.Close()

	// Initialize as empty slice instead of nil
	results := make([]map[string]interface{}, 0)

	for rows.Next() {
		var (
			ticker        string
			date          string
			openPrice     float64
			highPrice     float64
			lowPrice      float64
			closePrice    float64
			sharesTraded  int64
			valueTraded   int64
			numTrades     int
			change        float64
			changePercent float64
		)

		if err := rows.Scan(
			&ticker,
			&date,
			&openPrice,
			&highPrice,
			&lowPrice,
			&closePrice,
			&sharesTraded,
			&valueTraded,
			&numTrades,
			&change,
			&changePercent,
		); err != nil {
			s.logger.Error("Failed to scan row: %v", err)
			continue
		}

		results = append(results, map[string]interface{}{
			"ticker":            ticker,
			"date":              date,
			"open_price":        openPrice,
			"high_price":        highPrice,
			"low_price":         lowPrice,
			"close_price":       closePrice,
			"shares_traded":     sharesTraded,
			"value_traded":      valueTraded,
			"num_trades":        numTrades,
			"change":            change,
			"change_percentage": changePercent,
		})
	}

	s.logger.Debug("Successfully fetched %d stock prices", len(results))
	s.respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"stocks": results,
		"total":  len(results),
	})
}

// GetStockDetails returns detailed information for a specific stock
func (s *Server) GetStockDetails(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ticker := vars["ticker"]

	s.logger.Debug("GetStockDetails called for ticker: %s", ticker)

	// First get the company name
	var companyName string
	err := s.db.QueryRow(`
		SELECT company_name 
		FROM tickers 
		WHERE ticker = $1
	`, ticker).Scan(&companyName)

	if err == sql.ErrNoRows {
		s.logger.Error("Stock not found: %s", ticker)
		s.respondWithError(w, http.StatusNotFound, fmt.Sprintf("Stock not found: %s", ticker))
		return
	}
	if err != nil {
		s.logger.Error("Failed to fetch company name: %v", err)
		s.respondWithError(w, http.StatusInternalServerError, "Database error")
		return
	}

	s.logger.Debug("Found company name: %s for ticker: %s", companyName, ticker)

	// Then get the historical price data with proper change calculations
	query := `
		WITH daily_changes AS (
			SELECT 
				d.ticker,
				d.date,
				d.open_price,
				d.high_price,
				d.low_price,
				d.close_price,
				d.qty_of_shares_traded,
				d.value_of_shares_traded,
				d.num_trades,
				COALESCE(d.close_price - LAG(d.close_price) OVER (
					PARTITION BY d.ticker 
					ORDER BY d.date
				), 0) as price_change,
				CASE 
					WHEN LAG(d.close_price) OVER (
						PARTITION BY d.ticker 
						ORDER BY d.date
					) > 0 THEN 
						((d.close_price - LAG(d.close_price) OVER (
							PARTITION BY d.ticker 
							ORDER BY d.date
						)) / LAG(d.close_price) OVER (
							PARTITION BY d.ticker 
							ORDER BY d.date
						)) * 100
					ELSE 0 
				END as change_percentage
			FROM daily_stock_prices d
			WHERE d.ticker = $1
		)
		SELECT 
			ticker,
			to_char(date, 'YYYY-MM-DD') as date,
			open_price,
			high_price,
			low_price,
			close_price,
			qty_of_shares_traded,
			value_of_shares_traded,
			num_trades,
			price_change as change,
			change_percentage
		FROM daily_changes
		ORDER BY date DESC
	`

	rows, err := s.db.Query(query, ticker)
	if err != nil {
		s.logger.Error("Failed to fetch stock data: %v", err)
		s.respondWithError(w, http.StatusInternalServerError, "Failed to fetch stock data")
		return
	}
	defer rows.Close()

	var prices []map[string]interface{}
	for rows.Next() {
		var p struct {
			Ticker        string
			Date          string
			OpenPrice     float64
			HighPrice     float64
			LowPrice      float64
			ClosePrice    float64
			SharesTraded  int64
			ValueTraded   float64
			NumTrades     int
			Change        float64
			ChangePercent float64
		}

		err := rows.Scan(
			&p.Ticker, &p.Date, &p.OpenPrice, &p.HighPrice, &p.LowPrice,
			&p.ClosePrice, &p.SharesTraded, &p.ValueTraded, &p.NumTrades,
			&p.Change, &p.ChangePercent,
		)
		if err != nil {
			s.logger.Error("Error scanning row: %v", err)
			continue
		}

		prices = append(prices, map[string]interface{}{
			"ticker":            p.Ticker,
			"date":              p.Date,
			"open_price":        p.OpenPrice,
			"high_price":        p.HighPrice,
			"low_price":         p.LowPrice,
			"close_price":       p.ClosePrice,
			"shares_traded":     p.SharesTraded,
			"value_traded":      p.ValueTraded,
			"num_trades":        p.NumTrades,
			"change":            p.Change,
			"change_percentage": p.ChangePercent,
		})
	}

	s.logger.Debug("Found %d price records for %s", len(prices), ticker)

	response := map[string]interface{}{
		"ticker":       ticker,
		"company_name": companyName,
		"prices":       prices,
	}

	s.respondWithJSON(w, http.StatusOK, response)
}

// GetStockSparkline returns the last 10 days of closing prices for a ticker
func (s *Server) GetStockSparkline(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ticker := vars["ticker"]

	query := `
		WITH recent_prices AS (
			SELECT 
				close_price,
				date,
				ROW_NUMBER() OVER (PARTITION BY ticker ORDER BY date DESC) as rn
			FROM daily_stock_prices
			WHERE ticker = $1
		)
		SELECT 
			close_price,
			to_char(date, 'YYYY-MM-DD') as date
		FROM recent_prices
		WHERE rn <= 10
		ORDER BY date ASC
	`

	rows, err := s.db.Query(query, ticker)
	if err != nil {
		s.logger.Error("Failed to fetch sparkline data: %v", err)
		s.respondWithError(w, http.StatusInternalServerError, "Failed to fetch sparkline data")
		return
	}
	defer rows.Close()

	var prices []float64
	var dates []string
	for rows.Next() {
		var price float64
		var date string
		if err := rows.Scan(&price, &date); err != nil {
			s.logger.Error("Failed to scan sparkline row: %v", err)
			s.respondWithError(w, http.StatusInternalServerError, "Failed to process sparkline data")
			return
		}
		prices = append(prices, price)
		dates = append(dates, date)
	}

	if len(prices) == 0 {
		s.logger.Debug("No sparkline data found for ticker: %s", ticker)
		prices = []float64{0} // Provide at least one point
		dates = []string{time.Now().Format("2006-01-02")}
	}

	s.logger.Debug("Sparkline data for %s: prices=%v, dates=%v", ticker, prices, dates)

	response := map[string]interface{}{
		"ticker": ticker,
		"prices": prices,
		"dates":  dates,
	}

	s.respondWithJSON(w, http.StatusOK, response)
}

// GetStockChartData returns data formatted for Echarts
func (s *Server) GetStockChartData(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ticker := vars["ticker"]

	query := `
		SELECT 
			to_char(date, 'YYYY-MM-DD') as date,
			open_price,
			close_price,
			low_price,
			high_price,
			qty_of_shares_traded
		FROM daily_stock_prices
		WHERE ticker = $1
		ORDER BY date ASC
	`

	rows, err := s.db.Query(query, ticker)
	if err != nil {
		s.logger.Error("Failed to fetch chart data: %v", err)
		s.respondWithError(w, http.StatusInternalServerError, "Failed to fetch chart data")
		return
	}
	defer rows.Close()

	var dates []string
	var volumes []int64
	var candleData [][]float64

	for rows.Next() {
		var date string
		var open, close, low, high float64
		var volume int64

		err := rows.Scan(&date, &open, &close, &low, &high, &volume)
		if err != nil {
			s.logger.Error("Failed to scan row: %v", err)
			continue
		}

		dates = append(dates, date)
		volumes = append(volumes, volume)
		candleData = append(candleData, []float64{open, close, low, high})
	}

	response := map[string]interface{}{
		"ticker":     ticker,
		"dates":      dates,
		"volumes":    volumes,
		"candleData": candleData,
	}

	s.respondWithJSON(w, http.StatusOK, response)
}
