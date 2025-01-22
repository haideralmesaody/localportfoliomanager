package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

// Transaction Logic Documentation

/*
Transaction System Overview
--------------------------
The portfolio transaction system handles different types of financial transactions:
1. DEPOSIT: Cash deposits into the portfolio
2. WITHDRAW: Cash withdrawals from the portfolio
3. BUY: Stock purchase transactions
4. SELL: Stock sale transactions
5. DIVIDEND: Dividend payments received

Transaction Flow
---------------
1. Validation Phase:
   - Verify portfolio exists
   - Validate transaction type
   - Check required fields based on type
   - Verify sufficient funds/shares
   - Validate ticker exists (for stock transactions)

2. Processing Phase:
   - Start database transaction
   - Perform balance checks
   - Record the transaction
   - Update portfolio state
   - Commit or rollback

3. Response Phase:
   - Return transaction details
   - Include calculated fields (total amount)

Balance Calculations
------------------
Cash Balance = Sum of:
+ DEPOSIT amounts
- WITHDRAW amounts
- BUY (amount + fee)
+ SELL (amount - fee)
+ DIVIDEND amounts

Stock Position = Sum of:
+ BUY shares
- SELL shares

Transaction Rules
---------------
1. DEPOSIT/WITHDRAW:
   - Required: amount
   - No ticker or shares
   - Fee optional (usually 0)

2. BUY:
   - Required: ticker, shares, price, amount
   - Amount must equal shares × price
   - Must have sufficient cash balance
   - Fee required

3. SELL:
   - Required: ticker, shares, price, amount
   - Amount must equal shares × price
   - Must have sufficient shares
   - Fee required

4. DIVIDEND:
   - Required: ticker, amount
   - Must own shares of the stock
   - No fee
*/

// CreateTransaction handles creating a new transaction with full validation
func (s *Server) CreateTransaction(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	portfolioID, err := strconv.Atoi(vars["id"])
	if err != nil {
		s.respondWithError(w, http.StatusBadRequest, "Invalid portfolio ID")
		return
	}

	// Validate portfolio exists
	if err := s.validatePortfolio(portfolioID); err != nil {
		s.respondWithError(w, http.StatusNotFound, err.Error())
		return
	}

	// Parse request body
	var req CreateTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Validate transaction type
	if !isValidTransactionType(req.Type) {
		s.respondWithError(w, http.StatusBadRequest, "Invalid transaction type")
		return
	}

	// Validate required fields based on transaction type
	if err := s.validateTransactionRequest(req, portfolioID); err != nil {
		s.respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Start database transaction
	tx, err := s.db.Begin()
	if err != nil {
		s.logger.Error("Failed to begin transaction: %v", err)
		s.respondWithError(w, http.StatusInternalServerError, "Failed to process transaction")
		return
	}
	defer tx.Rollback()

	// Get latest balances and average cost
	cashBefore, sharesBefore, err := s.getLatestBalances(portfolioID, req.Ticker, tx)
	if err != nil {
		s.respondWithError(w, http.StatusInternalServerError, "Failed to get latest balances")
		return
	}

	averageCostBefore := 0.0
	if req.Type == Buy || req.Type == Sell {
		averageCostBefore, err = s.calculateAverageCost(portfolioID, req.Ticker, tx)
		if err != nil {
			s.respondWithError(w, http.StatusInternalServerError, "Failed to calculate average cost")
			return
		}
	}

	// Calculate new balances and average cost
	var cashAfter, sharesAfter, averageCostAfter float64
	switch req.Type {
	case Deposit:
		cashAfter = cashBefore + req.Amount
	case Withdraw:
		if cashBefore < req.Amount {
			s.respondWithError(w, http.StatusBadRequest, "Insufficient funds")
			return
		}
		cashAfter = cashBefore - req.Amount
	case Buy:
		totalCost := req.Amount + req.Fee
		if cashBefore < totalCost {
			s.respondWithError(w, http.StatusBadRequest, "Insufficient funds")
			return
		}
		cashAfter = cashBefore - totalCost
		sharesAfter = sharesBefore + req.Shares

		// Calculate new average cost
		totalCostBasis := (averageCostBefore * sharesBefore) + (req.Price * req.Shares)
		averageCostAfter = totalCostBasis / sharesAfter
	case Sell:
		if sharesBefore < req.Shares {
			s.respondWithError(w, http.StatusBadRequest, "Insufficient shares")
			return
		}
		// Calculate FIFO cost basis and realized gains
		avgCostBasis, realizedGain, err := s.calculateFIFOSale(
			portfolioID,
			req.Ticker,
			req.Shares,
			req.Price,
			tx,
		)
		if err != nil {
			s.respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to calculate FIFO sale: %v", err))
			return
		}

		cashAfter = cashBefore + req.Amount - req.Fee
		sharesAfter = sharesBefore - req.Shares
		averageCostAfter = avgCostBasis // Use the FIFO calculated cost basis

		// Store realized gain in transaction notes for reporting
		if req.Notes == "" {
			req.Notes = fmt.Sprintf("Realized gain: %.2f", realizedGain)
		} else {
			req.Notes = fmt.Sprintf("%s; Realized gain: %.2f", req.Notes, realizedGain)
		}
	case Dividend:
		// Validate share ownership
		query := `
            SELECT EXISTS (
                SELECT 1 FROM portfolio_transactions 
                WHERE portfolio_id = $1 AND ticker = $2
                AND type IN ('BUY', 'SELL')
                AND transaction_at <= $3
                GROUP BY ticker
                HAVING SUM(CASE WHEN type = 'BUY' THEN shares ELSE -shares END) > 0
            )
        `
		var hasShares bool
		err = tx.QueryRow(query, portfolioID, req.Ticker, req.TransactionAt).Scan(&hasShares)
		if err != nil {
			s.respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("failed to check shares ownership: %v", err))
			return
		}
		if !hasShares {
			s.respondWithError(w, http.StatusBadRequest, fmt.Sprintf("cannot receive dividend for %s: no shares owned at %v",
				req.Ticker, req.TransactionAt.Format("2006-01-02")))
			return
		}

		// Get current position details
		positionQuery := `
            SELECT shares_count_after
            FROM portfolio_transactions
            WHERE portfolio_id = $1 AND ticker = $2
            AND type IN ('BUY', 'SELL')
            ORDER BY transaction_at DESC, id DESC
            LIMIT 1
        `
		var currentShares float64
		err = tx.QueryRow(positionQuery, portfolioID, req.Ticker).Scan(&currentShares)
		if err != nil {
			s.respondWithError(w, http.StatusInternalServerError, "Failed to get current position")
			return
		}

		// Update cash and maintain position details
		cashAfter = cashBefore + req.Amount
		sharesAfter = currentShares
		averageCostAfter = averageCostBefore // Maintain same cost basis
	}

	// Validate ticker if it's a stock transaction
	if req.Type == Buy || req.Type == Sell || req.Type == Dividend {
		if err := s.validateTicker(req.Ticker); err != nil {
			s.respondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
	}

	// Prepare transaction values based on type
	var ticker sql.NullString
	var shares, price sql.NullFloat64

	if req.Type == Buy || req.Type == Sell || req.Type == Dividend {
		ticker = sql.NullString{String: req.Ticker, Valid: true}
		if req.Shares > 0 {
			shares = sql.NullFloat64{Float64: req.Shares, Valid: true}
		}
		if req.Price > 0 {
			price = sql.NullFloat64{Float64: req.Price, Valid: true}
		}
	}

	// Insert transaction with NULL handling
	var transaction Transaction

	if req.Type == Deposit || req.Type == Withdraw {
		// Don't set ticker, shares, or price - they should be NULL
		query := `
            INSERT INTO portfolio_transactions (
                portfolio_id, type, amount, fee, notes, transaction_at,
                cash_balance_before, cash_balance_after,
                shares_count_before, shares_count_after,
                average_cost_before, average_cost_after
            ) VALUES (
                $1, $2::transaction_type, $3, $4, $5, $6,
                $7, $8, NULL, NULL, NULL, NULL
            )
            RETURNING id, portfolio_id, type, amount, fee, notes, 
                      transaction_at, created_at,
                      cash_balance_before, cash_balance_after`

		err = tx.QueryRow(
			query,
			portfolioID,
			string(req.Type),
			req.Amount,
			req.Fee,
			req.Notes,
			req.TransactionAt,
			cashBefore,
			cashAfter,
		).Scan(
			&transaction.ID,
			&transaction.PortfolioID,
			&transaction.Type,
			&transaction.Amount,
			&transaction.Fee,
			&transaction.Notes,
			&transaction.TransactionAt,
			&transaction.CreatedAt,
			&transaction.CashBalanceBefore,
			&transaction.CashBalanceAfter,
		)
	} else {
		// Handle BUY/SELL/DIVIDEND transactions...
		// Insert transaction with stock details
		query := `
            INSERT INTO portfolio_transactions (
                portfolio_id, type, ticker, shares, price, 
                amount, fee, notes, transaction_at,
                cash_balance_before, cash_balance_after,
                shares_count_before, shares_count_after,
                average_cost_before, average_cost_after
            ) VALUES (
                $1, $2::transaction_type, $3, $4, $5,
                $6, $7, $8, $9,
                $10, $11, $12, $13, $14, $15
            )
            RETURNING id, portfolio_id, type, ticker, shares, price,
                      amount, fee, notes, transaction_at, created_at,
                      cash_balance_before, cash_balance_after,
                      shares_count_before, shares_count_after,
                      average_cost_before, average_cost_after
        `

		// For BUY transactions
		if req.Type == Buy {
			// Validate sufficient cash
			if cashBefore < (req.Amount + req.Fee) {
				s.respondWithError(w, http.StatusBadRequest,
					fmt.Sprintf("insufficient cash balance: have %.2f, need %.2f",
						cashBefore, req.Amount+req.Fee))
				return
			}

			// Update balances
			cashAfter = cashBefore - (req.Amount + req.Fee)
			sharesAfter = sharesBefore + req.Shares

			// Calculate new average cost
			totalCostBefore := sharesBefore * averageCostBefore
			newCost := req.Amount
			averageCostAfter = (totalCostBefore + newCost) / sharesAfter
		}

		// For SELL transactions
		if req.Type == Sell {
			// Validate sufficient shares
			if sharesBefore < req.Shares {
				s.respondWithError(w, http.StatusBadRequest,
					fmt.Sprintf("insufficient shares: have %.6f, want to sell %.6f",
						sharesBefore, req.Shares))
				return
			}

			// Update balances
			cashAfter = cashBefore + req.Amount - req.Fee
			sharesAfter = sharesBefore - req.Shares

			// Calculate realized gain
			realizedGain := (req.Price - averageCostBefore) * req.Shares

			// Add realized gain to notes
			if req.Notes != "" {
				req.Notes += "; "
			}
			req.Notes += fmt.Sprintf("Realized gain: %.2f", realizedGain)

			// Keep same average cost for remaining shares
			averageCostAfter = averageCostBefore
		}

		// For WITHDRAW transactions
		if req.Type == Withdraw {
			// Validate sufficient cash
			if cashBefore < (req.Amount + req.Fee) {
				s.respondWithError(w, http.StatusBadRequest,
					fmt.Sprintf("insufficient cash balance: have %.2f, need %.2f",
						cashBefore, req.Amount+req.Fee))
				return
			}

			// Update cash balance
			cashAfter = cashBefore - (req.Amount + req.Fee)
		}

		err = tx.QueryRow(
			query,
			portfolioID,
			string(req.Type),
			req.Ticker,
			req.Shares,
			req.Price,
			req.Amount,
			req.Fee,
			req.Notes,
			req.TransactionAt,
			cashBefore,
			cashAfter,
			sharesBefore,
			sharesAfter,
			averageCostBefore,
			averageCostAfter,
		).Scan(
			&transaction.ID,
			&transaction.PortfolioID,
			&transaction.Type,
			&ticker,
			&shares,
			&price,
			&transaction.Amount,
			&transaction.Fee,
			&transaction.Notes,
			&transaction.TransactionAt,
			&transaction.CreatedAt,
			&transaction.CashBalanceBefore,
			&transaction.CashBalanceAfter,
			&transaction.SharesCountBefore,
			&transaction.SharesCountAfter,
			&transaction.AverageCostBefore,
			&transaction.AverageCostAfter,
		)
	}

	if err != nil {
		s.logger.Error("Failed to create transaction: %v", err)
		s.logger.Debug("Transaction details: %+v", req)
		s.respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create transaction: %v", err))
		return
	}

	// Convert NULL values to appropriate values for DEPOSIT
	if ticker.Valid {
		transaction.Ticker = ticker.String
	}
	if shares.Valid {
		transaction.Shares = shares.Float64
	}
	if price.Valid {
		transaction.Price = price.Float64
	}

	// For DEPOSIT transactions, set standard values
	if transaction.Type == "DEPOSIT" {
		transaction.Ticker = "CASH"
		transaction.Shares = transaction.Amount
		transaction.Price = 1.0
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		s.logger.Error("Failed to commit transaction: %v", err)
		s.respondWithError(w, http.StatusInternalServerError, "Failed to complete transaction")
		return
	}

	// Return response with total amount
	response := TransactionResponse{
		Transaction: transaction,
		TotalAmount: transaction.Amount + transaction.Fee,
	}

	s.respondWithJSON(w, http.StatusCreated, response)
}

// Helper functions for transaction validation
func isValidTransactionType(t TransactionType) bool {
	switch t {
	case Buy, Sell, Deposit, Withdraw, Dividend:
		return true
	default:
		return false
	}
}

// Transaction validation functions

// validateTransactionRequest performs comprehensive validation of transaction data
func (s *Server) validateTransactionRequest(req CreateTransactionRequest, portfolioID int) error {
	// Common validations
	if req.Amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}
	if req.Fee < 0 {
		return fmt.Errorf("fee cannot be negative")
	}

	// Type-specific validations
	switch req.Type {
	case Buy:
		return validateBuyTransaction(req)
	case Sell:
		return validateSellTransaction(req)
	case Deposit, Withdraw:
		return validateCashTransaction(req)
	case Dividend:
		return s.validateDividendTransaction(req, portfolioID)
	default:
		return fmt.Errorf("invalid transaction type")
	}
}

// validateBuyTransaction validates buy transaction specific rules
func validateBuyTransaction(req CreateTransactionRequest) error {
	if req.Ticker == "" {
		return fmt.Errorf("ticker is required for buy transactions")
	}
	if req.Shares <= 0 {
		return fmt.Errorf("shares must be positive for buy transactions")
	}
	if req.Price <= 0 {
		return fmt.Errorf("price must be positive for buy transactions")
	}
	// Verify amount matches shares × price
	expectedAmount := req.Shares * req.Price
	if math.Abs(expectedAmount-req.Amount) > 0.01 {
		return fmt.Errorf("amount (%.2f) does not match shares × price (%.2f × %.2f = %.2f)",
			req.Amount, req.Shares, req.Price, expectedAmount)
	}
	return nil
}

// validateSellTransaction validates sell transaction specific rules
func validateSellTransaction(req CreateTransactionRequest) error {
	if req.Ticker == "" {
		return fmt.Errorf("ticker is required for sell transactions")
	}
	if req.Shares <= 0 {
		return fmt.Errorf("shares must be positive for sell transactions")
	}
	if req.Price <= 0 {
		return fmt.Errorf("price must be positive for sell transactions")
	}
	// Verify amount matches shares × price
	expectedAmount := req.Shares * req.Price
	if math.Abs(expectedAmount-req.Amount) > 0.01 {
		return fmt.Errorf("amount (%.2f) does not match shares × price (%.2f × %.2f = %.2f)",
			req.Amount, req.Shares, req.Price, expectedAmount)
	}
	return nil
}

// validateCashTransaction validates deposit/withdraw specific rules
func validateCashTransaction(req CreateTransactionRequest) error {
	if req.Amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}
	if req.Fee < 0 {
		return fmt.Errorf("fee cannot be negative")
	}
	if req.Ticker != "" {
		return fmt.Errorf("ticker should not be set for cash transactions")
	}
	if req.Shares != 0 {
		return fmt.Errorf("shares should not be set for cash transactions")
	}
	if req.Price != 0 {
		return fmt.Errorf("price should not be set for cash transactions")
	}
	return nil
}

// validateDividendTransaction validates dividend specific rules
func (s *Server) validateDividendTransaction(req CreateTransactionRequest, portfolioID int) error {
	if req.Ticker == "" {
		return fmt.Errorf("ticker is required for dividend transactions")
	}
	if req.Shares != 0 {
		return fmt.Errorf("shares should not be set for dividend transactions")
	}
	if req.Price != 0 {
		return fmt.Errorf("price should not be set for dividend transactions")
	}
	if req.Fee != 0 {
		return fmt.Errorf("fee should not be set for dividend transactions")
	}

	// Only verify share ownership at dividend date
	query := `
        SELECT EXISTS (
            SELECT 1
            FROM portfolio_transactions 
            WHERE portfolio_id = $1 
            AND ticker = $2
            AND type IN ('BUY', 'SELL')
            AND transaction_at <= $3
            GROUP BY ticker
            HAVING SUM(CASE WHEN type = 'BUY' THEN shares ELSE -shares END) > 0
        )
    `
	var hasShares bool
	err := s.db.QueryRow(query, portfolioID, req.Ticker, req.TransactionAt).Scan(&hasShares)
	if err != nil {
		return fmt.Errorf("failed to check shares ownership: %v", err)
	}
	if !hasShares {
		return fmt.Errorf("cannot receive dividend for %s: no shares owned at %v",
			req.Ticker, req.TransactionAt.Format("2006-01-02"))
	}
	return nil
}

// Transaction Validation Functions

// checkSufficientFunds verifies if the portfolio has enough cash for the transaction
func (s *Server) checkSufficientFunds(portfolioID int, amount float64, tx *sql.Tx) error {
	balance, err := s.getPortfolioBalance(portfolioID, tx)
	if err != nil {
		return fmt.Errorf("failed to check balance: %v", err)
	}
	if balance < amount {
		return fmt.Errorf("insufficient funds: have %.2f, need %.2f", balance, amount)
	}
	return nil
}

// checkSufficientShares verifies if the portfolio has enough shares for a sale
func (s *Server) checkSufficientShares(portfolioID int, ticker string, shares float64, tx *sql.Tx) error {
	available, err := s.getSharesBalance(portfolioID, ticker, tx)
	if err != nil {
		return fmt.Errorf("failed to check shares balance: %v", err)
	}
	if available < shares {
		return fmt.Errorf("insufficient shares: have %.2f, trying to sell %.2f", available, shares)
	}
	return nil
}

// Balance Calculation Functions

// getSharesBalance calculates the current shares for a given stock
func (s *Server) getSharesBalance(portfolioID int, ticker string, tx *sql.Tx) (float64, error) {
	query := `
        WITH running_balance AS (
            SELECT 
                SUM(CASE WHEN type = 'BUY' THEN shares ELSE -shares END) as balance
            FROM portfolio_transactions
            WHERE portfolio_id = $1 AND ticker = $2
            GROUP BY ticker
        )
        SELECT COALESCE(balance, 0) FROM running_balance
    `
	var shares float64
	err := tx.QueryRow(query, portfolioID, ticker).Scan(&shares)
	return shares, err
}

// ListTransactions returns all transactions for a portfolio
func (s *Server) ListTransactions(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	portfolioID, err := strconv.Atoi(vars["id"])
	if err != nil {
		s.logger.Error("Invalid portfolio ID: %v", err)
		s.respondWithError(w, http.StatusBadRequest, "Invalid portfolio ID")
		return
	}

	s.logger.Debug("Listing transactions for portfolio %d", portfolioID)

	query := `
        WITH transaction_details AS (
            SELECT 
                t.*,
                ROW_NUMBER() OVER (
                    PARTITION BY t.portfolio_id 
                    ORDER BY t.transaction_at DESC, t.id DESC
                ) as row_num,
                COUNT(*) OVER (
                    PARTITION BY t.portfolio_id
                ) as total_count
            FROM portfolio_transactions t
            WHERE t.portfolio_id = $1
        )
        SELECT 
            id, portfolio_id, type, ticker, shares, price,
            amount, fee, notes, transaction_at, created_at,
            cash_balance_before, cash_balance_after,
            shares_count_before, shares_count_after,
            average_cost_before, average_cost_after,
            total_count
        FROM transaction_details
        ORDER BY transaction_at DESC, id DESC
    `

	rows, err := s.db.Query(query, portfolioID)
	if err != nil {
		s.logger.Error("Failed to fetch transactions: %v", err)
		s.respondWithError(w, http.StatusInternalServerError, "Failed to fetch transactions")
		return
	}
	defer rows.Close()

	var transactions []Transaction
	var totalCount int
	for rows.Next() {
		var t Transaction
		err := rows.Scan(
			&t.ID,
			&t.PortfolioID,
			&t.Type,
			&t.Ticker,
			&t.Shares,
			&t.Price,
			&t.Amount,
			&t.Fee,
			&t.Notes,
			&t.TransactionAt,
			&t.CreatedAt,
			&t.CashBalanceBefore,
			&t.CashBalanceAfter,
			&t.SharesCountBefore,
			&t.SharesCountAfter,
			&t.AverageCostBefore,
			&t.AverageCostAfter,
			&totalCount,
		)
		if err != nil {
			s.logger.Error("Failed to scan transaction: %v", err)
			continue
		}

		transactions = append(transactions, t)
	}

	s.logger.Debug("Found %d transactions", len(transactions))

	summary, err := s.calculateTransactionSummary(portfolioID)
	if err != nil {
		s.logger.Error("Failed to calculate summary: %v", err)
	} else {
		s.logger.Debug("Transaction summary: deposits=%.2f, withdrawals=%.2f, realized=%.2f, unrealized=%.2f",
			summary.TotalDeposits,
			summary.TotalWithdrawals,
			summary.RealizedGains,
			summary.UnrealizedGains,
		)
	}

	// Convert Transaction to TransactionResponse
	var transactionResponses []TransactionResponse
	for _, t := range transactions {
		transactionResponses = append(transactionResponses, TransactionResponse{
			Transaction: t,
			TotalAmount: t.Amount + t.Fee,
		})
	}

	response := TransactionsListResponse{
		Transactions: transactionResponses, // Use the converted responses
		Total:        totalCount,
		Summary:      summary,
	}

	s.respondWithJSON(w, http.StatusOK, response)
}

// GetTransaction returns a specific transaction
func (s *Server) GetTransaction(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	_, err := strconv.Atoi(vars["id"])
	if err != nil {
		s.respondWithError(w, http.StatusBadRequest, "Invalid portfolio ID")
		return
	}

	_, err = strconv.Atoi(vars["txId"])
	if err != nil {
		s.respondWithError(w, http.StatusBadRequest, "Invalid transaction ID")
		return
	}

	// TODO: Implement transaction retrieval
	s.respondWithJSON(w, http.StatusNotImplemented, map[string]string{
		"message": "Transaction retrieval not implemented yet",
	})
}

// Add this method to Server struct
func (s *Server) validateTicker(ticker string) error {
	query := `SELECT COUNT(*) FROM tickers WHERE ticker = $1`
	var count int
	err := s.db.QueryRow(query, ticker).Scan(&count)
	if err != nil {
		s.logger.Error("Failed to check ticker existence: %v", err)
		return fmt.Errorf("failed to validate ticker")
	}
	if count == 0 {
		return fmt.Errorf("ticker %s not found in our database", ticker)
	}
	return nil
}

func (s *Server) validateDeposit(portfolioID int, tx *sql.Tx) (float64, error) {
	query := `
        SELECT COALESCE(
            SUM(CASE 
                WHEN type = 'DEPOSIT' THEN amount 
                WHEN type = 'WITHDRAW' THEN -amount 
                ELSE 0 
            END),
            0
        )
        FROM portfolio_transactions
        WHERE portfolio_id = $1
    `
	var balance float64
	err := tx.QueryRow(query, portfolioID).Scan(&balance)
	return balance, err
}

// Add new helper function to get latest balances
func (s *Server) getLatestBalances(portfolioID int, ticker string, tx *sql.Tx) (float64, float64, error) {
	// Get latest cash balance
	var cashBefore float64
	cashQuery := `
        SELECT COALESCE(cash_balance_after, 0)
        FROM portfolio_transactions
        WHERE portfolio_id = $1
        ORDER BY transaction_at DESC, id DESC
        LIMIT 1
    `
	err := tx.QueryRow(cashQuery, portfolioID).Scan(&cashBefore)
	if err != nil && err != sql.ErrNoRows {
		return 0, 0, fmt.Errorf("failed to get cash balance: %v", err)
	}

	// Get latest shares count for ticker
	var sharesBefore float64
	if ticker != "" {
		sharesQuery := `
            SELECT COALESCE(shares_count_after, 0)
            FROM portfolio_transactions
            WHERE portfolio_id = $1 AND ticker = $2
            ORDER BY transaction_at DESC, id DESC
            LIMIT 1
        `
		err = tx.QueryRow(sharesQuery, portfolioID, ticker).Scan(&sharesBefore)
		if err != nil && err != sql.ErrNoRows {
			return 0, 0, fmt.Errorf("failed to get shares count: %v", err)
		}
	}

	return cashBefore, sharesBefore, nil
}

// Add new function to calculate realized gains
func (s *Server) calculateRealizedGains(portfolioID int) (map[string]StockGains, error) {
	query := `
        WITH transactions_ordered AS (
            SELECT 
                ticker,
                type,
                shares,
                price,
                transaction_at,
                average_cost_before,
                fee
            FROM portfolio_transactions 
            WHERE portfolio_id = $1 
            AND type IN ('BUY', 'SELL')
            ORDER BY transaction_at, id
        ),
        gains_calc AS (
            SELECT 
                ticker,
                SUM(CASE 
                    WHEN type = 'SELL' 
                    THEN shares * (price - average_cost_before)
                    ELSE 0 
                END) as realized_gains,
                SUM(CASE 
                    WHEN type = 'SELL' 
                    THEN shares 
                    ELSE 0 
                END) as sold_shares,
                SUM(CASE 
                    WHEN type = 'SELL' 
                    THEN fee 
                    ELSE 0 
                END) as sell_fees,
                AVG(CASE 
                    WHEN type = 'BUY' 
                    THEN price 
                    ELSE NULL 
                END) as avg_buy_price,
                AVG(CASE 
                    WHEN type = 'SELL' 
                    THEN price 
                    ELSE NULL 
                END) as avg_sell_price
            FROM transactions_ordered
            GROUP BY ticker
            HAVING SUM(CASE WHEN type = 'SELL' THEN shares ELSE 0 END) > 0
        )
        SELECT 
            ticker,
            realized_gains,
            sold_shares,
            COALESCE(avg_buy_price, 0) as average_buy_price,
            COALESCE(avg_sell_price, 0) as average_sell_price
        FROM gains_calc
    `

	rows, err := s.db.Query(query, portfolioID)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate realized gains: %v", err)
	}
	defer rows.Close()

	gains := make(map[string]StockGains)
	for rows.Next() {
		var (
			ticker string
			gain   StockGains
		)

		err := rows.Scan(
			&ticker,
			&gain.RealizedGains,
			&gain.SoldShares,
			&gain.AverageBuyPrice,
			&gain.AverageSellPrice,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan realized gains: %v", err)
		}

		gains[ticker] = gain
	}

	return gains, nil
}

// Add this function to calculate average cost
func (s *Server) calculateAverageCost(portfolioID int, ticker string, tx *sql.Tx) (float64, error) {
	query := `
        SELECT COALESCE(average_cost_after, 0)
        FROM portfolio_transactions
        WHERE portfolio_id = $1 AND ticker = $2
        ORDER BY transaction_at DESC, id DESC
        LIMIT 1
    `
	var avgCost float64
	err := tx.QueryRow(query, portfolioID, ticker).Scan(&avgCost)
	if err != nil && err != sql.ErrNoRows {
		return 0, fmt.Errorf("failed to calculate average cost: %v", err)
	}
	return avgCost, nil
}

// Update unrealized gains calculation to use average cost
func (s *Server) calculateUnrealizedGains(portfolioID int) (map[string]float64, error) {
	query := `
        WITH current_positions AS (
            SELECT 
                t.ticker,
                t.shares_count_after as shares,
                t.average_cost_after as cost,
                sp.last_price as current_price
            FROM portfolio_transactions t
            JOIN stock_prices sp ON t.ticker = sp.ticker
            WHERE t.portfolio_id = $1
            AND t.shares_count_after > 0
            ORDER BY t.transaction_at DESC, t.id DESC
            LIMIT 1
        )
        SELECT 
            ticker,
            shares * (current_price - cost) as unrealized_gain
        FROM current_positions
    `

	rows, err := s.db.Query(query, portfolioID)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate unrealized gains: %v", err)
	}
	defer rows.Close()

	gains := make(map[string]float64)
	for rows.Next() {
		var (
			ticker string
			gain   float64
		)

		err := rows.Scan(&ticker, &gain)
		if err != nil {
			return nil, fmt.Errorf("failed to scan unrealized gain: %v", err)
		}

		gains[ticker] = gain
	}

	return gains, nil
}

// Add this function to calculate transaction summary
func (s *Server) calculateTransactionSummary(portfolioID int) (TransactionSummary, error) {
	query := `
        WITH latest_prices AS (
            SELECT DISTINCT ON (ticker) 
                ticker,
                close_price as last_price
            FROM daily_stock_prices
            ORDER BY ticker, date DESC
        ),
        positions AS (
            SELECT DISTINCT ON (t.ticker)
                t.ticker,
                t.shares_count_after as shares,
                t.average_cost_after as cost,
                lp.last_price as current_price
            FROM portfolio_transactions t
            JOIN latest_prices lp ON t.ticker = lp.ticker
            WHERE t.portfolio_id = $1 
            AND t.type IN ('BUY', 'SELL')
            AND t.shares_count_after > 0
            ORDER BY t.ticker, t.transaction_at DESC, t.id DESC
        ),
        summary AS (
            SELECT 
                COALESCE(SUM(CASE WHEN type = 'DEPOSIT' THEN amount ELSE 0 END), 0) as total_deposits,
                COALESCE(SUM(CASE WHEN type = 'WITHDRAW' THEN amount ELSE 0 END), 0) as total_withdrawals,
                COALESCE(SUM(fee), 0) as total_fees,
                COALESCE(SUM(CASE WHEN type = 'DIVIDEND' THEN amount ELSE 0 END), 0) as total_dividends,
                COALESCE(SUM(CASE 
                    WHEN type = 'SELL' 
                    THEN shares * (price - average_cost_before)
                    ELSE 0 
                END), 0) as realized_gains
            FROM portfolio_transactions
            WHERE portfolio_id = $1
        )
        SELECT 
            s.*,
            COALESCE(SUM(p.shares * (p.current_price - p.cost)), 0) as unrealized_gains
        FROM summary s
        CROSS JOIN LATERAL (
            SELECT shares, current_price, cost 
            FROM positions
        ) p
        GROUP BY 
            s.total_deposits,
            s.total_withdrawals,
            s.total_fees,
            s.total_dividends,
            s.realized_gains
    `

	var summary TransactionSummary
	err := s.db.QueryRow(query, portfolioID).Scan(
		&summary.TotalDeposits,
		&summary.TotalWithdrawals,
		&summary.TotalFees,
		&summary.TotalDividends,
		&summary.RealizedGains,
		&summary.UnrealizedGains,
	)
	if err != nil {
		return summary, fmt.Errorf("failed to calculate summary: %v", err)
	}

	// Calculate net cash flow
	summary.NetCashFlow = summary.TotalDeposits -
		summary.TotalWithdrawals +
		summary.TotalDividends +
		summary.RealizedGains -
		summary.TotalFees

	return summary, nil
}

// Add logging to calculateRealizedGainsByStock
func (s *Server) calculateRealizedGainsByStock(portfolioID int) (map[string]StockGains, error) {
	s.logger.Debug("Calculating realized gains by stock for portfolio %d", portfolioID)

	query := `
        WITH stock_transactions AS (
            SELECT 
                ticker,
                type,
                shares,
                price,
                average_cost_before,
                transaction_at
            FROM portfolio_transactions
            WHERE portfolio_id = $1 
            AND type IN ('BUY', 'SELL')
            ORDER BY transaction_at, id
        )
        SELECT 
            ticker,
            SUM(CASE 
                WHEN type = 'SELL' 
                THEN shares * (price - average_cost_before)
                ELSE 0 
            END) as realized_gains,
            SUM(CASE 
                WHEN type = 'SELL' 
                THEN shares 
                ELSE 0 
            END) as sold_shares,
            AVG(CASE 
                WHEN type = 'BUY' 
                THEN price 
                ELSE NULL 
            END) as avg_buy_price,
            AVG(CASE 
                WHEN type = 'SELL' 
                THEN price 
                ELSE NULL 
            END) as avg_sell_price
        FROM stock_transactions
        GROUP BY ticker
        HAVING SUM(CASE WHEN type = 'SELL' THEN shares ELSE 0 END) > 0
    `

	rows, err := s.db.Query(query, portfolioID)
	if err != nil {
		s.logger.Error("Failed to execute realized gains query: %v", err)
		return nil, fmt.Errorf("failed to calculate realized gains by stock: %v", err)
	}
	defer rows.Close()

	gains := make(map[string]StockGains)
	for rows.Next() {
		var (
			ticker string
			gain   StockGains
		)

		err := rows.Scan(
			&ticker,
			&gain.RealizedGains,
			&gain.SoldShares,
			&gain.AverageBuyPrice,
			&gain.AverageSellPrice,
		)
		if err != nil {
			s.logger.Error("Failed to scan realized gains row: %v", err)
			return nil, fmt.Errorf("failed to scan realized gains: %v", err)
		}

		s.logger.Debug("Scanned realized gains for %s: gains=%.2f, shares=%.2f, avgBuy=%.2f, avgSell=%.2f",
			ticker,
			gain.RealizedGains,
			gain.SoldShares,
			gain.AverageBuyPrice,
			gain.AverageSellPrice,
		)

		gains[ticker] = gain
	}

	s.logger.Debug("Completed realized gains calculation for %d stocks", len(gains))
	return gains, nil
}

// Add new FIFO tracking structure
type StockLot struct {
	Shares    float64
	CostBasis float64
	BuyDate   time.Time
}

// Add FIFO calculation for sells
func (s *Server) calculateFIFOSale(portfolioID int, ticker string, sharesToSell float64, sellPrice float64, tx *sql.Tx) (float64, float64, error) {
	// Get all buy lots in FIFO order
	query := `
        SELECT shares, price, transaction_at
        FROM portfolio_transactions
        WHERE portfolio_id = $1 
        AND ticker = $2
        AND type = 'BUY'
        AND shares > 0
        ORDER BY transaction_at ASC
    `

	rows, err := tx.Query(query, portfolioID, ticker)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get buy lots: %v", err)
	}
	defer rows.Close()

	var lots []StockLot
	for rows.Next() {
		var lot StockLot
		err := rows.Scan(&lot.Shares, &lot.CostBasis, &lot.BuyDate)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to scan lot: %v", err)
		}
		lots = append(lots, lot)
	}

	// Calculate realized gains using FIFO
	var remainingToSell = sharesToSell
	var totalCostBasis float64
	var realizedGain float64

	for _, lot := range lots {
		if remainingToSell <= 0 {
			break
		}

		sharesToSellFromLot := math.Min(remainingToSell, lot.Shares)
		costBasisForSale := sharesToSellFromLot * lot.CostBasis
		saleProceeds := sharesToSellFromLot * sellPrice

		realizedGain += saleProceeds - costBasisForSale
		totalCostBasis += costBasisForSale
		remainingToSell -= sharesToSellFromLot
	}

	if remainingToSell > 0 {
		return 0, 0, fmt.Errorf("insufficient shares available for sale")
	}

	averageCostBasis := totalCostBasis / sharesToSell
	return averageCostBasis, realizedGain, nil
}
