#!/bin/bash

echo "🚀 Building GoPiper..."
echo ""

# Step 1: Download dependencies
echo "[1/3] 📦 Downloading Go dependencies..."
go mod tidy
if [ $? -ne 0 ]; then
    echo "❌ Error downloading dependencies"
    exit 1
fi
echo "✅ Dependencies downloaded"
echo ""

# Step 2: Download Piper (if not present)
echo "[2/3] 📥 Downloading Piper TTS (if needed)..."
go generate
if [ $? -ne 0 ]; then
    echo "❌ Error downloading Piper"
    exit 1
fi
echo "✅ Piper ready"
echo ""

# Step 3: Build the project
echo "[3/3] 🔨 Building GoPiper..."
go build -o gopiper .
if [ $? -ne 0 ]; then
    echo "❌ Build failed"
    exit 1
fi
echo "✅ Build complete!"
echo ""

echo "=========================================="
echo "✨ GoPiper built successfully!"
echo "=========================================="
echo ""
echo "Run with: ./gopiper"
echo ""
