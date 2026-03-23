#!/usr/bin/env bash
# Start the local HTTP server (Mac / Linux)
echo "Starting server at http://localhost:8080"
echo "Press Ctrl+C to stop"
cd "$(dirname "$0")"
python3 -m http.server 8080
