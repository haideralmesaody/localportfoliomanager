# Remove log files
Remove-Item logs\*.log -ErrorAction SilentlyContinue

# Remove debug files
Remove-Item logs\debug.* -ErrorAction SilentlyContinue

# Remove frontend cache
Remove-Item -Recurse -Force frontend\.dart_tool -ErrorAction SilentlyContinue
Remove-Item frontend\.flutter-plugins, frontend\.flutter-plugins-dependencies, frontend\.metadata, frontend\.packages -ErrorAction SilentlyContinue

# Remove build artifacts
Remove-Item -Recurse -Force bin, pkg -ErrorAction SilentlyContinue

Write-Host "Cleanup complete!" -ForegroundColor Green