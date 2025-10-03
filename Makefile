.PHONY: all build clean run dev windows linux install-piper linux-arm64 linux-arm help

# Default target
all: install-piper build

# Install Piper if not present
install-piper:
	@if [ ! -d "piper" ]; then \
		echo "Installing Piper TTS..."; \
		go run install_piper.go; \
		echo "✓ Piper installed successfully"; \
	fi

# Build for current platform
build: install-piper
	@echo "Building for current platform..."
	go build .
	@echo "✓ Build complete"

# Run the server
run:
	@echo "Starting TTS server..."
	go run .

# Build for Windows
windows: install-piper
	@echo "Building for Windows (amd64)..."
	GOOS=windows GOARCH=amd64 go build -o gopiper-windows-amd64.exe .
	@echo "✓ Windows build complete: gopiper-windows-amd64.exe"

# Build for Linux
linux: install-piper
	@echo "Building for Linux (amd64)..."
	GOOS=linux GOARCH=amd64 go build -o gopiper-linux-amd64 .
	@echo "✓ Linux build complete: gopiper-linux-amd64"

# Build for Linux ARM64
linux-arm64: install-piper
	@echo "Building for Linux ARM64..."
	GOOS=linux GOARCH=arm64 go build -o gopiper-linux-arm64 .
	@echo "✓ Linux ARM64 build complete: gopiper-linux-arm64"

# Build for Linux ARM (32-bit)
linux-arm: install-piper
	@echo "Building for Linux ARM (32-bit)..."
	GOOS=linux GOARCH=arm go build -o gopiper-linux-arm .
	@echo "✓ Linux ARM build complete: gopiper-linux-arm"

# Build for all platforms
build-all: install-piper windows linux linux-arm64 linux-arm
	@echo ""
	@echo "========================================"
	@echo "All builds completed successfully!"
	@echo "========================================"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f gopiper gopiper.exe
	rm -f gopiper-windows-amd64.exe
	rm -f gopiper-linux-amd64
	rm -f gopiper-linux-arm64
	rm -f gopiper-linux-arm
	@echo "✓ Clean complete"

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	go mod download
	@echo "✓ Dependencies downloaded"

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...
	@echo "✓ Code formatted"

# Vet code
vet:
	@echo "Vetting code..."
	go vet ./...
	@echo "✓ Code vetted"

# Help
help:
	@echo "Available targets:"
	@echo "  make build         - Build for current platform"
	@echo "  make run           - Run the server"
	@echo "  make install-piper - Download and install Piper TTS"
	@echo "  make windows       - Build for Windows amd64"
	@echo "  make linux         - Build for Linux x86_64"
	@echo "  make linux-arm64   - Build for Linux ARM64 (aarch64)"
	@echo "  make linux-arm     - Build for Linux ARM 32-bit (armv7l)"
	@echo "  make build-all     - Build for all platforms"
	@echo "  make clean         - Remove build artifacts"
	@echo "  make test          - Run tests"
	@echo "  make deps          - Download dependencies"
	@echo "  make fmt           - Format code"
	@echo "  make vet           - Vet code"
	@echo "  make help          - Show this help"
