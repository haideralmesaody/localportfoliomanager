# API Endpoints Documentation
# Revision 1.0
# Date: 2025-01-22
## 1. Stock Data

*   **Base Path:** /api/stocks

*   **Available Endpoints:**

    *   **GET /api/stocks**
        *   Lists all stocks with their latest prices.
        *   **Query Parameters:**
            *   `limit` (optional): Maximum number of stocks to return (default: 50).
            *   `offset` (optional): Number of stocks to skip (default: 0).
        *   **Example Request:**
            ```http
            GET /api/stocks?limit=10&offset=20
            ```
        *   **Example Response:**
            ```json
            {
              "stocks": [
                {
                  "ticker": "AAPL",
                  "last_price": 170.34,
                  "change": 2.50,
                  "change_percentage": 1.49
                },
                {
                  "ticker": "MSFT",
                  "last_price": 285.00,
                  "change": -1.20,
                  "change_percentage": -0.42
                }
              ],
              "total": 500
            }
            ```
    *   **GET /api/stocks/{ticker}**
        *   Retrieves detailed information for a specific stock.
        *   **Parameters:**
            *   `ticker`: Stock symbol (e.g., AAPL, MSFT).
        *   **Example Request:**
            ```http
            GET /api/stocks/AAPL
            ```
        *   **Example Response:**
            ```json
            {
              "ticker": "AAPL",
              "last_price": 170.34,
              "open": 168.84,
              "high": 171.00,
              "low": 168.50,
              "volume": 1000000,
              "change": 2.50,
              "change_percentage": 1.49,
              "last_updated": "2024-02-20T15:30:00Z"
            }
            ```
    *   **GET /api/stocks/{ticker}/prices**
        *   Fetches historical price data for a specific stock.
        *   **Parameters:**
            *   `ticker`: Stock symbol.
        *   **Query Parameters:**
            *   `from` (optional): Start date in YYYY-MM-DD format.
            *   `to` (optional): End date in YYYY-MM-DD format.
            *   `interval` (optional): Data interval: "daily", "weekly", or "monthly" (default: "daily").
        *   **Example Request:**
            ```http
            GET /api/stocks/AAPL/prices?from=2023-01-01&to=2024-01-01&interval=monthly
            ```
        *   **Example Response:**
            ```json
            {
              "ticker": "AAPL",
              "interval": "monthly",
              "prices": [
                {
                  "date": "2023-01-01",
                  "open": 160.00,
                  "high": 165.00,
                  "low": 155.00,
                  "close": 162.50,
                  "volume": 1000000
                },
                {
                  "date": "2023-02-01",
                  "open": 162.50,
                  "high": 168.00,
                  "low": 160.00,
                  "close": 165.00,
                  "volume": 1200000
                }
              ]
            }
            ```

## 2. Portfolio Management

*   **Base Path:** /api/portfolios

*   **Available Endpoints:**

    *   **POST /api/portfolios**
        *   Creates a new portfolio.
        *   **Request Body:**
            ```json
            {
              "name": "My Portfolio",
              "description": "This is my personal investment portfolio."
            }
            ```
        *   **Example Response:**
            ```json
            {
              "id": 123,
              "name": "My Portfolio",
              "description": "This is my personal investment portfolio.",
              "created_at": "2024-02-20T16:00:00Z",
              "updated_at": "2024-02-20T16:00:00Z"
            }
            ```
    *   **GET /api/portfolios**
        *   Lists all portfolios.
        *   **Example Response:**
            ```json
            [
              {
                "id": 123,
                "name": "My Portfolio",
                "description": "This is my personal investment portfolio.",
                "created_at": "2024-02-20T16:00:00Z",
                "updated_at": "2024-02-20T16:00:00Z"
              }
            ]
            ```
    *   **GET /api/portfolios/{id}**
        *   Retrieves details of a specific portfolio.
        *   **Parameters:**
            *   `id`: Portfolio ID.
        *   **Example Request:**
            ```http
            GET /api/portfolios/123
            ```
        *   **Example Response:**
            ```json
            {
              "id": 123,
              "name": "My Portfolio",
              "description": "This is my personal investment portfolio.",
              "created_at": "2024-02-20T16:00:00Z",
              "updated_at": "2024-02-20T16:00:00Z"
            }
            ```
    *   **PUT /api/portfolios/{id}**
        *   Updates information of an existing portfolio.
        *   **Parameters:**
            *   `id`: Portfolio ID.
        *   **Request Body:**
            ```json
            {
              "name": "My Updated Portfolio",
              "description": "This is my updated investment portfolio."
            }
            ```
        *   **Example Response:**
            ```json
            {
              "id": 123,
              "name": "My Updated Portfolio",
              "description": "This is my updated investment portfolio.",
              "created_at": "2024-02-20T16:00:00Z",
              "updated_at": "2024-02-21T10:00:00Z"
            }
            ```
    *   **DELETE /api/portfolios/{id}**
        *   Deletes a portfolio.
        *   **Parameters:**
            *   `id`: Portfolio ID.
        *   **Example Request:**
            ```http
            DELETE /api/portfolios/123
            ```
        *   **Example Response:**
            ```json
            {
              "message": "Portfolio deleted successfully"
            }
            ```
    *   **GET /api/portfolios/{id}/performance**
        *   Calculates and returns performance metrics for a specific portfolio.
        *   **Parameters:**
            *   `id`: Portfolio ID.
        *   **Example Request:**
            ```http
            GET /api/portfolios/123/performance
            ```
        *   **Example Response:**
            ```json
            {
              "start_date": "2023-01-01T00:00:00Z",
              "end_date": "2024-02-21T00:00:00Z",
              "start_value": 10000.00,
              "end_value": 12500.00,
              "net_contributions": 5000.00,
              "realized_gains": 1000.00,
              "unrealized_gains": 1500.00,
              "dividend_income": 200.00,
              "time_weighted_return": 0.15,
              "money_weighted_return": 0.20
            }
            ```

## 3. Transaction Management

*   **Base Path:** /api/portfolios/{id}/transactions

*   **Available Endpoints:**

    *   **POST /api/portfolios/{id}/transactions**
        *   Records a new transaction for a portfolio.
        *   **Parameters:**
            *   `id`: Portfolio ID.
        *   **Request Body:**
            ```json
            {
              "type": "BUY",
              "ticker": "AAPL",
              "shares": 100,
              "price": 170.00,
              "amount": 17000.00,
              "fee": 10.00,
              "notes": "Purchased 100 shares of AAPL.",
              "transaction_at": "2024-02-21T10:00:00Z"
            }
            ```
        *   **Example Response:**
            ```json
            {
              "transaction": {
                "id": 456,
                "portfolio_id": 123,
                "type": "BUY",
                "ticker": "AAPL",
                "shares": 100,
                "price": 170.00,
                "amount": 17000.00,
                "fee": 10.00,
                "notes": "Purchased 100 shares of AAPL.",
                "transaction_at": "2024-02-21T10:00:00Z",
                "created_at": "2024-02-21T10:00:00Z",
                "cash_balance_before": 20000.00,
                "cash_balance_after": 3000.00,
                "shares_count_before": 0,
                "shares_count_after": 100,
                "average_cost_before": 0.00,
                "average_cost_after": 170.00
              },
              "total_amount": 17010.00
            }
            ```
    *   **GET /api/portfolios/{id}/transactions**
        *   Lists all transactions for a portfolio.
        *   **Parameters:**
            *   `id`: Portfolio ID.
        *   **Example Request:**
            ```http
            GET /api/portfolios/123/transactions
            ```
        *   **Example Response:**
            ```json
            {
              "transactions": [
                {
                  "transaction": {
                    "id": 456,
                    "portfolio_id": 123,
                    // ... other transaction details
                  },
                  "total_amount": 17010.00
                }
              ],
              "total": 5,
              "summary": {
                "total_deposits": 25000.00,
                "total_withdrawals": 5000.00,
                "total_fees": 50.00,
                "total_dividends": 200.00,
                "realized_gains": 1000.00,
                "unrealized_gains": 1500.00,
                "net_cash_flow": 21000.00
              }
            }
            ```
    *   **GET /api/portfolios/{id}/transactions/{txId}**
        *   Retrieves details of a specific transaction.
        *   **Parameters:**
            *   `id`: Portfolio ID.
            *   `txId`: Transaction ID.
        *   **Example Request:**
            ```http
            GET /api/portfolios/123/transactions/456
            ```
        *   **Example Response:**
            ```json
            {
              "transaction": {
                "id": 456,
                "portfolio_id": 123,
                // ... other transaction details
              },
              "total_amount": 17010.00
            }
            ```

## Additional Notes:

*   All endpoints return standard HTTP status codes to indicate success or failure.
*   Dates and times are formatted according to ISO 8601 (e.g., `2024-02-21T10:00:00Z`).
*   The `Content-Type` for both requests and responses is `application/json`.
*   Proper error handling is implemented to provide informative error messages in case of invalid requests or internal server errors.