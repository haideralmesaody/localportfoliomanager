package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

//Logic for handling transactions
// Both the prtfolio_holdings and Portfolio_transactions tables are used to store the transactions
// The portfolio_holdings table is used to store the holdings of the portfolio
// THe structure of the tables as below
/*
-- Table: public.portfolio_holdings

-- DROP TABLE IF EXISTS public.portfolio_holdings;

CREATE TABLE IF NOT EXISTS public.portfolio_holdings
(
    id bigint NOT NULL GENERATED ALWAYS AS IDENTITY ( INCREMENT 1 START 1 MINVALUE 1 MAXVALUE 9223372036854775807 CACHE 1 ),
    portfolio_id bigint NOT NULL,
    ticker character varying(255) COLLATE pg_catalog."default" NOT NULL,
    shares numeric(19,6) NOT NULL DEFAULT 0.000000,
    purchase_cost_average numeric(19,6) NOT NULL DEFAULT 0.000000,
    purchase_cost_fifo numeric(19,6) NOT NULL DEFAULT 0.000000,
    current_price numeric(19,6),
    price_last_date date,
    position_cost_average numeric(19,6),
    position_cost_fifo numeric(19,6),
    unrealized_gain_average numeric(19,6),
    unrealized_gain_fifo numeric(19,6),
    target_percentage numeric(5,2),
    current_percentage numeric(5,2),
    adjustment_percentage numeric(5,2),
    adjustment_value numeric(19,6),
    adjustment_quantity bigint,
    created_at timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT portfolio_holdings_pkey PRIMARY KEY (id),
    CONSTRAINT portfolio_holdings_portfolio_ticker_unique UNIQUE (portfolio_id, ticker),
    CONSTRAINT portfolio_holdings_portfolio_id_fkey FOREIGN KEY (portfolio_id)
        REFERENCES public.portfolios (id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
)

TABLESPACE pg_default;

ALTER TABLE IF EXISTS public.portfolio_holdings
    OWNER to postgres;


-- Table: public.portfolio_transactions

-- DROP TABLE IF EXISTS public.portfolio_transactions;

CREATE TABLE IF NOT EXISTS public.portfolio_transactions
(
    id integer NOT NULL DEFAULT nextval('portfolio_transactions_id_seq'::regclass),
    portfolio_id integer,
    type transaction_type NOT NULL,
    ticker character varying(10) COLLATE pg_catalog."default",
    shares numeric(15,6),
    price numeric(15,6),
    amount numeric(15,2) NOT NULL,
    fee numeric(10,2) NOT NULL DEFAULT 0,
    notes text COLLATE pg_catalog."default",
    transaction_at timestamp with time zone NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    cash_balance_before numeric(15,2),
    cash_balance_after numeric(15,2),
    shares_count_before numeric(15,6),
    shares_count_after numeric(15,6),
    average_cost_before numeric(10,2),
    average_cost_after numeric(10,2),
    CONSTRAINT portfolio_transactions_pkey PRIMARY KEY (id),
    CONSTRAINT portfolio_transactions_portfolio_id_fkey FOREIGN KEY (portfolio_id)
        REFERENCES public.portfolios (id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE NO ACTION,
    CONSTRAINT portfolio_transactions_ticker_fkey FOREIGN KEY (ticker)
        REFERENCES public.tickers (ticker) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE NO ACTION,
    CONSTRAINT valid_stock_transaction CHECK ((type = ANY (ARRAY['BUY'::transaction_type, 'SELL'::transaction_type])) AND ticker IS NOT NULL AND shares IS NOT NULL AND price IS NOT NULL OR (type = ANY (ARRAY['DEPOSIT'::transaction_type, 'WITHDRAW'::transaction_type])) AND ticker IS NULL AND shares IS NULL AND price IS NULL OR type = 'DIVIDEND'::transaction_type AND ticker IS NOT NULL AND amount IS NOT NULL),
    CONSTRAINT positive_amount CHECK (amount > 0::numeric),
    CONSTRAINT non_negative_fee CHECK (fee >= 0::numeric)
)

TABLESPACE pg_default;

ALTER TABLE IF EXISTS public.portfolio_transactions
    OWNER to postgres;
-- Index: idx_portfolio_transactions_portfolio_date

-- DROP INDEX IF EXISTS public.idx_portfolio_transactions_portfolio_date;

CREATE INDEX IF NOT EXISTS idx_portfolio_transactions_portfolio_date
    ON public.portfolio_transactions USING btree
    (portfolio_id ASC NULLS LAST, transaction_at ASC NULLS LAST)
    TABLESPACE pg_default;
-- Index: idx_portfolio_transactions_portfolio_date_cash

-- DROP INDEX IF EXISTS public.idx_portfolio_transactions_portfolio_date_cash;

CREATE INDEX IF NOT EXISTS idx_portfolio_transactions_portfolio_date_cash
    ON public.portfolio_transactions USING btree
    (portfolio_id ASC NULLS LAST, transaction_at ASC NULLS LAST, cash_balance_after ASC NULLS LAST)
    TABLESPACE pg_default;
-- Index: idx_portfolio_transactions_portfolio_ticker_date

-- DROP INDEX IF EXISTS public.idx_portfolio_transactions_portfolio_ticker_date;

CREATE INDEX IF NOT EXISTS idx_portfolio_transactions_portfolio_ticker_date
    ON public.portfolio_transactions USING btree
    (portfolio_id ASC NULLS LAST, ticker COLLATE pg_catalog."default" ASC NULLS LAST, transaction_at ASC NULLS LAST, shares_count_after ASC NULLS LAST)
    TABLESPACE pg_default
    WHERE ticker IS NOT NULL;

*/
//Transaction handlers
func (s *Server) GetTransactions(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	portfolioID, err := strconv.Atoi(vars["id"])
	if err != nil {
		s.respondWithError(w, http.StatusBadRequest, "Invalid portfolio ID")
		return
	}

	query := `
		SELECT 
			id, portfolio_id, type::text, 
			COALESCE(ticker, '') as ticker,  -- Handle NULL ticker
			COALESCE(shares, 0) as shares, 
			COALESCE(price, 0) as price, 
			amount, fee,
			COALESCE(notes, '') as notes,  -- Handle NULL notes
			transaction_at, created_at,
			COALESCE(cash_balance_before, 0) as cash_balance_before,
			COALESCE(cash_balance_after, 0) as cash_balance_after,
			COALESCE(shares_count_before, 0) as shares_count_before,
			COALESCE(shares_count_after, 0) as shares_count_after,
			COALESCE(average_cost_before, 0) as average_cost_before,
			COALESCE(average_cost_after, 0) as average_cost_after
		FROM portfolio_transactions
		WHERE portfolio_id = $1
		ORDER BY transaction_at DESC, id DESC`

	rows, err := s.db.Query(query, portfolioID)
	if err != nil {
		s.logger.Error("Failed to fetch transactions: %v", err)
		s.respondWithError(w, http.StatusInternalServerError, "Failed to fetch transactions")
		return
	}
	defer rows.Close()

	var transactions []Transaction
	for rows.Next() {
		var t Transaction
		var typeStr string
		err := rows.Scan(
			&t.ID, &t.PortfolioID, &typeStr, &t.Ticker,
			&t.Shares, &t.Price, &t.Amount, &t.Fee,
			&t.Notes, &t.TransactionAt, &t.CreatedAt,
			&t.CashBalanceBefore, &t.CashBalanceAfter,
			&t.SharesCountBefore, &t.SharesCountAfter,
			&t.AverageCostBefore, &t.AverageCostAfter,
		)
		if err != nil {
			s.logger.Error("Error scanning transaction: %v", err)
			s.respondWithError(w, http.StatusInternalServerError, "Error scanning transaction")
			return
		}
		t.Type = TransactionType(typeStr)
		transactions = append(transactions, t)
	}

	s.respondWithJSON(w, http.StatusOK, transactions)
}

// CreateDeposit handles deposit transactions
func (s *Server) CreateDeposit(portfolioID int, req TransactionRequest, tx *sql.Tx) error {
	s.logger.Debug("Creating deposit transaction for portfolio %d", portfolioID)

	// Get current cash balance
	cashBefore, err := s.getPortfolioBalance(portfolioID, tx)
	if err != nil {
		return fmt.Errorf("failed to get current balance: %v", err)
	}

	// Calculate new balance
	cashAfter := cashBefore + req.Amount

	// Update cash balance
	result, err := tx.Exec(`
		UPDATE portfolio_holdings 
		SET shares = shares + $3,
			updated_at = CURRENT_TIMESTAMP
		WHERE portfolio_id = $1 AND ticker = $2
	`, portfolioID, "CASH", req.Amount)

	if err != nil {
		return fmt.Errorf("failed to update cash holdings: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error checking update result: %v", err)
	}

	if rowsAffected == 0 {
		// Insert new cash holding if it doesn't exist
		_, err = tx.Exec(`
			INSERT INTO portfolio_holdings (portfolio_id, ticker, shares)
			VALUES ($1, $2, $3)
		`, portfolioID, "CASH", req.Amount)
		if err != nil {
			return fmt.Errorf("failed to create cash holding: %v", err)
		}
	}

	// Record transaction
	_, err = tx.Exec(`
		INSERT INTO portfolio_transactions (
			portfolio_id, type, amount, fee, notes, transaction_at,
			cash_balance_before, cash_balance_after
		) VALUES ($1, 'DEPOSIT', $2, $3, $4, $5, $6, $7)
	`, portfolioID, req.Amount, req.Fee, req.Notes, req.TransactionAt,
		cashBefore, cashAfter)

	if err != nil {
		return fmt.Errorf("failed to record transaction: %v", err)
	}

	return nil
}

// CreateWithdraw handles withdraw transactions
func (s *Server) CreateWithdraw(portfolioID int, req TransactionRequest, tx *sql.Tx) error {
	s.logger.Debug("Creating withdraw transaction for portfolio %d", portfolioID)

	// Get current cash balance
	cashBefore, err := s.getPortfolioBalance(portfolioID, tx)
	if err != nil {
		return fmt.Errorf("failed to get current balance: %v", err)
	}

	// Validate sufficient funds
	if cashBefore < req.Amount {
		return fmt.Errorf("insufficient funds: have %.2f, need %.2f", cashBefore, req.Amount)
	}

	// Calculate new balance
	cashAfter := cashBefore - req.Amount

	// Insert withdraw transaction - Note NULL values for ticker, shares, and price
	query := `
		INSERT INTO portfolio_transactions (
			portfolio_id, type, ticker, shares, price, amount, fee,
			notes, transaction_at,
			cash_balance_before, cash_balance_after,
			shares_count_before, shares_count_after,
			average_cost_before, average_cost_after
		) VALUES (
			$1, 'WITHDRAW', NULL, NULL, NULL, $2, $3,
			$4, $5,
			$6, $7,
			0, 0,
			0, 0
		) RETURNING id`

	var transactionID int
	err = tx.QueryRow(
		query,
		portfolioID,
		req.Amount,
		0, // Fee is 0 for withdrawals
		req.Notes,
		req.TransactionAt,
		cashBefore,
		cashAfter,
	).Scan(&transactionID)

	if err != nil {
		return fmt.Errorf("failed to insert withdraw transaction: %v", err)
	}

	// Update cash holdings
	holdingsQuery := `
		UPDATE portfolio_holdings 
		SET 
			shares = shares - $2,
			updated_at = CURRENT_TIMESTAMP
		WHERE portfolio_id = $1 
		AND ticker = 'CASH'
		AND shares >= $2` // Ensure sufficient balance

	result, err := tx.Exec(holdingsQuery, portfolioID, req.Amount)
	if err != nil {
		return fmt.Errorf("failed to update cash holdings: %v", err)
	}

	// Check if update was successful
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error checking update result: %v", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("insufficient cash balance for withdrawal")
	}

	s.logger.Debug("Successfully created withdraw transaction %d", transactionID)
	return nil
}

// CreateBuy handles buy transactions
func (s *Server) CreateBuy(portfolioID int, req TransactionRequest, tx *sql.Tx) error {
	// Validate ticker
	if err := s.validateTicker(req.Ticker, tx); err != nil {
		return err
	}

	// Initialize holding for this ticker if it doesn't exist
	if err := s.initializeTickerHolding(portfolioID, req.Ticker, tx); err != nil {
		return err
	}

	// Get current cash and share balances
	cashBefore, err := s.getPortfolioBalance(portfolioID, tx)
	if err != nil {
		return fmt.Errorf("failed to get cash balance: %v", err)
	}

	var sharesBefore float64
	err = tx.QueryRow(`
		SELECT COALESCE(shares, 0) 
		FROM portfolio_holdings 
		WHERE portfolio_id = $1 AND ticker = $2
	`, portfolioID, req.Ticker).Scan(&sharesBefore)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to get current shares: %v", err)
	}

	// Calculate totals
	totalCost := (req.Shares * req.Price) + req.Fee
	cashAfter := cashBefore - totalCost

	// Validate sufficient funds
	if cashAfter < 0 {
		return fmt.Errorf("insufficient funds: have %.2f, need %.2f", cashBefore, totalCost)
	}

	sharesAfter := sharesBefore + req.Shares

	// Create FIFO lot
	_, err = tx.Exec(`
		INSERT INTO portfolio_stock_lots (
			portfolio_id, ticker, shares, remaining_shares,
			purchase_price, purchase_date
		) VALUES ($1, $2, $3, $3, $4, $5)
	`, portfolioID, req.Ticker, req.Shares, req.Price, req.TransactionAt)
	if err != nil {
		return fmt.Errorf("failed to create stock lot: %v", err)
	}

	// Update holdings with FIFO cost
	_, err = tx.Exec(`
		INSERT INTO portfolio_holdings (
			portfolio_id, ticker, shares,
			purchase_cost_average, purchase_cost_fifo,
			current_price, price_last_date
		) VALUES (
			$1, $2, $3, $4, $4, $5, $6
		)
		ON CONFLICT (portfolio_id, ticker) DO UPDATE SET
			shares = portfolio_holdings.shares + $3,
			purchase_cost_average = (portfolio_holdings.shares * portfolio_holdings.purchase_cost_average + $3 * $4) 
				/ (portfolio_holdings.shares + $3),
			purchase_cost_fifo = (
				SELECT SUM(shares * purchase_price) / SUM(shares)
				FROM portfolio_stock_lots
				WHERE portfolio_id = $1 AND ticker = $2
			),
			current_price = $5,
			price_last_date = $6,
			updated_at = CURRENT_TIMESTAMP
	`, portfolioID, req.Ticker, req.Shares, req.Price, req.Price, req.TransactionAt)

	if err != nil {
		return fmt.Errorf("failed to update holdings: %v", err)
	}

	// Update cash balance
	_, err = tx.Exec(`
		UPDATE portfolio_holdings 
		SET shares = shares - $2,
			updated_at = CURRENT_TIMESTAMP
		WHERE portfolio_id = $1 AND ticker = 'CASH'
	`, portfolioID, totalCost)

	if err != nil {
		return fmt.Errorf("failed to update cash balance: %v", err)
	}

	// Record transaction
	_, err = tx.Exec(`
		INSERT INTO portfolio_transactions (
			portfolio_id, type, ticker, shares, price, 
			amount, fee, notes, transaction_at,
			cash_balance_before, cash_balance_after,
			shares_count_before, shares_count_after,
			average_cost_before, average_cost_after
		) VALUES ($1, 'BUY', $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`, portfolioID, req.Ticker, req.Shares, req.Price,
		totalCost, req.Fee, req.Notes, req.TransactionAt,
		cashBefore, cashAfter,
		sharesBefore, sharesAfter,
		sharesBefore*req.Price, sharesAfter*req.Price)

	if err != nil {
		return fmt.Errorf("failed to record transaction: %v", err)
	}

	return nil
}

// getFIFOLots retrieves available lots for selling in FIFO order
func (s *Server) getFIFOLots(portfolioID int, ticker string, tx *sql.Tx) ([]StockLot, error) {
	query := `
		SELECT id, portfolio_id, ticker, shares, price, purchase_date, remaining_shares, created_at
		FROM portfolio_stock_lots
		WHERE portfolio_id = $1 
		AND ticker = $2 
		AND remaining_shares > 0
		ORDER BY purchase_date ASC, id ASC`

	rows, err := tx.Query(query, portfolioID, ticker)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch stock lots: %v", err)
	}
	defer rows.Close()

	var lots []StockLot
	for rows.Next() {
		var lot StockLot
		err := rows.Scan(
			&lot.ID, &lot.PortfolioID, &lot.Ticker,
			&lot.Shares, &lot.PurchasePrice, &lot.PurchaseDate,
			&lot.RemainingShares, &lot.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning lot: %v", err)
		}
		lots = append(lots, lot)
	}

	return lots, nil
}

// CreateSell handles sell transactions
func (s *Server) CreateSell(portfolioID int, req TransactionRequest, tx *sql.Tx) error {
	// Get current holding
	holding, err := s.getHolding(portfolioID, req.Ticker, tx)
	if err != nil {
		return fmt.Errorf("failed to get holding: %v", err)
	}

	// Validate sufficient shares
	if holding.Shares < req.Shares {
		return fmt.Errorf("insufficient shares: have %.2f, need %.2f", holding.Shares, req.Shares)
	}

	// Calculate totals
	totalProceeds := (req.Shares * req.Price) - req.Fee
	sharesAfter := holding.Shares - req.Shares

	// Update holdings
	query := `
		UPDATE portfolio_holdings 
		SET 
			shares = $3,
			updated_at = CURRENT_TIMESTAMP,
			current_price = $4,
			price_last_date = $5
		WHERE portfolio_id = $1 AND ticker = $2`

	// Just execute without storing result
	if _, err := tx.Exec(query, portfolioID, req.Ticker, sharesAfter, req.Price, req.TransactionAt); err != nil {
		return fmt.Errorf("failed to update holdings: %v", err)
	}

	// Update cash balance
	_, err = tx.Exec(`
		UPDATE portfolio_holdings 
		SET shares = shares + $2,
			updated_at = CURRENT_TIMESTAMP
		WHERE portfolio_id = $1 AND ticker = 'CASH'
	`, portfolioID, totalProceeds)

	if err != nil {
		return fmt.Errorf("failed to update cash balance: %v", err)
	}

	// Update FIFO lots and calculate realized gain
	realizedGainFIFO, err := s.updateFIFOLots(portfolioID, req.Ticker, req.Shares, req.Price, tx)
	if err != nil {
		return fmt.Errorf("failed to update FIFO lots: %v", err)
	}

	// Calculate average cost realized gain
	realizedGainAvg := req.Shares * (req.Price - holding.PurchaseCostAverage)

	// Record transaction with realized gains
	return s.recordTransaction(tx, portfolioID, req, holding.Shares, sharesAfter,
		realizedGainAvg, realizedGainFIFO)
}

// Update CreateTransaction to handle withdrawals
func (s *Server) CreateTransaction(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	portfolioID, err := strconv.Atoi(vars["id"])
	if err != nil {
		s.respondWithError(w, http.StatusBadRequest, "Invalid portfolio ID")
		return
	}

	// Parse request body
	var req TransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Start transaction
	tx, err := s.db.Begin()
	if err != nil {
		s.respondWithError(w, http.StatusInternalServerError, "Failed to start transaction")
		return
	}
	defer tx.Rollback()

	// Initialize holdings if needed
	err = s.initializePortfolioHoldings(portfolioID, tx)
	if err != nil {
		s.respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to initialize holdings: %v", err))
		return
	}

	// Validate ticker if present
	if req.Ticker != "" {
		if err := s.validateTicker(req.Ticker, tx); err != nil {
			s.respondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
	}

	// Add before processing transaction
	exists, err := s.checkTransactionExists(portfolioID, req, tx)
	if err != nil {
		s.respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to check transaction: %v", err))
		return
	}
	if exists {
		s.respondWithError(w, http.StatusConflict, "Transaction already exists")
		return
	}

	// Process based on transaction type
	switch req.Type {
	case Deposit:
		err = s.CreateDeposit(portfolioID, req, tx)
	case Withdraw:
		err = s.CreateWithdraw(portfolioID, req, tx)
	case Buy:
		err = s.CreateBuy(portfolioID, req, tx)
	case Sell:
		err = s.CreateSell(portfolioID, req, tx)
	default:
		s.respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Invalid transaction type: %s", req.Type))
		return
	}

	if err != nil {
		s.respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := tx.Commit(); err != nil {
		s.respondWithError(w, http.StatusInternalServerError, "Failed to commit transaction")
		return
	}

	s.respondWithJSON(w, http.StatusCreated, map[string]string{
		"message": "Transaction created successfully",
	})

	//Buy Transaction Logic
	//Sell Transaction Logic
	//Dividend Transaction Logic
}

// ListTransactions handles GET requests for transactions
func (s *Server) ListTransactions(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	portfolioID, err := strconv.Atoi(vars["id"])
	if err != nil {
		s.respondWithError(w, http.StatusBadRequest, "Invalid portfolio ID")
		return
	}

	// Get transactions from database
	query := `
		SELECT id, portfolio_id, type, ticker, shares, price, amount, fee,
			   notes, transaction_at, created_at,
			   cash_balance_before, cash_balance_after,
			   shares_count_before, shares_count_after,
			   average_cost_before, average_cost_after
		FROM portfolio_transactions
		WHERE portfolio_id = $1
		ORDER BY transaction_at DESC, id DESC`

	rows, err := s.db.Query(query, portfolioID)
	if err != nil {
		s.respondWithError(w, http.StatusInternalServerError, "Failed to fetch transactions")
		return
	}
	defer rows.Close()

	var transactions []Transaction
	for rows.Next() {
		var t Transaction
		err := rows.Scan(
			&t.ID, &t.PortfolioID, &t.Type, &t.Ticker,
			&t.Shares, &t.Price, &t.Amount, &t.Fee,
			&t.Notes, &t.TransactionAt, &t.CreatedAt,
			&t.CashBalanceBefore, &t.CashBalanceAfter,
			&t.SharesCountBefore, &t.SharesCountAfter,
			&t.AverageCostBefore, &t.AverageCostAfter,
		)
		if err != nil {
			s.respondWithError(w, http.StatusInternalServerError, "Error scanning transaction")
			return
		}
		transactions = append(transactions, t)
	}

	s.respondWithJSON(w, http.StatusOK, transactions)
}

// getPortfolioBalance gets the current cash balance
func (s *Server) getPortfolioBalance(portfolioID int, tx *sql.Tx) (float64, error) {
	var balance float64
	query := `
		SELECT COALESCE(shares, 0) 
		FROM portfolio_holdings 
		WHERE portfolio_id = $1 AND ticker = 'CASH'`

	err := tx.QueryRow(query, portfolioID).Scan(&balance)
	if err == sql.ErrNoRows {
		// If no holdings exist, return 0 balance
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("error getting balance: %v", err)
	}

	return balance, nil
}

// getHolding gets the current holding for a ticker
func (s *Server) getHolding(portfolioID int, ticker string, tx *sql.Tx) (*Holding, error) {
	var holding Holding
	query := `
		SELECT 
			id, portfolio_id, ticker, shares,
			COALESCE(purchase_cost_average, 0) as purchase_cost_average,
			COALESCE(purchase_cost_fifo, 0) as purchase_cost_fifo,
			COALESCE(current_price, 0) as current_price,
			COALESCE(price_last_date, CURRENT_TIMESTAMP) as price_last_date,
			position_cost_average,     -- Allow NULL
			position_cost_fifo,        -- Allow NULL
			unrealized_gain_average,   -- Allow NULL
			unrealized_gain_fifo,      -- Allow NULL
			COALESCE(target_percentage, 0) as target_percentage,
			COALESCE(current_percentage, 0) as current_percentage,
			COALESCE(adjustment_percentage, 0) as adjustment_percentage,
			COALESCE(adjustment_value, 0) as adjustment_value,
			COALESCE(adjustment_quantity, 0) as adjustment_quantity,
			created_at,
			updated_at
		FROM portfolio_holdings 
		WHERE portfolio_id = $1 AND ticker = $2`

	err := tx.QueryRow(query, portfolioID, ticker).Scan(
		&holding.ID,
		&holding.PortfolioID,
		&holding.Ticker,
		&holding.Shares,
		&holding.PurchaseCostAverage,
		&holding.PurchaseCostFIFO,
		&holding.CurrentPrice,
		&holding.PriceLastDate,
		&holding.PositionCostAverage,
		&holding.PositionCostFIFO,
		&holding.UnrealizedGainAverage,
		&holding.UnrealizedGainFIFO,
		&holding.TargetPercentage,
		&holding.CurrentPercentage,
		&holding.AdjustmentPercentage,
		&holding.AdjustmentValue,
		&holding.AdjustmentQuantity,
		&holding.CreatedAt,
		&holding.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("holding not found for ticker %s", ticker)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get holding: %v", err)
	}

	return &holding, nil
}

// Add helper function
func (s *Server) checkTransactionExists(portfolioID int, req TransactionRequest, tx *sql.Tx) (bool, error) {
	var exists bool
	err := tx.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM portfolio_transactions 
			WHERE portfolio_id = $1 
			AND type = $2 
			AND ABS(amount - $3) < 0.01
			AND transaction_at = $4
			AND (
				(ticker = $5 OR (ticker IS NULL AND $5 = '')) 
				AND (ABS(shares - $6) < 0.01 OR shares IS NULL)
				AND (ABS(price - $7) < 0.01 OR price IS NULL)
				AND (ABS(fee - $8) < 0.01)
			)
		)
	`, portfolioID, req.Type, req.Amount, req.TransactionAt,
		req.Ticker, req.Shares, req.Price, req.Fee).Scan(&exists)
	return exists, err
}

// Add recordTransaction method
func (s *Server) recordTransaction(tx *sql.Tx, portfolioID int, req TransactionRequest, sharesBefore, sharesAfter float64, realizedGainAvg, realizedGainFIFO float64) error {
	// Get current cash balance
	cashBefore, err := s.getPortfolioBalance(portfolioID, tx)
	if err != nil {
		return fmt.Errorf("failed to get cash balance: %v", err)
	}

	// Calculate cash after
	var cashAfter float64
	switch req.Type {
	case Buy:
		cashAfter = cashBefore - ((req.Shares * req.Price) + req.Fee)
	case Sell:
		cashAfter = cashBefore + ((req.Shares * req.Price) - req.Fee)
	case Deposit:
		cashAfter = cashBefore + req.Amount
	case Withdraw:
		cashAfter = cashBefore - req.Amount
	}

	// Record transaction with realized gains
	_, err = tx.Exec(`
		INSERT INTO portfolio_transactions (
			portfolio_id, type, ticker, shares, price, 
			amount, fee, notes, transaction_at,
			cash_balance_before, cash_balance_after,
			shares_count_before, shares_count_after,
			average_cost_before, average_cost_after,
			realized_gain_avg, realized_gain_fifo
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
	`,
		portfolioID,
		req.Type,
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
		sharesBefore*req.Price,
		sharesAfter*req.Price,
		realizedGainAvg,
		realizedGainFIFO,
	)

	if err != nil {
		return fmt.Errorf("failed to record transaction: %v", err)
	}

	return nil
}

func (s *Server) updateFIFOLots(portfolioID int, ticker string, sharesToSell float64, sellPrice float64, tx *sql.Tx) (float64, error) {
	var realizedGain float64
	remainingToSell := sharesToSell

	rows, err := tx.Query(`
		SELECT id, remaining_shares, purchase_price 
		FROM portfolio_stock_lots
		WHERE portfolio_id = $1 AND ticker = $2 AND remaining_shares > 0
		ORDER BY purchase_date ASC, id ASC
		FOR UPDATE
	`, portfolioID, ticker)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	for rows.Next() && remainingToSell > 0 {
		var id int
		var remainingShares, purchasePrice float64
		if err := rows.Scan(&id, &remainingShares, &purchasePrice); err != nil {
			return 0, err
		}

		sharesToSellFromLot := math.Min(remainingToSell, remainingShares)
		remainingToSell -= sharesToSellFromLot
		realizedGain += sharesToSellFromLot * (sellPrice - purchasePrice)

		_, err = tx.Exec(`
			UPDATE portfolio_stock_lots 
			SET remaining_shares = remaining_shares - $1
			WHERE id = $2
		`, sharesToSellFromLot, id)
		if err != nil {
			return 0, err
		}
	}

	if remainingToSell > 0 {
		return 0, fmt.Errorf("insufficient shares in FIFO lots")
	}

	return realizedGain, nil
}
