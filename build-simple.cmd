@echo off
echo 🚀 Building GoPiper...
echo.

REM Step 1: Download dependencies
echo [1/3] 📦 Downloading Go dependencies...
go mod tidy
if %errorlevel% neq 0 (
    echo ❌ Error downloading dependencies
    exit /b %errorlevel%
)
echo ✅ Dependencies downloaded
echo.

REM Step 2: Download Piper (if not present)
echo [2/3] 📥 Downloading Piper TTS (if needed)...
go generate
if %errorlevel% neq 0 (
    echo ❌ Error downloading Piper
    exit /b %errorlevel%
)
echo ✅ Piper ready
echo.

REM Step 3: Build the project
echo [3/3] 🔨 Building GoPiper...
go build .
if %errorlevel% neq 0 (
    echo ❌ Build failed
    exit /b %errorlevel%
)
echo ✅ Build complete!
echo.

echo ==========================================
echo ✨ GoPiper built successfully!
echo ==========================================
echo.
echo Run with: gopiper.exe
echo.
pause
