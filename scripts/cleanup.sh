#!/bin/bash

# Remove log files
rm -f logs/*.log

# Remove debug files
rm -f logs/debug.*

# Remove frontend cache
rm -rf frontend/.dart_tool/
rm -f frontend/.flutter-plugins
rm -f frontend/.flutter-plugins-dependencies
rm -f frontend/.metadata
rm -f frontend/.packages

# Remove build artifacts
rm -rf bin/
rm -rf pkg/

echo "Cleanup complete!" 