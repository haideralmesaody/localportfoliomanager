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
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	_ "github.com/lib/pq" // Add this line - PostgreSQL driver
)

// StockData represents the structure of our scraped data
type StockData struct {
	Date        string
	OpenPrice   string
	HighPrice   string
	LowPrice    string
	ClosePrice  string
	Volume      string
	TotalShares string
	NumTrades   string
	Change      float64
	ChangePerc  float64
}

type Scraper struct {
	logger      *utils.Logger
	ctx         context.Context
	cancel      context.CancelFunc
	config      *utils.Config
	perfTracker *utils.PerformanceTracker
}

func NewScraper(logger *utils.Logger, ctx context.Context, cancel context.CancelFunc, config *utils.Config) *Scraper {
	return &Scraper{
		logger:      logger,
		ctx:         ctx,
		cancel:      cancel,
		config:      config,
		perfTracker: utils.NewPerformanceTracker(),
	}
}

func (s *Scraper) GetStockData(ticker string) ([]StockData, error) {
	// Add debug logging for config values
	s.logger.Debug("Scraper config: MaxPages=%d, Timeout=%d, Delay=%d",
		s.config.Scraper.MaxPages,
		s.config.Scraper.Timeout,
		s.config.Scraper.Delay)

	// Try to load existing data
	existingData, err := s.loadExistingData(ticker)
	if err != nil {
		s.logger.Debug("Error loading existing data: %v", err)
		// Continue with full scrape if there's an error
	}

	// Disable image loading before navigation
	err = chromedp.Run(s.ctx,
		network.Enable(),
		emulation.SetCPUThrottlingRate(1),
		network.SetExtraHTTPHeaders(map[string]interface{}{
			"Accept": "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
		}),
		network.SetBlockedURLS([]string{
			"*.png",
			"*.jpg",
			"*.jpeg",
			"*.gif",
			"*.webp",
			"*.svg",
			"*.ico",
		}),
	)
	if err != nil {
		s.logger.Debug("Failed to set image blocking: %v", err)
		// Continue anyway as this is not critical
	}

	url := fmt.Sprintf("http://www.isx-iq.net/isxportal/portal/companyprofilecontainer.html?currLanguage=en&companyCode=%s%%20&activeTab=0", ticker)
	fmt.Printf("Starting data extraction for ticker: %s\n", ticker)

	// Add dialog handler before navigation
	chromedp.ListenTarget(s.ctx, func(ev interface{}) {
		if ev, ok := ev.(*page.EventJavascriptDialogOpening); ok {
			s.logger.Debug("Dialog detected: %s", ev.Message)
			go func() {
				if err := chromedp.Run(s.ctx,
					page.HandleJavaScriptDialog(true),
				); err != nil {
					s.logger.Debug("Failed to handle dialog: %v", err)
				}
			}()
		}
	})

	// Navigate to the page
	err = chromedp.Run(s.ctx,
		chromedp.Navigate(url),
		chromedp.WaitReady("body"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to navigate: %v", err)
	}

	// Set up date range and trigger search
	err = chromedp.Run(s.ctx,
		chromedp.Evaluate(`
                (() => {
                    const dateInput = document.querySelector("#fromDate");
                    dateInput.value = "01/01/2020"; 
                    const event = new Event('change', { bubbles: true });
                    dateInput.dispatchEvent(event);

                    const searchButton = document.querySelector("#command > div.filterbox > div.button-all > input[type=button]");
                    searchButton.click();
                    return true;
                })()
            `, nil),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to set date range: %v", err)
	}

	// Wait for table to load
	time.Sleep(2 * time.Second)

	var allStockData []StockData
	currentPage := 1
	maxPages := s.config.Scraper.MaxPages
	foundOverlap := false
	previousPageCount := 0 // Track previous page record count

	fmt.Printf("Starting data extraction, will process %d pages\n", maxPages)

	for currentPage <= maxPages && !foundOverlap {
		// Extract data from current page
		var pageData []StockData
		err = chromedp.Run(s.ctx,
			chromedp.Evaluate(`
                (() => {
                    const table = document.getElementById('dispTable');
                    const rows = table.querySelectorAll('tbody tr');
                    return Array.from(rows).map(row => {
                        const cells = row.querySelectorAll('td');
                        return {
                            Date: cells[9].textContent.trim(),
                            OpenPrice: cells[7].textContent.trim(),
                            HighPrice: cells[6].textContent.trim(),
                            LowPrice: cells[5].textContent.trim(),
                            ClosePrice: cells[8].textContent.trim(),
                            Volume: cells[1].textContent.trim(),
                            TotalShares: cells[2].textContent.trim(),
                            NumTrades: cells[0].textContent.trim()
                        };
                    });
                })()
            `, &pageData),
		)
		if err != nil {
			fmt.Printf("Error extracting data from page %d: %v\n", currentPage, err)
			return nil, fmt.Errorf("failed to extract data from page %d: %v", currentPage, err)
		}

		// Check if we've reached the end of data
		if len(pageData) == 0 {
			s.logger.Debug("No more data found on page %d, stopping extraction", currentPage)
			break
		}

		// Check if we got fewer records than the previous page (last page usually has fewer records)
		if previousPageCount > 0 && len(pageData) < previousPageCount {
			s.logger.Debug("Found last page with %d records (previous had %d)", len(pageData), previousPageCount)
		}
		previousPageCount = len(pageData)

		fmt.Printf("Successfully extracted %d records from page %d\n", len(pageData), currentPage)

		if len(existingData) > 0 {
			foundOverlap, newPageData := s.findOverlap(existingData, pageData)
			if foundOverlap {
				s.logger.Debug("Found overlap with existing data on page %d", currentPage)
				pageData = newPageData
				if len(pageData) == 0 {
					s.logger.Info("No new data to add, stopping extraction")
					return nil, nil
				}
			}
		}

		allStockData = append(allStockData, pageData...)

		if foundOverlap {
			s.logger.Info("Found overlap with existing data, stopping extraction")
			break
		}

		// Check if we've reached the end of data
		if len(pageData) < 25 { // Assuming 25 is the standard page size
			s.logger.Debug("Reached last page (incomplete page), stopping extraction")
			break
		}

		if currentPage >= maxPages {
			break
		}

		// Navigate to next page
		nextPage := currentPage + 1
		fmt.Printf("Navigating to page %d...\n", nextPage)
		err = chromedp.Run(s.ctx,
			chromedp.Evaluate(fmt.Sprintf(`
                (() => {
                    doAjax('companyperformancehistoryfilter.html',
                           'fromDate=01/01/2020&d-6716032-p=%d&toDate=23/12/2024&companyCode=%s',
                           'ajxDspId');
                    return true;
                })()
            `, nextPage, ticker), nil),
		)
		if err != nil {
			fmt.Printf("Failed to navigate to page %d: %v\n", nextPage, err)
			break
		}

		time.Sleep(time.Duration(s.config.Scraper.Delay) * time.Second)
		currentPage++
	}

	// Append existing data if we have any
	if len(existingData) > 0 {
		allStockData = append(allStockData, existingData...)
	}

	// Calculate changes for all data
	allStockData = s.CalculatePriceChanges(allStockData)

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
func processSingleTicker(s *Scraper, logger *utils.Logger, ticker string) error {
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
func processTickerList(s *Scraper, logger *utils.Logger, tickers []string) error {
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
	if len(data) < 2 {
		return data
	}

	// Process each day's data starting from most recent (data is in reverse chronological order)
	for i := 0; i < len(data)-1; i++ {
		currentClose, err := strconv.ParseFloat(data[i].ClosePrice, 64)
		if err != nil {
			s.logger.Debug("Error parsing current close price: %v", err)
			continue
		}

		previousClose, err := strconv.ParseFloat(data[i+1].ClosePrice, 64)
		if err != nil {
			s.logger.Debug("Error parsing previous close price: %v", err)
			continue
		}

		// Calculate change and change percentage
		data[i].Change = currentClose - previousClose
		if previousClose != 0 {
			data[i].ChangePerc = (data[i].Change / previousClose) * 100
		}
	}

	return data
}

// Update loadExistingData to get last 25 records
func (s *Scraper) loadExistingData(ticker string) ([]StockData, error) {
	// Create database connection
	db, err := sql.Open("postgres", s.config.Database.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	// Get last 25 records for proper overlap detection
	rows, err := db.Query(`
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
			change_percentage
		FROM daily_stock_prices 
		WHERE ticker = $1 
		ORDER BY date DESC
		LIMIT 25`, ticker) // Changed to 25 records
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

// Add SaveStockData method
func (s *Scraper) SaveStockData(ticker string, data []StockData) error {
	s.logger.Info("Saving stock data for %s to the database", ticker)

	// Calculate changes before saving
	data = s.CalculatePriceChanges(data)

	// Create database connection
	db, err := sql.Open("postgres", s.config.Database.DSN)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	// Begin transaction
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // Rollback if we don't commit

	// Prepare the insert statement
	stmt, err := tx.Prepare(`
		INSERT INTO daily_stock_prices (
			date, ticker, open_price, high_price, low_price, close_price,
			qty_of_shares_traded, value_of_shares_traded, num_trades, change, change_percentage
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (date, ticker) 
		DO UPDATE SET
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

	// Insert each record
	for _, record := range data {
		// Parse date
		parsedDate, err := time.Parse("02/01/2006", record.Date)
		if err != nil {
			s.logger.Debug("Failed to parse date %s: %v", record.Date, err)
			continue
		}

		// Parse numeric values
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

	// Commit transaction
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

	// Create database connection
	db, err := sql.Open("postgres", s.config.Database.DSN)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	// Call the stored procedure
	_, err = db.Exec("CALL recalculate_price_changes()")
	if err != nil {
		return fmt.Errorf("failed to recalculate price changes: %w", err)
	}

	s.logger.Info("Successfully recalculated all price changes")
	return nil
}
