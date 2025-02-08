package scraper

import (
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
	"localportfoliomanager/internal/utils"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	_ "github.com/lib/pq" // Add this line - PostgreSQL driver
)

// StockData represents the structure of our scraped data
type StockData struct {
	Date            string
	OpenPrice       string
	HighPrice       string
	LowPrice        string
	ClosePrice      string
	Volume          string
	TotalShares     string
	NumTrades       string
	Change          float64
	ChangePerc      float64
	SparklinePrices []float64
	SparklineDates  []string
}

type Scraper struct {
	logger      *utils.AppLogger
	ctx         context.Context
	cancel      context.CancelFunc
	config      *utils.Config
	perfTracker *utils.PerformanceTracker
	db          *sql.DB
}

func NewScraper(logger *utils.AppLogger, ctx context.Context, cancel context.CancelFunc, config *utils.Config) *Scraper {
	// Initialize the database connection here
	db, err := sql.Open("postgres", config.Database.DSN)
	if err != nil {
		fmt.Println("Error connecting to database:", err)
	}

	// Create context with options and suppress CDP logging
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("log-level", "3"),                  // Increase log level
		chromedp.Flag("headless", false),                 // Changed to false to disable headless mode
		chromedp.Flag("enable-logging", false),           // Disable CDP logging
		chromedp.Flag("silent-debugger", true),           // Silence debugger
		chromedp.Flag("suppress-cookie-errors", true),    // Suppress cookie errors
		chromedp.Flag("no-sandbox", true),                // Add no-sandbox flag
		chromedp.Flag("disable-setuid-sandbox", true),    // Disable setuid sandbox
		chromedp.Flag("ignore-certificate-errors", true), // Ignore cert errors
		chromedp.Flag("window-size", "1920,1080"),        // Set a larger window size
	)

	// Create allocator context with options
	allocCtx, cancel := chromedp.NewExecAllocator(ctx, opts...)

	// Create new context with custom logger that ignores cookie errors
	ctx, cancel = chromedp.NewContext(allocCtx,
		chromedp.WithLogf(func(format string, args ...interface{}) {
			// Only log if it's not a cookie error
			if !strings.Contains(strings.ToLower(fmt.Sprintf(format, args...)), "cookie") {
				logger.Debug(format, args...)
			}
		}),
	)

	return &Scraper{
		logger:      logger,
		ctx:         ctx,
		cancel:      cancel,
		config:      config,
		perfTracker: utils.NewPerformanceTracker(),
		db:          db,
	}
}

func (s *Scraper) GetStockData(ticker string) ([]StockData, error) {
	s.logger.Info("Starting data collection for ticker: %s", ticker)

	// Get latest date from database with better logging
	latestDate, err := s.getLatestDate(ticker)
	if err != nil {
		s.logger.Error("Error checking latest date: %v", err)
		return nil, err
	}
	s.logger.Info("Latest date in DB for %s: %s", ticker, latestDate)

	var allStockData []StockData
	currentPage := 1
	maxPages := s.config.Scraper.MaxPages
	foundOverlap := false
	consecutiveErrors := 0
	maxRetries := 3

	for currentPage <= maxPages && !foundOverlap {
		s.logger.Debug("Scraping page %d for ticker %s", currentPage, ticker)

		// Add delay between requests
		if currentPage > 1 {
			time.Sleep(3 * time.Second)
		}

		pageData, err := s.scrapePageData(currentPage, ticker)
		if err != nil {
			consecutiveErrors++
			s.logger.Error("Error on page %d for %s: %v", currentPage, ticker, err)

			if consecutiveErrors >= maxRetries {
				s.logger.Error("Max retries reached for ticker %s", ticker)
				break
			}

			time.Sleep(5 * time.Second)
			continue
		}

		consecutiveErrors = 0

		if pageData == nil || len(pageData) == 0 {
			s.logger.Debug("No more data found for ticker %s", ticker)
			break
		}

		// Log the dates we're comparing
		if len(pageData) > 0 {
			s.logger.Info("First scraped record date: %s, Latest DB date: %s",
				pageData[0].Date, latestDate)
		}

		// Process data and check for overlap with improved logging
		for _, record := range pageData {
			s.logger.Debug("Comparing dates - Record: %s, Latest DB: %s", record.Date, latestDate)

			// Parse dates for proper comparison
			recordDate, err := time.Parse("02/01/2006", record.Date)
			if err != nil {
				s.logger.Error("Failed to parse record date %s: %v", record.Date, err)
				continue
			}

			dbDate, err := time.Parse("2006-01-02", latestDate)
			if err != nil {
				s.logger.Error("Failed to parse DB date %s: %v", latestDate, err)
				continue
			}

			// Compare dates properly
			if !recordDate.After(dbDate) {
				foundOverlap = true
				s.logger.Info("Found overlap - Record date: %s not after DB date: %s",
					recordDate.Format("02/01/2006"), dbDate.Format("02/01/2006"))
				break
			}

			s.logger.Debug("Adding new record for date: %s", record.Date)
			allStockData = append(allStockData, record)
		}

		if foundOverlap {
			break
		}

		currentPage++
	}

	s.logger.Info("Collected %d new records for ticker %s", len(allStockData), ticker)
	if len(allStockData) > 0 {
		s.logger.Info("New records date range: %s to %s",
			allStockData[len(allStockData)-1].Date,
			allStockData[0].Date)
	}

	return allStockData, nil
}

func (s *Scraper) SaveToCSV(ticker string, data []StockData) error {
	if len(data) == 0 {
		return fmt.Errorf("no data to save")
	}

	// Create the output directory if it doesn't exist
	err := os.MkdirAll("output", 0755)
	if err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	// Create CSV file
	filename := fmt.Sprintf("output/%s_data.csv", ticker)
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create CSV file: %v", err)
	}
	defer file.Close()

	// Create CSV writer
	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header with new column
	headers := []string{"Date", "Open", "High", "Low", "Close", "Change", "Change%", "Volume", "T.Shares", "Trades"}
	if err := writer.Write(headers); err != nil {
		return fmt.Errorf("failed to write headers: %v", err)
	}

	// Write data including new fields
	for _, record := range data {
		row := []string{
			record.Date,
			record.OpenPrice,
			record.HighPrice,
			record.LowPrice,
			record.ClosePrice,
			fmt.Sprintf("%.3f", record.Change),       // Change
			fmt.Sprintf("%.2f%%", record.ChangePerc), // Change%
			record.Volume,
			record.TotalShares,
			record.NumTrades,
		}
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("failed to write record: %v", err)
		}
	}

	s.logger.Info("Successfully saved data to %s", filename)
	return nil
}

func (s *Scraper) Close() {
	fmt.Println("\nClosing browser...")
	if s.cancel != nil {
		// Create a new context with a short timeout for cleanup
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Try to close gracefully
		if err := chromedp.Run(ctx, chromedp.Stop()); err != nil {
			fmt.Printf("Error during graceful shutdown: %v\n", err)
		}

		s.cancel()
		time.Sleep(2 * time.Second)
		fmt.Println("Browser closed successfully")
	}
}

func (s *Scraper) GetPerformanceTracker() *utils.PerformanceTracker {
	return s.perfTracker
}

// PreflightCheck verifies all dependencies and configurations
func (s *Scraper) PreflightCheck() error {
	checks := []struct {
		name  string
		check func() error
	}{
		{"Config Validation", s.validateConfig},
		{"Directory Structure", s.checkDirectories},
		{"Browser Launch", s.testBrowserLaunch},
		{"Network Settings", s.testNetworkSettings},
	}

	for _, c := range checks {
		s.logger.Debug("Running preflight check: %s", c.name)
		if err := c.check(); err != nil {
			return fmt.Errorf("%s check failed: %v", c.name, err)
		}
		s.logger.Debug("%s check passed", c.name)
	}

	return nil
}

func (s *Scraper) validateConfig() error {
	if s.config == nil {
		return fmt.Errorf("configuration is nil")
	}
	if s.config.Scraper.Timeout <= 0 {
		return fmt.Errorf("invalid timeout value")
	}
	if s.config.Scraper.MaxPages <= 0 {
		return fmt.Errorf("invalid max pages value")
	}
	return nil
}

func (s *Scraper) checkDirectories() error {
	dirs := []string{
		"output",
		"logs",
		"temp_builds",
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create %s directory: %v", dir, err)
		}
	}
	return nil
}

func (s *Scraper) testBrowserLaunch() error {
	ctx, cancel := context.WithTimeout(s.ctx, 10*time.Second)
	defer cancel()

	return chromedp.Run(ctx, chromedp.Navigate("about:blank"))
}

func (s *Scraper) testNetworkSettings() error {
	ctx, cancel := context.WithTimeout(s.ctx, 10*time.Second)
	defer cancel()

	return chromedp.Run(ctx,
		network.Enable(),
		network.SetCacheDisabled(true),
		emulation.SetUserAgentOverride("Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/91.0.4472.124 Safari/537.36"),
	)
}

// Add browser refresh mechanism
func (s *Scraper) refreshBrowser() error {
	s.logger.Debug("Refreshing browser session")

	// Cancel old context
	if s.cancel != nil {
		s.cancel()
	}

	// Create new context and browser
	ctx, cancel := chromedp.NewContext(context.Background())
	s.ctx = ctx
	s.cancel = cancel

	// Test new browser
	err := chromedp.Run(ctx, chromedp.Navigate("about:blank"))
	if err != nil {
		return fmt.Errorf("failed to refresh browser: %v", err)
	}

	return nil
}

// Add this function before processTickerList
func processSingleTicker(s *Scraper, logger *utils.AppLogger, ticker string) error {
	logger.Info("Processing ticker: %s", ticker)

	// Get stock data
	stockDataList, err := s.GetStockData(ticker)
	if err != nil {
		logger.Error("Error processing %s: %v", ticker, err)
		return err
	}

	// Save the fetched data to a CSV file
	err = s.SaveToCSV(ticker, stockDataList)
	if err != nil {
		logger.Error("Error saving data for %s: %v", ticker, err)
		return err
	}

	logger.Info("Successfully processed %s. Data saved to output/%s_data.csv", ticker, ticker)
	return nil
}

// Update processTickerList in main.go to handle browser refresh
func processTickerList(s *Scraper, logger *utils.AppLogger, tickers []string) error {
	totalTickers := len(tickers)
	logger.Info("Starting to process %d tickers", totalTickers)

	for i, ticker := range tickers {
		logger.Info("Processing ticker %d/%d: %s", i+1, totalTickers, ticker)

		// Refresh browser every 5 tickers
		if i > 0 && i%5 == 0 {
			logger.Debug("Performing browser refresh")
			if err := s.refreshBrowser(); err != nil {
				logger.Error("Failed to refresh browser: %v", err)
				time.Sleep(30 * time.Second) // Hard-coded 30 second wait
				continue
			}
		}

		err := processSingleTicker(s, logger, ticker)
		if err != nil {
			logger.Error("Failed to process ticker %s: %v", ticker, err)
			// If navigation fails, try refreshing the browser
			if err.Error() == "failed to navigate: context canceled" {
				logger.Debug("Navigation failed, refreshing browser")
				if err := s.refreshBrowser(); err != nil {
					logger.Error("Failed to refresh browser: %v", err)
				}
			}
			time.Sleep(10 * time.Second)
			continue
		}

		if i < totalTickers-1 {
			logger.Debug("Waiting 10 seconds before next ticker")
			time.Sleep(10 * time.Second)
		}
	}

	// Generate and log aggregate performance report
	report := s.GetPerformanceTracker().GenerateAggregateReport()
	logger.Info("Aggregate Performance Report:\n%s", report)

	logger.Info("Completed processing %d tickers", totalTickers)
	return nil
}

// Change from calculatePriceChanges to CalculatePriceChanges
func (s *Scraper) CalculatePriceChanges(data []StockData) []StockData {
	s.logger.Debug("Starting price change calculations for %d records", len(data))

	if len(data) < 2 {
		s.logger.Debug("Not enough data for change calculations (need at least 2 records)")
		return data
	}

	// Process each day's data starting from most recent (data is in reverse chronological order)
	for i := 0; i < len(data)-1; i++ {
		currentClose, err := strconv.ParseFloat(data[i].ClosePrice, 64)
		if err != nil {
			s.logger.Error("Error parsing current close price for date %s: %v", data[i].Date, err)
			continue
		}

		previousClose, err := strconv.ParseFloat(data[i+1].ClosePrice, 64)
		if err != nil {
			s.logger.Error("Error parsing previous close price for date %s: %v", data[i+1].Date, err)
			continue
		}

		// Calculate change (current - previous)
		data[i].Change = currentClose - previousClose

		// Calculate change percentage ((current - previous) / previous) * 100
		if previousClose != 0 {
			data[i].ChangePerc = (data[i].Change / previousClose) * 100
		}

		s.logger.Debug("Date: %s, Current: %.3f, Previous: %.3f, Change: %.3f, Change%%: %.2f%%",
			data[i].Date, currentClose, previousClose, data[i].Change, data[i].ChangePerc)
	}

	// Handle the last (oldest) record
	if len(data) > 0 {
		lastIdx := len(data) - 1
		data[lastIdx].Change = 0
		data[lastIdx].ChangePerc = 0
		s.logger.Debug("Set change to 0 for oldest record date: %s", data[lastIdx].Date)
	}

	return data
}

// Update loadExistingData to get last 25 records
func (s *Scraper) loadExistingData(ticker string) ([]StockData, error) {
	rows, err := s.db.Query(`
		SELECT 
			TO_CHAR(date, 'DD/MM/YYYY') as formatted_date,
			open_price,
			high_price,
			low_price,
			close_price,
			qty_of_shares_traded,
			value_of_shares_traded,
			num_trades,
			change,
			change_percentage,
			(
				SELECT array_agg(close_price ORDER BY date)
				FROM (
					SELECT close_price, date 
					FROM daily_stock_prices 
					WHERE ticker = $1 
					ORDER BY date DESC 
					LIMIT 30
				) sub
			) as sparkline_prices,
			(
				SELECT array_agg(TO_CHAR(date, 'DD/MM/YYYY') ORDER BY date)
				FROM (
					SELECT date 
					FROM daily_stock_prices 
					WHERE ticker = $1 
					ORDER BY date DESC 
					LIMIT 30
				) sub
			) as sparkline_dates
		FROM daily_stock_prices 
		WHERE ticker = $1 
		ORDER BY date DESC
		LIMIT 25`, ticker)
	if err != nil {
		return nil, fmt.Errorf("failed to query existing data: %w", err)
	}
	defer rows.Close()

	var data []StockData
	for rows.Next() {
		var record StockData
		var openPrice, highPrice, lowPrice, closePrice float64
		var volume, totalShares, numTrades int64

		err := rows.Scan(
			&record.Date,
			&openPrice,
			&highPrice,
			&lowPrice,
			&closePrice,
			&volume,
			&totalShares,
			&numTrades,
			&record.Change,
			&record.ChangePerc,
			&record.SparklinePrices,
			&record.SparklineDates,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Convert numeric values back to strings to match StockData struct
		record.OpenPrice = fmt.Sprintf("%.3f", openPrice)
		record.HighPrice = fmt.Sprintf("%.3f", highPrice)
		record.LowPrice = fmt.Sprintf("%.3f", lowPrice)
		record.ClosePrice = fmt.Sprintf("%.3f", closePrice)
		record.Volume = fmt.Sprintf("%d", volume)
		record.TotalShares = fmt.Sprintf("%d", totalShares)
		record.NumTrades = fmt.Sprintf("%d", numTrades)

		data = append(data, record)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	s.logger.Debug("Loaded %d existing records for %s from database", len(data), ticker)
	return data, nil
}

// Update findOverlap to handle multiple records
func (s *Scraper) findOverlap(existingData []StockData, newData []StockData) (bool, []StockData) {
	if len(existingData) == 0 || len(newData) == 0 {
		return false, newData
	}

	// Create a map of existing dates for faster lookup
	existingDates := make(map[string]bool)
	for _, record := range existingData {
		existingDates[record.Date] = true
	}

	// Find new records that don't exist in the database
	var nonOverlappedData []StockData
	foundOverlap := false

	for _, record := range newData {
		if existingDates[record.Date] {
			foundOverlap = true
			// Once we find an overlap, we can stop as older records will also overlap
			break
		}
		nonOverlappedData = append(nonOverlappedData, record)
	}

	s.logger.Debug("Found %d new records before overlap", len(nonOverlappedData))
	return foundOverlap, nonOverlappedData
}

// deleteLastThreeRecords deletes the last 3 records for a given ticker.
// Note: PostgreSQL does not allow ORDER BY in a DELETE statement,
// so we use a subquery with ctid.
func (s *Scraper) deleteLastThreeRecords(ticker string) error {
	query := `
		DELETE FROM daily_stock_prices
		WHERE ctid IN (
			SELECT ctid
			FROM daily_stock_prices
			WHERE ticker = $1
			ORDER BY date DESC
			LIMIT 3
		)
	`
	_, err := s.db.Exec(query, ticker)
	if err != nil {
		return fmt.Errorf("failed to delete records: %w", err)
	}
	return nil
}

// ValidateAndSaveStockData validates and saves new/updated stock data to the database.
// This method also calculates price changes.
func (s *Scraper) ValidateAndSaveStockData(ticker string, data []StockData) error {
	// Calculate changes before saving
	data = s.CalculatePriceChanges(data)

	// Begin transaction
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // Rollback if not committed

	stmt, err := tx.Prepare(`
		INSERT INTO daily_stock_prices (
			date, ticker, open_price, high_price, low_price, close_price,
			qty_of_shares_traded, value_of_shares_traded, num_trades, change, change_percentage
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (date, ticker) DO UPDATE SET
			open_price = EXCLUDED.open_price,
			high_price = EXCLUDED.high_price,
			low_price = EXCLUDED.low_price,
			close_price = EXCLUDED.close_price,
			qty_of_shares_traded = EXCLUDED.qty_of_shares_traded,
			value_of_shares_traded = EXCLUDED.value_of_shares_traded,
			num_trades = EXCLUDED.num_trades,
			change = EXCLUDED.change,
			change_percentage = EXCLUDED.change_percentage,
			updated_at = CURRENT_TIMESTAMP
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, record := range data {
		parsedDate, err := time.Parse("02/01/2006", record.Date)
		if err != nil {
			s.logger.Debug("Failed to parse date %s: %v", record.Date, err)
			continue
		}

		openPrice := parseFloat(record.OpenPrice)
		highPrice := parseFloat(record.HighPrice)
		lowPrice := parseFloat(record.LowPrice)
		closePrice := parseFloat(record.ClosePrice)
		volume := parseInt(record.Volume)
		totalShares := parseInt(record.TotalShares)
		numTrades := parseInt(record.NumTrades)

		_, err = stmt.Exec(
			parsedDate,
			ticker,
			openPrice,
			highPrice,
			lowPrice,
			closePrice,
			volume,
			totalShares,
			numTrades,
			record.Change,
			record.ChangePerc,
		)
		if err != nil {
			return fmt.Errorf("failed to insert stock data: %w", err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.logger.Info("Successfully saved %d records for %s to the database", len(data), ticker)
	return nil
}

// Add this function to get tickers from the database
func (s *Scraper) GetTickersFromDB() ([]string, error) {
	// Create database connection using config DSN
	db, err := sql.Open("postgres", s.config.Database.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	// Query tickers table
	rows, err := db.Query("SELECT ticker FROM tickers")
	if err != nil {
		return nil, fmt.Errorf("failed to query tickers: %w", err)
	}
	defer rows.Close()

	var tickers []string
	for rows.Next() {
		var ticker string
		if err := rows.Scan(&ticker); err != nil {
			return nil, fmt.Errorf("failed to scan ticker: %w", err)
		}
		tickers = append(tickers, ticker)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tickers: %w", err)
	}

	s.logger.Debug("Loaded %d tickers from database", len(tickers))
	return tickers, nil
}

// Add these helper functions at the top level of the file

func parseFloat(s string) float64 {
	// Remove any commas and spaces
	s = strings.ReplaceAll(s, ",", "")
	s = strings.TrimSpace(s)

	// Handle empty or invalid strings
	if s == "" || s == "-" {
		return 0
	}

	// Try to parse the float
	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return val
}

func parseInt(s string) int64 {
	// Remove any commas and spaces
	s = strings.ReplaceAll(s, ",", "")
	s = strings.TrimSpace(s)

	// Handle empty or invalid strings
	if s == "" || s == "-" {
		return 0
	}

	// Try to parse the integer
	val, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return val
}

// Add this new method
func (s *Scraper) RecalculateAllPriceChanges() error {
	s.logger.Info("Starting recalculation of all price changes")

	// Call the stored procedure
	_, err := s.db.Exec("CALL recalculate_price_changes()")
	if err != nil {
		return fmt.Errorf("failed to recalculate price changes: %w", err)
	}

	s.logger.Info("Successfully recalculated all price changes")
	return nil
}

// ScrapeStockPrices scrapes current stock prices for all tickers
func (s *Scraper) ScrapeStockPrices() error {
	s.logger.Info("Starting stock price scraping...")
	defer s.logger.Info("Completed stock price scraping")

	// Get list of tickers from database
	tickers, err := s.GetTickersFromDB()
	if err != nil {
		s.logger.Error("Failed to get tickers from database: %v", err)
		return fmt.Errorf("failed to get tickers: %v", err)
	}

	if len(tickers) == 0 {
		s.logger.Error("No tickers found in database")
		return fmt.Errorf("no tickers found")
	}

	s.logger.Info("Found %d tickers in database", len(tickers))

	// Process each ticker
	for _, ticker := range tickers {
		s.logger.Info("Processing ticker: %s", ticker)

		stockDataList, err := s.GetStockData(ticker)
		if err != nil {
			s.logger.Error("Failed to get stock data for %s: %v", ticker, err)
			continue
		}

		// Calculate price changes
		stockDataList = s.CalculatePriceChanges(stockDataList)

		// Save to database
		err = s.ValidateAndSaveStockData(ticker, stockDataList)
		if err != nil {
			s.logger.Error("Failed to save data for %s: %v", ticker, err)
			continue
		}

		s.logger.Info("Successfully processed ticker: %s", ticker)
	}

	// After all tickers are processed, recalculate changes
	if err := s.RecalculateAllPriceChanges(); err != nil {
		s.logger.Error("Failed to recalculate price changes: %v", err)
		return fmt.Errorf("failed to recalculate price changes: %v", err)
	}

	s.logger.Info("Scraping completed successfully")
	return nil
}

// 1. First, add a function to get the latest date we have
func (s *Scraper) getLatestDate(ticker string) (string, error) {
	var latestDate sql.NullTime
	err := s.db.QueryRow(`
		SELECT MAX(date) 
		FROM daily_stock_prices 
		WHERE ticker = $1
	`, ticker).Scan(&latestDate)

	if err != nil {
		return "", err
	}

	if !latestDate.Valid {
		s.logger.Info("No existing data found for ticker %s", ticker)
		return "2000-01-01", nil // Return a very old date if no data exists
	}

	formattedDate := latestDate.Time.Format("2006-01-02")
	s.logger.Info("Latest date in database for %s: %s", ticker, formattedDate)
	return formattedDate, nil
}

// 2. Modify the GetStockData function to use overlap detection
func (s *Scraper) scrapePageData(currentPage int, ticker string) ([]StockData, error) {
	s.logger.Debug("Starting scrapePageData for ticker: %s, page: %d", ticker, currentPage)

	// Create a new context with a longer timeout (60 seconds instead of 30)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create new browser context for each scrape
	allocCtx, allocCancel := chromedp.NewExecAllocator(ctx, chromedp.DefaultExecAllocatorOptions[:]...)
	defer allocCancel()

	browserCtx, browserCancel := chromedp.NewContext(allocCtx)
	defer browserCancel()

	url := fmt.Sprintf("http://www.isx-iq.net/isxportal/portal/companyprofilecontainer.html?currLanguage=en&companyCode=%s%%20&activeTab=0", ticker)

	// Add more robust error handling and retries for navigation
	var navigationError error
	for attempts := 0; attempts < 3; attempts++ {
		err := chromedp.Run(browserCtx,
			chromedp.Navigate(url),
			chromedp.WaitReady("body", chromedp.ByQuery),
		)
		if err == nil {
			navigationError = nil
			break
		}
		navigationError = err
		time.Sleep(2 * time.Second)
	}

	if navigationError != nil {
		return nil, fmt.Errorf("failed to navigate after retries: %v", navigationError)
	}

	// Add explicit waits and checks for form elements
	err := chromedp.Run(browserCtx,
		chromedp.WaitVisible("#fromDate", chromedp.ByID),
		chromedp.WaitVisible("#command > div.filterbox > div.button-all > input[type=button]", chromedp.ByQuery),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to find form elements: %v", err)
	}

	// Set date and trigger search with better error handling
	err = chromedp.Run(browserCtx,
		chromedp.SetValue("#fromDate", "01/01/2020", chromedp.ByID),
		chromedp.Sleep(1*time.Second),
		chromedp.Click("#command > div.filterbox > div.button-all > input[type=button]", chromedp.ByQuery),
		chromedp.Sleep(2*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to set date and search: %v", err)
	}

	// Wait for and verify table data
	var hasData bool
	err = chromedp.Run(browserCtx,
		chromedp.WaitVisible("#dispTable", chromedp.ByID),
		chromedp.Evaluate(`!!document.querySelector("#dispTable tbody tr")`, &hasData),
	)
	if err != nil || !hasData {
		return nil, fmt.Errorf("table data not found or error: %v", err)
	}

	// Extract data with improved error handling
	var pageData []StockData
	err = chromedp.Run(browserCtx,
		chromedp.Evaluate(`
			(() => {
				try {
					const rows = document.querySelectorAll("#dispTable tbody tr");
					if (!rows || rows.length === 0) return null;
					
					const data = [];
					for (const row of rows) {
						const cells = row.querySelectorAll("td");
						if (cells.length < 10) continue;
						
						data.push({
							Date: cells[9].textContent.trim(),
							OpenPrice: cells[7].textContent.trim(),
							HighPrice: cells[6].textContent.trim(),
							LowPrice: cells[5].textContent.trim(),
							ClosePrice: cells[8].textContent.trim(),
							Volume: cells[1].textContent.trim(),
							TotalShares: cells[2].textContent.trim(),
							NumTrades: cells[0].textContent.trim()
						});
					}
					return data.length > 0 ? data : null;
				} catch (e) {
					console.error("Scraping error:", e);
					return null;
				}
			})()
		`, &pageData),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to extract data: %v", err)
	}

	if pageData == nil || len(pageData) == 0 {
		s.logger.Debug("No data found for ticker %s on page %d", ticker, currentPage)
		return nil, nil
	}

	s.logger.Debug("Successfully scraped %d records for ticker %s on page %d", len(pageData), ticker, currentPage)
	return pageData, nil
}

// Add this helper function
func (s *Scraper) setDateRangeAndSearch(ctx context.Context) (bool, error) {
	var success bool
	err := chromedp.Run(ctx,
		chromedp.Evaluate(`
			(() => {
				try {
					// Set date range
					const dateInput = document.querySelector("#fromDate");
					if (!dateInput) throw new Error("Date input not found");
					
					dateInput.value = "01/01/2020";
					dateInput.dispatchEvent(new Event('change', { bubbles: true }));
					
					// Click search button
					const searchButton = document.querySelector("#command > div.filterbox > div.button-all > input[type=button]");
					if (!searchButton) throw new Error("Search button not found");
					
					searchButton.click();
					return true;
				} catch (e) {
					console.error('Setup error:', e);
					return false;
				}
			})()
		`, &success),
	)

	return success, err
}
