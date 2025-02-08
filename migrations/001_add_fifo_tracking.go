package migrations

import (
	"database/sql"
	"fmt"
)

func AddFIFOTracking(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Drop the foreign key constraint for ticker
	_, err = tx.Exec(`
		ALTER TABLE portfolio_holdings 
		DROP CONSTRAINT IF EXISTS portfolio_holdings_ticker_fkey
	`)
	if err != nil {
		return fmt.Errorf("failed to drop foreign key constraint: %v", err)
	}

	// Add a check constraint to ensure ticker is valid when it's not CASH
	_, err = tx.Exec(`
		ALTER TABLE portfolio_holdings 
		ADD CONSTRAINT valid_ticker_or_cash 
		CHECK (
			ticker = 'CASH' OR 
			EXISTS (SELECT 1 FROM tickers t WHERE t.ticker = portfolio_holdings.ticker)
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to add check constraint: %v", err)
	}

	return tx.Commit()
}
