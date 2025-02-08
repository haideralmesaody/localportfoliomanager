package reporting

import (
	"database/sql"
	"fmt"
	"math"
	"time"
)

// ReportingService handles portfolio performance calculations and reporting
type ReportingService struct {
	db *sql.DB
}

func NewReportingService(db *sql.DB) *ReportingService {
	return &ReportingService{db: db}
}

// GeneratePerformanceReport creates a comprehensive performance report
func (s *ReportingService) GeneratePerformanceReport(portfolioID int, period string) (*PerformanceReport, error) {
	fmt.Printf("Starting report generation for portfolio %d\n", portfolioID)

	var report PerformanceReport

	// Get basic portfolio info
	err := s.db.QueryRow(`
		SELECT id, name 
		FROM portfolios 
		WHERE id = $1
	`, portfolioID).Scan(&report.PortfolioID, &report.Name)

	if err != nil {
		return nil, fmt.Errorf("failed to get portfolio: %v", err)
	}

	fmt.Printf("Found portfolio: %s (ID: %d)\n", report.Name, report.PortfolioID)

	// Set report metadata
	report.ReportDate = time.Now()
	report.ReportPeriod = period

	// Get current positions and values
	fmt.Println("Getting current positions...")
	err = s.getCurrentPositions(portfolioID, &report)
	if err != nil {
		return nil, fmt.Errorf("failed to get positions: %v", err)
	}
	fmt.Printf("Current positions: Cash=%f, Stocks=%f\n", report.CashBalance, report.StocksValue)

	// Get performance metrics
	fmt.Println("Getting performance metrics...")
	err = s.getPerformanceMetrics(portfolioID, period, &report)
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics: %v", err)
	}

	fmt.Printf("Performance metrics: Return=%f%%\n", report.ReturnPercent)

	// Calculate returns
	irr, xirr, err := s.CalculateReturns(portfolioID, s.getPeriodStartDate(period), time.Now())
	if err != nil {
		return nil, err
	}
	report.IRR = irr
	report.XIRR = xirr

	// Calculate additional metrics
	err = s.calculateAdditionalMetrics(portfolioID, &report)
	if err != nil {
		return nil, err
	}

	return &report, nil
}

func (s *ReportingService) getCurrentPositions(portfolioID int, report *PerformanceReport) error {
	fmt.Printf("Querying positions for portfolio %d\n", portfolioID)

	rows, err := s.db.Query(`
		WITH latest_prices AS (
			SELECT ticker, close_price, date
			FROM daily_stock_prices
			WHERE (ticker, date) IN (
				SELECT ticker, MAX(date)
				FROM daily_stock_prices
				GROUP BY ticker
			)
		)
		SELECT 
			h.ticker,
			h.shares,
			COALESCE(lp.close_price, h.current_price, h.purchase_cost_average, 0) as current_price,
			h.shares * COALESCE(lp.close_price, h.current_price, h.purchase_cost_average, 0) as current_value,
			h.shares * COALESCE(h.purchase_cost_fifo, 0) as cost_basis,
			h.shares * (COALESCE(lp.close_price, h.current_price, h.purchase_cost_average, 0) - COALESCE(h.purchase_cost_fifo, 0)) as unrealized_gain,
			COALESCE(lp.date, h.price_last_date, NOW()) as price_last_date
		FROM portfolio_holdings h
		LEFT JOIN latest_prices lp ON h.ticker = lp.ticker
		WHERE h.portfolio_id = $1
		ORDER BY h.ticker
	`, portfolioID)
	if err != nil {
		return fmt.Errorf("failed to get positions: %v", err)
	}
	defer rows.Close()

	// Initialize values
	report.CashBalance = 0
	report.StocksValue = 0
	report.Holdings = make([]HoldingPerformance, 0)

	for rows.Next() {
		var h HoldingPerformance
		err := rows.Scan(
			&h.Ticker,
			&h.Shares,
			&h.CurrentPrice,
			&h.CurrentValue,
			&h.CostBasis,
			&h.UnrealizedGain,
			&h.LastUpdate,
		)
		if err != nil {
			return fmt.Errorf("failed to scan holding: %v", err)
		}

		fmt.Printf("Found holding: %s, Shares=%f, Price=%f, Value=%f\n",
			h.Ticker, h.Shares, h.CurrentPrice, h.CurrentValue)

		if h.Ticker == "CASH" {
			report.CashBalance = h.Shares // For CASH, shares = amount
			fmt.Printf("Set cash balance: %f\n", report.CashBalance)
		} else {
			report.StocksValue += h.CurrentValue
			report.Holdings = append(report.Holdings, h)
			fmt.Printf("Added stock value: %f (total=%f)\n", h.CurrentValue, report.StocksValue)
		}
	}

	// Set total portfolio value
	report.CurrentValue = report.CashBalance + report.StocksValue
	fmt.Printf("Total portfolio value: Cash=%f + Stocks=%f = %f\n",
		report.CashBalance, report.StocksValue, report.CurrentValue)

	return nil
}

func (s *ReportingService) getPerformanceMetrics(portfolioID int, period string, report *PerformanceReport) error {
	fmt.Println("\nCalculating Performance Metrics:")

	// Get total invested amount
	var totalInvested float64
	err := s.db.QueryRow(`
		SELECT COALESCE(SUM(CASE 
			WHEN type = 'DEPOSIT' THEN amount 
			WHEN type = 'WITHDRAW' THEN -amount
			ELSE 0 
		END), 0)
		FROM portfolio_transactions 
		WHERE portfolio_id = $1
	`, portfolioID).Scan(&totalInvested)
	if err != nil {
		return fmt.Errorf("failed to get total invested: %v", err)
	}
	fmt.Printf("Total Invested: %f\n", totalInvested)

	// Calculate returns
	report.RealizedGains = 0   // Sum of realized gains from sells
	report.UnrealizedGains = 0 // Current value - Cost basis
	report.DividendIncome = 0  // Sum of dividends
	report.TotalReturn = report.RealizedGains + report.UnrealizedGains + report.DividendIncome

	// Calculate return percentage
	if totalInvested > 0 {
		report.ReturnPercent = (report.TotalReturn / totalInvested) * 100
	}

	fmt.Printf("Performance Breakdown:\n")
	fmt.Printf("- Realized Gains: %f\n", report.RealizedGains)
	fmt.Printf("- Unrealized Gains: %f\n", report.UnrealizedGains)
	fmt.Printf("- Dividend Income: %f\n", report.DividendIncome)
	fmt.Printf("- Total Return: %f\n", report.TotalReturn)
	fmt.Printf("- Return Percent: %f%%\n", report.ReturnPercent)

	return nil
}

func (s *ReportingService) getPeriodStartDate(period string) time.Time {
	now := time.Now()
	switch period {
	case "YTD":
		return time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
	case "1Y":
		return now.AddDate(-1, 0, 0)
	case "1M":
		return now.AddDate(0, -1, 0)
	default:
		return time.Time{} // Beginning of time for "ALL"
	}
}

// CalculateReturns calculates IRR and XIRR for a given period
func (s *ReportingService) CalculateReturns(portfolioID int, startDate, endDate time.Time) (irr, xirr float64, err error) {
	// Get all cash flows including current portfolio value
	rows, err := s.db.Query(`
		WITH all_flows AS (
			-- Regular cash flows
			SELECT 
				transaction_at as flow_date,
				CASE 
					WHEN type = 'DEPOSIT' THEN -amount 
					WHEN type = 'WITHDRAW' THEN amount
					WHEN type = 'DIVIDEND' THEN amount
					ELSE 0 
				END as flow_amount
			FROM portfolio_transactions 
			WHERE portfolio_id = $1 
				AND transaction_at BETWEEN $2 AND $3
			
			UNION ALL
			
			-- Add current portfolio value as final flow
			SELECT 
				$3 as flow_date,
				(
					SELECT COALESCE(SUM(CASE 
						WHEN ticker = 'CASH' THEN shares
						ELSE shares * current_price
					END), 0)
					FROM portfolio_holdings
					WHERE portfolio_id = $1
				) as flow_amount
		)
		SELECT flow_date, flow_amount 
		FROM all_flows 
		ORDER BY flow_date
	`, portfolioID, startDate, endDate)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get cash flows: %v", err)
	}
	defer rows.Close()

	var flows []struct {
		date   time.Time
		amount float64
	}

	for rows.Next() {
		var f struct {
			date   time.Time
			amount float64
		}
		if err := rows.Scan(&f.date, &f.amount); err != nil {
			return 0, 0, fmt.Errorf("failed to scan cash flow: %v", err)
		}
		flows = append(flows, f)
	}

	// Calculate IRR using Newton's method
	irr = calculateIRR(flows)
	xirr = calculateXIRR(flows)

	return irr, xirr, nil
}

// Add this function to calculate additional performance metrics
func (s *ReportingService) calculateAdditionalMetrics(portfolioID int, report *PerformanceReport) error {
	// Calculate daily/weekly/monthly/YTD returns
	err := s.db.QueryRow(`
		WITH daily_values AS (
			SELECT 
				date_trunc('day', transaction_at) as date,
				SUM(CASE 
					WHEN type IN ('DEPOSIT', 'BUY') THEN -amount
					WHEN type IN ('WITHDRAW', 'SELL') THEN amount
					WHEN type = 'DIVIDEND' THEN amount
					ELSE 0
				END) as daily_cashflow
			FROM portfolio_transactions
			WHERE portfolio_id = $1
			GROUP BY date_trunc('day', transaction_at)
		),
		period_returns AS (
			SELECT
				(SELECT (current_value - initial_value) / NULLIF(initial_value, 0) * 100
				FROM (
					SELECT SUM(daily_cashflow) as current_value
					FROM daily_values
					WHERE date >= CURRENT_DATE - INTERVAL '1 day'
				) c,
				(
					SELECT SUM(daily_cashflow) as initial_value
					FROM daily_values
					WHERE date >= CURRENT_DATE - INTERVAL '2 days'
					AND date < CURRENT_DATE - INTERVAL '1 day'
				) i) as daily_return,
				
				-- Weekly return calculation
				(SELECT (current_value - initial_value) / NULLIF(initial_value, 0) * 100
				FROM (
					SELECT SUM(daily_cashflow) as current_value
					FROM daily_values
					WHERE date >= CURRENT_DATE - INTERVAL '7 days'
				) c,
				(
					SELECT SUM(daily_cashflow) as initial_value
					FROM daily_values
					WHERE date >= CURRENT_DATE - INTERVAL '14 days'
					AND date < CURRENT_DATE - INTERVAL '7 days'
				) i) as weekly_return,
				
				-- Monthly return calculation
				(SELECT (current_value - initial_value) / NULLIF(initial_value, 0) * 100
				FROM (
					SELECT SUM(daily_cashflow) as current_value
					FROM daily_values
					WHERE date >= CURRENT_DATE - INTERVAL '30 days'
				) c,
				(
					SELECT SUM(daily_cashflow) as initial_value
					FROM daily_values
					WHERE date >= CURRENT_DATE - INTERVAL '60 days'
					AND date < CURRENT_DATE - INTERVAL '30 days'
				) i) as monthly_return,
				
				-- YTD return calculation
				(SELECT (current_value - initial_value) / NULLIF(initial_value, 0) * 100
				FROM (
					SELECT SUM(daily_cashflow) as current_value
					FROM daily_values
					WHERE date >= date_trunc('year', CURRENT_DATE)
				) c,
				(
					SELECT SUM(daily_cashflow) as initial_value
					FROM daily_values
					WHERE date < date_trunc('year', CURRENT_DATE)
				) i) as ytd_return
		)
		SELECT 
			COALESCE(daily_return, 0),
			COALESCE(weekly_return, 0),
			COALESCE(monthly_return, 0),
			COALESCE(ytd_return, 0)
		FROM period_returns
	`, portfolioID).Scan(
		&report.DailyReturn,
		&report.WeeklyReturn,
		&report.MonthlyReturn,
		&report.YTDReturn,
	)
	if err != nil {
		return fmt.Errorf("failed to calculate period returns: %v", err)
	}

	// Calculate risk metrics
	err = s.calculateRiskMetrics(portfolioID, report)
	if err != nil {
		return err
	}

	return nil
}

func (s *ReportingService) calculateRiskMetrics(portfolioID int, report *PerformanceReport) error {
	// Calculate volatility using daily returns
	err := s.db.QueryRow(`
		WITH daily_returns AS (
			SELECT 
				date_trunc('day', transaction_at) as date,
				(SUM(CASE 
					WHEN type IN ('DEPOSIT', 'BUY') THEN -amount
					WHEN type IN ('WITHDRAW', 'SELL') THEN amount
					WHEN type = 'DIVIDEND' THEN amount
					ELSE 0
				END) / NULLIF(LAG(SUM(CASE 
					WHEN type IN ('DEPOSIT', 'BUY') THEN -amount
					WHEN type IN ('WITHDRAW', 'SELL') THEN amount
					WHEN type = 'DIVIDEND' THEN amount
					ELSE 0
				END)) OVER (ORDER BY date_trunc('day', transaction_at)), 0) - 1) * 100 as daily_return
			FROM portfolio_transactions
			WHERE portfolio_id = $1
			GROUP BY date_trunc('day', transaction_at)
		)
		SELECT 
			COALESCE(STDDEV(daily_return) * SQRT(252), 0) as volatility
		FROM daily_returns
	`, portfolioID).Scan(&report.Volatility)
	if err != nil {
		return fmt.Errorf("failed to calculate risk metrics: %v", err)
	}

	// Calculate maximum drawdown
	err = s.calculateDrawdown(portfolioID, report)
	if err != nil {
		return err
	}

	return nil
}

// calculateIRR calculates Internal Rate of Return using Newton's method
func calculateIRR(flows []struct {
	date   time.Time
	amount float64
}) float64 {
	const (
		maxIterations = 100
		tolerance     = 0.000001
		guess         = 0.1 // 10% initial guess
	)

	// Newton's method implementation
	rate := guess
	for i := 0; i < maxIterations; i++ {
		f := 0.0  // NPV
		df := 0.0 // Derivative of NPV

		for _, flow := range flows {
			t := float64(flow.date.Sub(flows[0].date).Hours()) / 24 / 365 // years
			v := math.Pow(1+rate, t)
			f += flow.amount / v
			df += -t * flow.amount / math.Pow(1+rate, t+1)
		}

		// Check if we're close enough
		if math.Abs(f) < tolerance {
			break
		}

		// Update rate using Newton's formula
		rate = rate - f/df
	}

	return rate * 100 // Convert to percentage
}

// calculateXIRR calculates XIRR using Excel's method
func calculateXIRR(flows []struct {
	date   time.Time
	amount float64
}) float64 {
	// Similar to IRR but accounts for irregular intervals
	const (
		maxIterations = 100
		tolerance     = 0.000001
		guess         = 0.1
	)

	rate := guess
	for i := 0; i < maxIterations; i++ {
		f := 0.0
		df := 0.0

		for _, flow := range flows {
			t := float64(flow.date.Sub(flows[0].date).Hours()) / 24 / 365
			v := math.Pow(1+rate, t)
			f += flow.amount / v
			df += -t * flow.amount / math.Pow(1+rate, t+1)
		}

		if math.Abs(f) < tolerance {
			break
		}

		rate = rate - f/df
	}

	return rate * 100
}

// calculateDrawdown calculates maximum drawdown and drawdown periods
func (s *ReportingService) calculateDrawdown(portfolioID int, report *PerformanceReport) error {
	// Get daily portfolio values
	rows, err := s.db.Query(`
		WITH daily_values AS (
			SELECT 
				date_trunc('day', transaction_at) as date,
				SUM(CASE 
					WHEN type IN ('DEPOSIT', 'BUY') THEN -amount
					WHEN type IN ('WITHDRAW', 'SELL') THEN amount
					WHEN type = 'DIVIDEND' THEN amount
					ELSE 0
				END) OVER (ORDER BY transaction_at) as cumulative_value
			FROM portfolio_transactions
			WHERE portfolio_id = $1
			ORDER BY date
		)
		SELECT date, cumulative_value
		FROM daily_values
	`, portfolioID)
	if err != nil {
		return fmt.Errorf("failed to get daily values: %v", err)
	}
	defer rows.Close()

	var (
		maxValue    float64
		currentDD   float64
		maxDD       float64
		ddStart     time.Time
		ddEnd       time.Time
		currentPeak time.Time
	)

	for rows.Next() {
		var (
			date  time.Time
			value float64
		)
		if err := rows.Scan(&date, &value); err != nil {
			return fmt.Errorf("failed to scan daily value: %v", err)
		}

		if value > maxValue {
			maxValue = value
			currentPeak = date
			currentDD = 0
		} else {
			currentDD = (maxValue - value) / maxValue * 100
			if currentDD > maxDD {
				maxDD = currentDD
				ddStart = currentPeak
				ddEnd = date
			}
		}
	}

	report.MaxDrawdown = maxDD
	if maxDD > 0 {
		report.DrawdownPeriods = append(report.DrawdownPeriods, Drawdown{
			StartDate:  ddStart,
			EndDate:    ddEnd,
			Percentage: maxDD,
			Duration:   int(ddEnd.Sub(ddStart).Hours() / 24),
		})
	}

	return nil
}
