-- Create enum for transaction types
CREATE TYPE transaction_type AS ENUM (
    'DEPOSIT',    -- Make sure these match exactly with
    'WITHDRAW',   -- the TransactionType constants in
    'BUY',        -- our Go code
    'SELL',
    'DIVIDEND'
);

-- Create portfolios table
CREATE TABLE portfolios (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create portfolio transactions table
CREATE TABLE portfolio_transactions (
    id SERIAL PRIMARY KEY,
    portfolio_id INTEGER REFERENCES portfolios(id) ON DELETE CASCADE,
    type transaction_type NOT NULL,
    ticker VARCHAR(10) REFERENCES tickers(ticker),
    shares DECIMAL(15,6),
    price DECIMAL(15,6),
    amount DECIMAL(15,2) NOT NULL,
    fee DECIMAL(10,2) NOT NULL DEFAULT 0,
    notes TEXT,
    transaction_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Add constraints
    CONSTRAINT valid_stock_transaction CHECK (
        (type IN ('BUY', 'SELL') AND ticker IS NOT NULL AND shares IS NOT NULL AND price IS NOT NULL)
        OR
        (type IN ('DEPOSIT', 'WITHDRAW') AND ticker IS NULL AND shares IS NULL AND price IS NULL)
        OR
        (type = 'DIVIDEND' AND ticker IS NOT NULL AND amount IS NOT NULL)
    ),
    CONSTRAINT positive_amount CHECK (amount > 0),
    CONSTRAINT non_negative_fee CHECK (fee >= 0)
);

-- Create indexes for faster queries
CREATE INDEX idx_portfolio_transactions_portfolio_date 
ON portfolio_transactions(portfolio_id, transaction_at);

CREATE INDEX idx_portfolio_transactions_ticker 
ON portfolio_transactions(ticker) WHERE ticker IS NOT NULL; 