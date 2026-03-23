@echo off
echo Starting server at http://localhost:8080
echo Press Ctrl+C to stop
cd /d "%~dp0"
python -m http.server 8080
