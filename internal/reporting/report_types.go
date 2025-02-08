package reporting

import "time"

// PerformanceReport represents a comprehensive portfolio performance report
type PerformanceReport struct {
	// Basic Info
	PortfolioID  int       `json:"portfolio_id"`
	Name         string    `json:"name"`
	ReportDate   time.Time `json:"report_date"`
	ReportPeriod string    `json:"report_period"` // e.g., "YTD", "1Y", "ALL"

	// Position Summary
	CurrentValue float64 `json:"current_value"`
	CashBalance  float64 `json:"cash_balance"`
	StocksValue  float64 `json:"stocks_value"`

	// Performance Summary
	RealizedGains   float64 `json:"realized_gains"`
	UnrealizedGains float64 `json:"unrealized_gains"`
	DividendIncome  float64 `json:"dividend_income"`
	TotalReturn     float64 `json:"total_return"`
	ReturnPercent   float64 `json:"return_percent"`

	// Cash Flow Summary
	Deposits    float64 `json:"deposits"`
	Withdrawals float64 `json:"withdrawals"`
	NetCashFlow float64 `json:"net_cash_flow"`

	// Performance Metrics
	IRR  float64 `json:"irr"`
	XIRR float64 `json:"xirr"`

	// Holdings Performance
	Holdings []HoldingPerformance `json:"holdings"`

	// Additional Performance Metrics
	DailyReturn   float64 `json:"daily_return"`
	WeeklyReturn  float64 `json:"weekly_return"`
	MonthlyReturn float64 `json:"monthly_return"`
	YTDReturn     float64 `json:"ytd_return"`
	OneYearReturn float64 `json:"one_year_return"`

	// Risk Metrics
	Volatility      float64    `json:"volatility"`
	SharpeRatio     float64    `json:"sharpe_ratio"`
	MaxDrawdown     float64    `json:"max_drawdown"`
	DrawdownPeriods []Drawdown `json:"drawdown_periods"`
}

// HoldingPerformance represents performance metrics for a single holding
type HoldingPerformance struct {
	Ticker         string    `json:"ticker"`
	Shares         float64   `json:"shares"`
	CurrentPrice   float64   `json:"current_price"`
	CurrentValue   float64   `json:"current_value"`
	CostBasis      float64   `json:"cost_basis"`
	UnrealizedGain float64   `json:"unrealized_gain"`
	RealizedGain   float64   `json:"realized_gain"`
	DividendIncome float64   `json:"dividend_income"`
	TotalReturn    float64   `json:"total_return"`
	ReturnPercent  float64   `json:"return_percent"`
	LastUpdate     time.Time `json:"last_update"`
}

// Add new types
type Drawdown struct {
	StartDate  time.Time `json:"start_date"`
	EndDate    time.Time `json:"end_date"`
	Percentage float64   `json:"percentage"`
	Duration   int       `json:"duration_days"`
}
