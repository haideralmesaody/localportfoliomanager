# Test script for portfolio performance reporting
param (
    [string]$BaseUrl = "http://localhost:8080"
)

$ErrorActionPreference = "Stop"
$timestamp = Get-Date -Format "yyyyMMdd_HHmmss"
$logFile = "portfolio_reporting_test_${timestamp}.log"

function Write-Log {
    param($Message)
    $logMessage = "$(Get-Date -Format 'yyyy-MM-dd HH:mm:ss') - $Message"
    Write-Host $logMessage
    Add-Content -Path $logFile -Value $logMessage
}

function Test-PerformanceReport {
    param (
        [int]$PortfolioId,
        [string]$Period = "ALL"
    )

    Write-Log "Testing performance report for portfolio $PortfolioId (Period: $Period)"
    
    $response = Invoke-RestMethod -Uri "$BaseUrl/api/portfolios/$PortfolioId/performance?period=$Period" -Method Get
    
    Write-Log "Performance Report:"
    Write-Log ($response | ConvertTo-Json -Depth 10)
    
    # Validate report structure
    $requiredFields = @(
        "current_value",
        "cash_balance",
        "stocks_value"
    )

    foreach ($field in $requiredFields) {
        if ($null -eq $response.$field) {
            throw "Missing required field: $field"
        }
    }

    # Validate calculations
    $calculatedTotal = $response.cash_balance + $response.stocks_value
    if ([Math]::Abs($calculatedTotal - $response.current_value) -gt 0.01) {
        throw "Total value mismatch: $calculatedTotal != $($response.current_value)"
    }

    $calculatedReturn = $response.realized_gains + $response.unrealized_gains + $response.dividend_income
    if ([Math]::Abs($calculatedReturn - $response.total_return) -gt 0.01) {
        throw "Total return mismatch: $calculatedReturn != $($response.total_return)"
    }

    Write-Log "Validation passed for period: $Period"
    Write-Log "Current Value: $($response.current_value)"
    Write-Log "Total Return: $($response.total_return) ($($response.return_percent)%)"
    
    return $response
}

try {
    Write-Log "Starting Portfolio Reporting Test Sequence"
    Write-Log "============================================"
    
    # Test different periods
    $periods = @("ALL", "YTD", "1Y", "1M")
    foreach ($currentPeriod in $periods) {
        Write-Log ""
        Write-Log "Testing period: $currentPeriod"
        Write-Log "----------------------------------------"
        $report = Test-PerformanceReport -PortfolioId 1 -Period $currentPeriod
        
        # Log key metrics
        Write-Log "Key Metrics for period $currentPeriod"
        Write-Log "- Current Value: $($report.current_value)"
        Write-Log "- Cash Balance: $($report.cash_balance)"
        Write-Log "- Stocks Value: $($report.stocks_value)"
        Write-Log "- Total Return: $($report.total_return)"
        Write-Log "- Return %: $($report.return_percent)%"
        Write-Log "- IRR: $($report.irr)%"
        Write-Log "- Volatility: $($report.volatility)%"
        Write-Log "- Sharpe Ratio: $($report.sharpe_ratio)"
        Write-Log "----------------------------------------"
    }
    
    Write-Log "All tests completed successfully"
    
} catch {
    Write-Log "Error: $_"
    throw
} finally {
    Write-Log "Test sequence complete. Log file: $logFile"
} 