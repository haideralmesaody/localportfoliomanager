package main

import (
	"context"
	"localportfoliomanager/internal/utils"
	"localportfoliomanager/scraper"
	"log"

	"github.com/chromedp/chromedp"
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

func main() {
	// Initialize logger
	logger := utils.NewLogger()
	logger.Debug("Initializing new scraper...")

	// Load configuration
	config, err := utils.LoadConfig()
	if err != nil {
		logger.Fatal("Failed to load configuration: %v", err)
	}

	// Initialize Chrome
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	// Create new scraper
	scraper := scraper.NewScraper(logger, ctx, cancel, config)

	// Get list of tickers from database
	tickers, err := scraper.GetTickersFromDB()
	if err != nil {
		logger.Fatal("Failed to get tickers from database: %v", err)
	}

	if len(tickers) == 0 {
		logger.Fatal("No tickers found in database")
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
	logger.Info("All tickers processed. Starting price changes recalculation...")
	if err := scraper.RecalculateAllPriceChanges(); err != nil {
		logger.Error("Failed to recalculate price changes: %v", err)
	}

	logger.Info("Scraping completed successfully")
}
