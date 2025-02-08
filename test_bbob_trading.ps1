# Test script for comprehensive portfolio transaction testing
$portfolioId = 1  # Using portfolio ID 1
$logFile = "bbob_trading_test_$(Get-Date -Format 'yyyyMMdd_HHmmss').log"

function Write-Log {
    param($Message)
    $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    "$timestamp - $Message" | Tee-Object -FilePath $logFile -Append
}

function Write-Section {
    param($Title)
    Write-Log "`n============================================"
    Write-Log $Title
    Write-Log "============================================`n"
}

function Get-Holdings {
    $holdings = Invoke-RestMethod -Uri "http://localhost:8080/api/portfolios/$portfolioId/holdings" -Method Get
    $holdingsJson = $holdings | ConvertTo-Json -Depth 10
    Write-Log "Current Holdings:"
    Write-Log $holdingsJson
    return $holdings
}

function Get-Transactions {
    $transactions = Invoke-RestMethod -Uri "http://localhost:8080/api/portfolios/$portfolioId/transactions" -Method Get
    $transactionsJson = $transactions | ConvertTo-Json -Depth 10
    Write-Log "Transaction History:"
    Write-Log $transactionsJson
    return $transactions
}

# Start Test Sequence
Write-Section "Starting Comprehensive Trading Test Sequence"
Write-Log "Initial State Check"
Get-Holdings
Get-Transactions

# Test Case 1: Initial Deposit
Write-Section "Test Case 1: Initial Deposit"
$depositResult = Invoke-RestMethod -Uri "http://localhost:8080/api/portfolios/$portfolioId/transactions" `
-Method Post `
-ContentType "application/json" `
-Body '{
    "type": "DEPOSIT",
    "amount": 1000000,
    "fee": 0,
    "notes": "Initial cash deposit",
    "transaction_at": "2024-02-06T10:00:00Z"
}'
Write-Log "Initial Deposit Result: $($depositResult | ConvertTo-Json)"
Get-Holdings

# Test Case 2: First Buy - BBOB
Write-Section "Test Case 2: First Buy - BBOB"
$firstBuyResult = Invoke-RestMethod -Uri "http://localhost:8080/api/portfolios/$portfolioId/transactions" `
-Method Post `
-ContentType "application/json" `
-Body '{
    "type": "BUY",
    "ticker": "BBOB",
    "shares": 100000,
    "price": 4.19,
    "amount": 419000,
    "fee": 1000,
    "notes": "Initial BBOB position",
    "transaction_at": "2024-02-06T11:00:00Z"
}'
Write-Log "First Buy Result: $($firstBuyResult | ConvertTo-Json)"
Get-Holdings

# Test Case 3: Second Buy - BBOB at Different Price
Write-Section "Test Case 3: Second Buy - BBOB"
$secondBuyResult = Invoke-RestMethod -Uri "http://localhost:8080/api/portfolios/$portfolioId/transactions" `
-Method Post `
-ContentType "application/json" `
-Body '{
    "type": "BUY",
    "ticker": "BBOB",
    "shares": 50000,
    "price": 4.15,
    "amount": 207500,
    "fee": 500,
    "notes": "Second BBOB buy",
    "transaction_at": "2024-02-06T12:00:00Z"
}'
Write-Log "Second Buy Result: $($secondBuyResult | ConvertTo-Json)"
Get-Holdings

# Test Case 4: Partial Sell - BBOB
Write-Section "Test Case 4: Partial Sell - BBOB"
$partialSellResult = Invoke-RestMethod -Uri "http://localhost:8080/api/portfolios/$portfolioId/transactions" `
-Method Post `
-ContentType "application/json" `
-Body '{
    "type": "SELL",
    "ticker": "BBOB",
    "shares": 75000,
    "price": 4.25,
    "amount": 318750,
    "fee": 750,
    "notes": "Taking partial profits",
    "transaction_at": "2024-02-06T13:00:00Z"
}'
Write-Log "Partial Sell Result: $($partialSellResult | ConvertTo-Json)"
Get-Holdings

# Test Case 5: Withdrawal Test
Write-Section "Test Case 5: Withdrawal Test"
$withdrawResult = Invoke-RestMethod -Uri "http://localhost:8080/api/portfolios/$portfolioId/transactions" `
-Method Post `
-ContentType "application/json" `
-Body '{
    "type": "WITHDRAW",
    "amount": 100000,
    "fee": 0,
    "notes": "Test withdrawal",
    "transaction_at": "2024-02-06T14:00:00Z"
}'
Write-Log "Withdrawal Result: $($withdrawResult | ConvertTo-Json)"
Get-Holdings

# Test Case 6: Final Sell - Complete Position
Write-Section "Test Case 6: Complete Position Sale"
$finalSellResult = Invoke-RestMethod -Uri "http://localhost:8080/api/portfolios/$portfolioId/transactions" `
-Method Post `
-ContentType "application/json" `
-Body '{
    "type": "SELL",
    "ticker": "BBOB",
    "shares": 75000,
    "price": 4.30,
    "amount": 322500,
    "fee": 750,
    "notes": "Closing position",
    "transaction_at": "2024-02-06T15:00:00Z"
}'
Write-Log "Final Sell Result: $($finalSellResult | ConvertTo-Json)"
Get-Holdings

# Final Verification
Write-Section "Final State Verification"
Write-Log "Final Holdings:"
Get-Holdings
Write-Log "Complete Transaction History:"
Get-Transactions

Write-Section "Test Sequence Complete"
Write-Log "Log file saved as: $logFile"

# Verification Queries
Write-Section "FIFO Lot Status"
$lots = Invoke-RestMethod -Uri "http://localhost:8080/api/portfolios/$portfolioId/lots" -Method Get
Write-Log "Current Lots:"
Write-Log ($lots | ConvertTo-Json -Depth 10)

Write-Section "Portfolio Summary"
$summary = Invoke-RestMethod -Uri "http://localhost:8080/api/portfolios/$portfolioId/summary" -Method Get
Write-Log "Portfolio Summary:"
Write-Log ($summary | ConvertTo-Json -Depth 10) 