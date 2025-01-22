ALTER TABLE portfolio_transactions
DROP COLUMN cash_balance_before,
DROP COLUMN cash_balance_after,
DROP COLUMN shares_count_before,
DROP COLUMN shares_count_after;

DROP INDEX IF EXISTS idx_portfolio_transactions_portfolio_date_cash;
DROP INDEX IF EXISTS idx_portfolio_transactions_portfolio_ticker_date; 