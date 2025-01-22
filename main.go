package main

import (
	"context"
	"localportfoliomanager/internal/api"
	"localportfoliomanager/internal/utils"
	"localportfoliomanager/scraper"
	"log"

	"github.com/chromedp/chromedp"
	"github.com/robfig/cron/v3"
	"github.com/spf13/viper"
)

func initConfig() *utils.Config {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file: %s", err)
	}

	var config utils.Config
	if err := viper.Unmarshal(&config); err != nil {
		log.Fatalf("Error unmarshaling config: %s", err)
	}

	// Build the DSN string
	config.BuildDSN()

	return &config
}

func runScraper(logger *utils.Logger, config *utils.Config) {
	// Initialize Chrome
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	// Create new scraper
	scraper := scraper.NewScraper(logger, ctx, cancel, config)

	// Get list of tickers from database
	tickers, err := scraper.GetTickersFromDB()
	if err != nil {
		logger.Error("Failed to get tickers from database: %v", err)
		return
	}

	if len(tickers) == 0 {
		logger.Error("No tickers found in database")
		return
	}

	logger.Info("Found %d tickers in database", len(tickers))

	// Process each ticker
	for _, ticker := range tickers {
		logger.Info("Processing ticker: %s", ticker)

		stockDataList, err := scraper.GetStockData(ticker)
		if err != nil {
			logger.Error("Failed to get stock data for %s: %v", ticker, err)
			continue
		}

		// Calculate price changes
		stockDataList = scraper.CalculatePriceChanges(stockDataList)

		// Save to database
		err = scraper.SaveStockData(ticker, stockDataList)
		if err != nil {
			logger.Error("Failed to save data for %s: %v", ticker, err)
			continue
		}

		logger.Info("Successfully processed ticker: %s", ticker)
	}

	// After all tickers are processed, recalculate changes
	if err := scraper.RecalculateAllPriceChanges(); err != nil {
		logger.Error("Failed to recalculate price changes: %v", err)
	}

	logger.Info("Scraping completed successfully")
}

func main() {
	// Initialize logger
	logger := utils.NewLogger()
	logger.Info("Starting Local Portfolio Manager...")

	// Load configuration
	config, err := utils.LoadConfig()
	if err != nil {
		logger.Fatal("Failed to load configuration: %v", err)
	}

	// Create API server
	server := api.NewServer(logger, config)

	// Initialize Chrome context for scraper
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	// Initialize scraper
	stockScraper := scraper.NewScraper(logger, ctx, cancel, config)

	// Run initial scrape
	logger.Info("Running initial stock price scrape...")
	go func() {
		if err := stockScraper.ScrapeStockPrices(); err != nil {
			logger.Error("Initial scrape failed: %v", err)
		}
	}()

	// Set up cron job for hourly scraping
	c := cron.New()
	// Run scraper every hour during trading hours (10 AM to 6 PM Iraq time)
	c.AddFunc("0 10-18 * * 1-5", func() {
		logger.Info("Starting scheduled hourly scrape...")
		if err := stockScraper.ScrapeStockPrices(); err != nil {
			logger.Error("Scheduled scrape failed: %v", err)
		}
	})
	c.Start()
	logger.Info("Scraper scheduler started")

	// Start the server (this will block)
	logger.Info("Starting API server...")
	if err := server.Start(); err != nil {
		logger.Fatal("Server failed to start: %v", err)
	}
}
