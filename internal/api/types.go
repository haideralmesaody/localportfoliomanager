package api

import (
	"fmt"
	"strings"
	"time"
)

// StockResponse represents the structure for a single stock in the list
type StockResponse struct {
	Ticker           string    `json:"ticker"`
	LastPrice        float64   `json:"last_price"`
	Change           float64   `json:"change"`
	ChangePercentage float64   `json:"change_percentage"`
	SparklinePrices  []float64 `json:"sparkline_prices"`
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

// TransactionType represents the type of transaction
type TransactionType string

const (
	Deposit  TransactionType = "DEPOSIT"
	Withdraw TransactionType = "WITHDRAW"
	Buy      TransactionType = "BUY"
	Sell     TransactionType = "SELL"
	Dividend TransactionType = "DIVIDEND"
)

// Custom time type that can handle both formats
type JSONTime time.Time

func (t *JSONTime) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), "\"")

	// Try parsing with timezone
	tt, err := time.Parse(time.RFC3339, s)
	if err == nil {
		*t = JSONTime(tt)
		return nil
	}

	// Try parsing without timezone (assume UTC)
	tt, err = time.Parse("2006-01-02T15:04:05.999", s)
	if err == nil {
		*t = JSONTime(tt.UTC())
		return nil
	}

	return err
}

// Add Format method
func (t JSONTime) Format(layout string) string {
	return time.Time(t).Format(layout)
}

// Add method to convert to time.Time
func (t JSONTime) Time() time.Time {
	return time.Time(t)
}

// TransactionRequest represents the incoming transaction request
type TransactionRequest struct {
	Type          TransactionType `json:"type"`
	Ticker        string          `json:"ticker"`
	Shares        float64         `json:"shares"`
	Price         float64         `json:"price"`
	Amount        float64         `json:"amount"`
	Fee           float64         `json:"fee"`
	Notes         string          `json:"notes"`
	TransactionAt time.Time       `json:"transaction_at"`
}

// Validate checks if the transaction request is valid
func (r *TransactionRequest) Validate() error {
	switch r.Type {
	case Buy, Sell:
		if r.Ticker == "" {
			return fmt.Errorf("ticker is required for %s transactions", r.Type)
		}
		if r.Shares <= 0 {
			return fmt.Errorf("shares must be positive for %s transactions", r.Type)
		}
		if r.Price <= 0 {
			return fmt.Errorf("price must be positive for %s transactions", r.Type)
		}
		if r.Amount <= 0 {
			return fmt.Errorf("amount must be positive for %s transactions", r.Type)
		}
		if r.Fee < 0 {
			return fmt.Errorf("fee cannot be negative")
		}
	case Deposit, Withdraw:
		if r.Amount <= 0 {
			return fmt.Errorf("amount must be positive for %s transactions", r.Type)
		}
	default:
		return fmt.Errorf("invalid transaction type: %s", r.Type)
	}
	return nil
}

// Transaction represents a portfolio transaction
type Transaction struct {
	ID                int             `json:"id"`
	PortfolioID       int             `json:"portfolio_id"`
	Type              TransactionType `json:"type"`
	Ticker            string          `json:"ticker"`
	Shares            float64         `json:"shares"`
	Price             float64         `json:"price"`
	Amount            float64         `json:"amount"`
	Fee               float64         `json:"fee"`
	Notes             string          `json:"notes"`
	TransactionAt     time.Time       `json:"transaction_at"`
	CreatedAt         time.Time       `json:"created_at"`
	CashBalanceBefore float64         `json:"cash_balance_before"`
	CashBalanceAfter  float64         `json:"cash_balance_after"`
	SharesCountBefore float64         `json:"shares_count_before"`
	SharesCountAfter  float64         `json:"shares_count_after"`
	AverageCostBefore float64         `json:"average_cost_before"`
	AverageCostAfter  float64         `json:"average_cost_after"`
	RealizedGainAvg   float64         `json:"realized_gain_avg"`
	RealizedGainFIFO  float64         `json:"realized_gain_fifo"`
}

// TransactionResponse includes the transaction and calculated fields
type TransactionResponse struct {
	Transaction Transaction `json:"transaction"`
	TotalAmount float64     `json:"total_amount"`
}

// TransactionSummary represents portfolio transaction summary
type TransactionSummary struct {
	TotalDeposits    float64 `json:"total_deposits"`
	TotalWithdrawals float64 `json:"total_withdrawals"`
	TotalFees        float64 `json:"total_fees"`
	TotalDividends   float64 `json:"total_dividends"`
	RealizedGains    float64 `json:"realized_gains"`
	UnrealizedGains  float64 `json:"unrealized_gains"`
	NetCashFlow      float64 `json:"net_cash_flow"`
}

// TransactionsListResponse represents the response for listing transactions
type TransactionsListResponse struct {
	Transactions []TransactionResponse `json:"transactions"`
	Total        int                   `json:"total"`
	Summary      TransactionSummary    `json:"summary"`
}

type StockGains struct {
	RealizedGains    float64 `json:"realized_gains"`
	SoldShares       float64 `json:"sold_shares"`
	AverageBuyPrice  float64 `json:"average_buy_price"`
	AverageSellPrice float64 `json:"average_sell_price"`
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
	Ticker       string  `json:"ticker"`
	Shares       float64 `json:"shares"`
	AverageCost  float64 `json:"average_cost"`
	CurrentPrice float64 `json:"current_price"`
	CostBasis    float64 `json:"cost_basis"`
}

// Add this type definition
type LatestStockData struct {
	Ticker           string  `json:"ticker"`
	Date             string  `json:"date"`
	OpenPrice        float64 `json:"open_price"`
	HighPrice        float64 `json:"high_price"`
	LowPrice         float64 `json:"low_price"`
	ClosePrice       float64 `json:"close_price"`
	SharesTraded     int64   `json:"qty_of_shares_traded"`
	ValueTraded      float64 `json:"value_of_shares_traded"`
	NumberOfTrades   int     `json:"num_trades"`
	Change           float64 `json:"change"`
	ChangePercentage float64 `json:"change_percentage"`
}

// Add this type definition
type TransactionHistoryResponse struct {
	PortfolioID  int           `json:"portfolio_id"`
	Transactions []Transaction `json:"transactions"`
}

// Holding represents a portfolio holding
type Holding struct {
	ID                    int64      `json:"id"`
	PortfolioID           int64      `json:"portfolio_id"`
	Ticker                string     `json:"ticker"`
	Shares                float64    `json:"shares"`
	PurchaseCostAverage   float64    `json:"purchase_cost_average"`
	PurchaseCostFIFO      float64    `json:"purchase_cost_fifo"`
	CurrentPrice          float64    `json:"current_price"`
	PriceLastDate         time.Time  `json:"price_last_date"`
	PositionCostAverage   *float64   `json:"position_cost_average"`   // Make nullable
	PositionCostFIFO      *float64   `json:"position_cost_fifo"`      // Make nullable
	UnrealizedGainAverage *float64   `json:"unrealized_gain_average"` // Make nullable
	UnrealizedGainFIFO    *float64   `json:"unrealized_gain_fifo"`    // Make nullable
	TargetPercentage      float64    `json:"target_percentage"`
	CurrentPercentage     float64    `json:"current_percentage"`
	AdjustmentPercentage  float64    `json:"adjustment_percentage"`
	AdjustmentValue       float64    `json:"adjustment_value"`
	AdjustmentQuantity    int64      `json:"adjustment_quantity"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
	Lots                  []StockLot `json:"lots,omitempty"`
}

// StockLot represents a FIFO lot for stock purchases
type StockLot struct {
	ID              int       `json:"id"`
	PortfolioID     int       `json:"portfolio_id"`
	Ticker          string    `json:"ticker"`
	Shares          float64   `json:"shares"`
	RemainingShares float64   `json:"remaining_shares"`
	PurchasePrice   float64   `json:"purchase_price"`
	PurchaseDate    time.Time `json:"purchase_date"`
	CreatedAt       time.Time `json:"created_at"`
}

// StockTransaction represents a buy/sell transaction request
type StockTransaction struct {
	Type          TransactionType `json:"type"`
	Ticker        string          `json:"ticker"`
	Shares        float64         `json:"shares"`
	Price         float64         `json:"price"`
	Fee           float64         `json:"fee,omitempty"`
	Notes         string          `json:"notes,omitempty"`
	TransactionAt time.Time       `json:"transaction_at"`
}

// Add PortfolioSummary type
type PortfolioSummary struct {
	Name                string    `json:"name"`
	Description         string    `json:"description"`
	TotalValue          float64   `json:"total_value"`
	TotalCostAverage    float64   `json:"total_cost_average"`
	TotalCostFIFO       float64   `json:"total_cost_fifo"`
	TotalGainAverage    float64   `json:"total_gain_average"`
	TotalGainFIFO       float64   `json:"total_gain_fifo"`
	RealizedGainAverage float64   `json:"realized_gain_average"`
	RealizedGainFIFO    float64   `json:"realized_gain_fifo"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}
