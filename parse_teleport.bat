@echo off
echo Building parser...
cd /d "%~dp0parser"
go build -o ../parse.exe .
if errorlevel 1 (
    echo Build failed!
    pause
    exit /b 1
)
cd /d "%~dp0"

echo.
echo Parsing Teleport (this may take 1-3 minutes)...
parse.exe --repo "../teleport" --output callgraph.json
echo.
echo Done! Now run serve.bat and open http://localhost:8080
pause
