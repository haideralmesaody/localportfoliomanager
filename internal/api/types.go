package api

import (
	"time"
)

// StockResponse represents the structure for a single stock in the list
type StockResponse struct {
	Ticker           string  `json:"ticker"`
	LastPrice        float64 `json:"last_price"`
	Change           float64 `json:"change"`
	ChangePercentage float64 `json:"change_percentage"`
}

// StocksListResponse represents the paginated response for the stocks list
type StocksListResponse struct {
	Stocks []StockResponse `json:"stocks"`
	Total  int             `json:"total"`
}

// StockDetailResponse represents the structure for stock details
type StockDetailResponse struct {
	Ticker           string    `json:"ticker"`
	LastPrice        float64   `json:"last_price"`
	Open             float64   `json:"open"`
	High             float64   `json:"high"`
	Low              float64   `json:"low"`
	Volume           int64     `json:"volume"`
	Change           float64   `json:"change"`
	ChangePercentage float64   `json:"change_percentage"`
	LastUpdated      time.Time `json:"last_updated"`
}

// Add these new types for historical prices
type StockPriceData struct {
	Date             time.Time `json:"date"`
	Open             float64   `json:"open"`
	High             float64   `json:"high"`
	Low              float64   `json:"low"`
	Close            float64   `json:"close"`
	Volume           int64     `json:"volume"`
	Change           float64   `json:"change"`
	ChangePercentage float64   `json:"change_percentage"`
}

type StockPricesResponse struct {
	Ticker   string           `json:"ticker"`
	Interval string           `json:"interval"`
	Prices   []StockPriceData `json:"prices"`
}

// Transaction types
type TransactionType string

const (
	Buy      TransactionType = "BUY"
	Sell     TransactionType = "SELL"
	Deposit  TransactionType = "DEPOSIT"
	Withdraw TransactionType = "WITHDRAW"
	Dividend TransactionType = "DIVIDEND"
)

// Transaction represents a portfolio transaction
type Transaction struct {
	ID                int             `json:"id"`
	PortfolioID       int             `json:"portfolio_id"`
	Type              TransactionType `json:"type"`
	Ticker            string          `json:"ticker,omitempty"`
	Shares            float64         `json:"shares,omitempty"`
	Price             float64         `json:"price,omitempty"`
	Amount            float64         `json:"amount"`
	Fee               float64         `json:"fee"`
	Notes             string          `json:"notes,omitempty"`
	TransactionAt     time.Time       `json:"transaction_at"`
	CreatedAt         time.Time       `json:"created_at"`
	CashBalanceBefore float64         `json:"cash_balance_before"`
	CashBalanceAfter  float64         `json:"cash_balance_after"`
	SharesCountBefore float64         `json:"shares_count_before,omitempty"`
	SharesCountAfter  float64         `json:"shares_count_after,omitempty"`
	AverageCostBefore float64         `json:"average_cost_before,omitempty"`
	AverageCostAfter  float64         `json:"average_cost_after,omitempty"`
}

// CreateTransactionRequest represents the request to create a new transaction
type CreateTransactionRequest struct {
	Type          TransactionType `json:"type" validate:"required,oneof=DEPOSIT WITHDRAW BUY SELL DIVIDEND"`
	Ticker        string          `json:"ticker,omitempty"`
	Shares        float64         `json:"shares,omitempty"`
	Price         float64         `json:"price,omitempty"`
	Amount        float64         `json:"amount" validate:"required,gt=0"`
	Fee           float64         `json:"fee" validate:"gte=0"`
	Notes         string          `json:"notes,omitempty"`
	TransactionAt time.Time       `json:"transaction_at" validate:"required"`
}

// TransactionResponse represents a transaction with additional calculated fields
type TransactionResponse struct {
	Transaction
	TotalAmount float64 `json:"total_amount"` // Amount + Fee
}

// Add new type for transaction summary
type TransactionSummary struct {
	TotalDeposits        float64               `json:"total_deposits"`
	TotalWithdrawals     float64               `json:"total_withdrawals"`
	TotalFees            float64               `json:"total_fees"`
	TotalDividends       float64               `json:"total_dividends"`
	RealizedGains        float64               `json:"realized_gains"`
	RealizedGainsByStock map[string]StockGains `json:"realized_gains_by_stock"`
	UnrealizedGains      float64               `json:"unrealized_gains"`
	NetCashFlow          float64               `json:"net_cash_flow"`
}

type StockGains struct {
	RealizedGains    float64 `json:"realized_gains"`
	SoldShares       float64 `json:"sold_shares"`
	AverageBuyPrice  float64 `json:"average_buy_price"`
	AverageSellPrice float64 `json:"average_sell_price"`
}

// Update TransactionsListResponse
type TransactionsListResponse struct {
	Transactions []TransactionResponse `json:"transactions"`
	Total        int                   `json:"total"`
	Summary      TransactionSummary    `json:"summary"`
}

// Portfolio performance types
type PortfolioPerformance struct {
	StartDate        time.Time `json:"start_date"`
	EndDate          time.Time `json:"end_date"`
	StartValue       float64   `json:"start_value"`
	EndValue         float64   `json:"end_value"`
	NetContributions float64   `json:"net_contributions"`
	RealizedGains    float64   `json:"realized_gains"`
	UnrealizedGains  float64   `json:"unrealized_gains"`
	DividendIncome   float64   `json:"dividend_income"`
	TWR              float64   `json:"time_weighted_return"`  // Time-weighted return
	MWR              float64   `json:"money_weighted_return"` // Money-weighted return
}

// TransactionHistoryItem represents a transaction in the history
type TransactionHistoryItem struct {
	ID            int       `json:"id"`
	Type          string    `json:"type"`
	Ticker        string    `json:"ticker,omitempty"`
	Shares        float64   `json:"shares,omitempty"`
	Price         float64   `json:"price,omitempty"`
	Amount        float64   `json:"amount"`
	Fee           float64   `json:"fee"`
	Notes         string    `json:"notes,omitempty"`
	TransactionAt time.Time `json:"transaction_at"`
	CashBalance   float64   `json:"cash_balance"`
	SharesCount   float64   `json:"shares_count,omitempty"`
	RealizedGain  float64   `json:"realized_gain,omitempty"`
}

// PortfolioHolding represents a single stock holding
type PortfolioHolding struct {
	Ticker         string  `json:"ticker"`
	Shares         float64 `json:"shares"`
	AverageCost    float64 `json:"average_cost"`
	CurrentPrice   float64 `json:"current_price"`
	CostBasis      float64 `json:"cost_basis"`
	MarketValue    float64 `json:"market_value"`
	UnrealizedGain float64 `json:"unrealized_gain"`
	RealizedGain   float64 `json:"realized_gain"`
}

// PortfolioHoldingsResponse represents the portfolio holdings response
type PortfolioHoldingsResponse struct {
	Holdings       []PortfolioHolding `json:"holdings"`
	TotalCost      float64            `json:"total_cost"`
	TotalValue     float64            `json:"total_value"`
	UnrealizedGain float64            `json:"unrealized_gain"`
	RealizedGain   float64            `json:"realized_gain"`
	TotalGain      float64            `json:"total_gain"`
}
