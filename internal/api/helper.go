package api

import (
	"database/sql"
	"fmt"
)

// Helper function to get portfolio cash balance
func (s *Server) getPortfolioBalance(portfolioID int, tx *sql.Tx) (float64, error) {
	query := `
        SELECT COALESCE(cash_balance_after, 0)
        FROM portfolio_transactions
        WHERE portfolio_id = $1
        ORDER BY transaction_at DESC, id DESC
        LIMIT 1
    `

	var balance float64
	err := tx.QueryRow(query, portfolioID).Scan(&balance)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("failed to get cash balance: %v", err)
	}

	return balance, nil
}

// Helper function to validate portfolio exists
func (s *Server) validatePortfolio(portfolioID int) error {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM portfolios WHERE id = $1)`

	err := s.db.QueryRow(query, portfolioID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check portfolio: %v", err)
	}
	if !exists {
		return fmt.Errorf("portfolio not found")
	}

	return nil
}
