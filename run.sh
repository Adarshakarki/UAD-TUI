#!/bin/bash

# 1. Build the binary (outputs 'uad-tui')
echo "Building UAD-TUI..."
go build -o uad-tui .

# 2. Check if build was successful
if [ $? -eq 0 ]; then
    # 3. Run the application
    ./uad-tui
fi