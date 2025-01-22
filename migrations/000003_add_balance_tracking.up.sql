-- Add balance tracking columns to portfolio_transactions
ALTER TABLE portfolio_transactions
ADD COLUMN cash_balance_before DECIMAL(10,2) NOT NULL,
ADD COLUMN cash_balance_after DECIMAL(10,2) NOT NULL,
ADD COLUMN shares_count_before DECIMAL(10,2),
ADD COLUMN shares_count_after DECIMAL(10,2),
ADD COLUMN average_cost_before DECIMAL(10,2),
ADD COLUMN average_cost_after DECIMAL(10,2);

-- Add index for faster balance lookups
CREATE INDEX idx_portfolio_transactions_balances 
ON portfolio_transactions(portfolio_id, transaction_at, id);

-- Add index for faster shares lookups
CREATE INDEX idx_portfolio_transactions_shares 
ON portfolio_transactions(portfolio_id, ticker, transaction_at, id); 