SELECT * FROM portfolio_transactions WHERE portfolio_id = 2;

-- Check transaction types
SELECT DISTINCT type FROM portfolio_transactions;

-- Check portfolio existence
SELECT * FROM portfolios WHERE id = 2;

-- Check if we have any transactions
SELECT COUNT(*) FROM portfolio_transactions WHERE portfolio_id = 2; 