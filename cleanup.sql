-- First, let's identify the correct sequence of transactions
WITH valid_transactions AS (
    SELECT id, portfolio_id, type, ticker, shares, price, amount, fee, 
           transaction_at, created_at,
           SUM(CASE 
               WHEN type = 'BUY' THEN shares 
               WHEN type = 'SELL' THEN -shares 
               ELSE 0 
           END) OVER (ORDER BY transaction_at) as running_shares
    FROM portfolio_transactions
    WHERE portfolio_id = 2
    ORDER BY transaction_at
)
-- Delete invalid transactions (where running_shares goes negative)
DELETE FROM portfolio_transactions 
WHERE id IN (
    SELECT id 
    FROM valid_transactions 
    WHERE running_shares < 0
); 