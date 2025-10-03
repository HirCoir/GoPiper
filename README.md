# 🎤 GoPiper

<div align="center">

![Go](https://img.shields.io/badge/Go-00ADD8?style=for-the-badge&logo=go&logoColor=white)
![Piper](https://img.shields.io/badge/Piper-TTS-blue?style=for-the-badge)
![License](https://img.shields.io/badge/license-MIT-green?style=for-the-badge)
![Platform](https://img.shields.io/badge/platform-Windows%20%7C%20Linux%20%7C%20ARM-lightgrey?style=for-the-badge)

### A high-performance Text-to-Speech server written in Go using Piper TTS

*Automatic installation • Multi-architecture • REST API • Web Interface*

[Features](#-features) • [Quick Start](#-quick-start) • [Installation](#-installation) • [API](#-api) • [Models](#-voice-models)

</div>

---

## 🎯 Features

- ✅ **Automatic Piper Installation** - Downloads Piper TTS automatically on first build
- ✅ **Multi-Architecture Support** - Windows (amd64), Linux (x86_64, ARM64, ARM)
- ✅ **Parallel Audio Processing** - Configurable concurrent sentence processing
- ✅ **REST API** - Simple HTTP API for text-to-speech conversion
- ✅ **Embedded Web Interface** - Built-in web UI for easy testing
- ✅ **Native Audio Processing** - No FFmpeg required
- ✅ **Multiple Voice Models** - Support for any Piper ONNX model
- ✅ **Advanced Text Processing** - Smart sentence splitting and normalization
- ✅ **Queue Management** - Intelligent task queuing system
- ✅ **Cross-Platform** - Single binary deployment

## 🚀 Quick Start

### Prerequisites

- Go 1.21 or higher
- Internet connection (for first build only)

### 1. Clone the Repository

```bash
git clone https://github.com/HirCoir/gopiper.git
cd gopiper
```

### 2. Build

GoPiper automatically downloads Piper TTS during the build process:

```bash
# Option 1: Simple Build Scripts (Recommended - Does everything!)
./build-simple.sh    # Linux/Mac
build-simple.cmd     # Windows

# Option 2: Using Make
make build

# Option 3: Manual Steps
go mod tidy    # Download Go dependencies
go generate    # Download Piper TTS
go build       # Build the project
```

**What happens during build:**
1. **Downloads Go dependencies** (`go mod tidy`)
2. **Downloads Piper TTS** (~25-50 MB, first time only)
3. **Compiles the project** (creates `gopiper` binary)

**Important:**
- **First build**: Takes longer due to Piper download
- **Subsequent builds**: Instant, Piper already cached
- The `build-simple` scripts do all 3 steps automatically

### 3. Add Voice Models

Download voice models from [Piper Voices](https://huggingface.co/rhasspy/piper-voices):

```bash
# Create models directory
mkdir models
cd models

# Example: Download English voice
wget https://huggingface.co/rhasspy/piper-voices/resolve/main/en/en_US/lessac/medium/en_US-lessac-medium.onnx
wget https://huggingface.co/rhasspy/piper-voices/resolve/main/en/en_US/lessac/medium/en_US-lessac-medium.onnx.json

# Or Spanish voice
wget https://huggingface.co/rhasspy/piper-voices/resolve/main/es/es_MX/cortana/medium/es_MX-cortana-medium.onnx
wget https://huggingface.co/rhasspy/piper-voices/resolve/main/es/es_MX/cortana/medium/es_MX-cortana-medium.onnx.json
```

### 4. Run

```bash
./gopiper
```

🎉 **That's it!** Open your browser at `http://localhost:3000`

## 📦 Installation

### Option 1: Build from Source

```bash
# Clone and build
git clone https://github.com/HirCoir/gopiper.git
cd gopiper
go build

# Run
./gopiper
```

### Option 2: Using Make

```bash
# Build for current platform
make build

# Build for all platforms
make build-all

# Run without building
make run
```

### Option 3: Cross-Compilation

```bash
# Windows (from any OS)
GOOS=windows GOARCH=amd64 go build -o gopiper.exe

# Linux x86_64
GOOS=linux GOARCH=amd64 go build -o gopiper-linux-amd64

# Linux ARM64 (Raspberry Pi 4, etc.)
GOOS=linux GOARCH=arm64 go build -o gopiper-linux-arm64

# Linux ARM (Raspberry Pi 3, etc.)
GOOS=linux GOARCH=arm go build -o gopiper-linux-arm
```

## 🎮 Usage

### Basic Usage

**1️⃣ Start the server:**
```bash
./gopiper
```

**2️⃣ Open web interface:**
```
http://localhost:3000
```

**3️⃣ Convert text to speech via API:**
```bash
curl -X POST http://localhost:3000/convert \
  -H "Content-Type: application/json" \
  -d '{
    "text": "Hello, this is a test",
    "modelPath": "models/en_US-lessac-medium.onnx"
  }' \
  --output speech.mp3
```

**4️⃣ Play the audio:**
```bash
# Linux
mpg123 speech.mp3

# Windows
start speech.mp3

# macOS
afplay speech.mp3
```

### Configuration

Create a `.env` file in the project root:

```env
# Server configuration
PORT=3000
HOST=127.0.0.1

# Text processing
MAX_TEXT=5000

# Performance (optional, auto-detected by default)
MAX_THREADS=8
```

### Command Line Options

```bash
# Run with custom port
PORT=8080 ./gopiper

# Run with custom host
HOST=0.0.0.0 ./gopiper

# Run on all interfaces
HOST=0.0.0.0 PORT=8080 ./gopiper
```

## 🔌 API

### Endpoints

#### `POST /convert`

Convert text to speech and return MP3 audio.

**Request:**
```json
{
  "text": "Text to convert to speech",
  "modelPath": "models/en_US-lessac-medium.onnx",
  "speaker": 0,
  "noise_scale": 0.667,
  "length_scale": 1.0,
  "noise_w": 0.8
}
```

**Response:** MP3 audio file (binary)

**Example:**
```bash
curl -X POST http://localhost:3000/convert \
  -H "Content-Type: application/json" \
  -d '{"text": "Hello world", "modelPath": "models/en_US-lessac-medium.onnx"}' \
  --output output.mp3
```

#### `GET /models`

List all available voice models.

**Response:**
```json
{
  "success": true,
  "models": [
    {
      "name": "en_US-lessac-medium",
      "path": "models/en_US-lessac-medium.onnx",
      "language": "English (US)"
    }
  ]
}
```

#### `GET /queue-status`

Get current queue status.

**Response:**
```json
{
  "running": 2,
  "queued": 5,
  "maxConcurrent": 8
}
```

#### `GET /settings`

Get current server settings.

**Response:**
```json
{
  "maxThreads": 8,
  "autoDetectThreads": true
}
```

#### `POST /settings`

Update server settings.

**Request:**
```json
{
  "maxThreads": 16,
  "autoDetectThreads": false
}
```

## 🏗️ Architecture

### Supported Platforms

**Windows:**
- ✅ x86_64 (amd64)

**Linux:**
- ✅ x86_64 (amd64) - Standard PCs and servers
- ✅ ARM64 (aarch64) - Raspberry Pi 4/5, AWS Graviton, NVIDIA Jetson
- ✅ ARM (armv7l) - Raspberry Pi 3, older ARM devices

### How It Works

1. **Automatic Download**: On first build, GoPiper downloads the appropriate Piper binary for your OS/architecture
2. **Embedded Resources**: Piper binaries are embedded in the final executable
3. **Temporary Extraction**: On runtime, Piper is extracted to a temporary directory
4. **Parallel Processing**: Text is split into sentences and processed concurrently
5. **Native Audio**: WAV files are concatenated using native Go (no FFmpeg)
6. **MP3 Conversion**: Final audio is converted to MP3 for web delivery

## 🔧 Development

### Project Structure

```
gopiper/
├── main.go              # Server initialization and routing
├── audio.go             # Audio generation and processing
├── audio_native.go      # Native WAV concatenation
├── handlers.go          # HTTP request handlers
├── models.go            # Model scanning and management
├── queue.go             # Task queue implementation
├── text_processing.go   # Text normalization and splitting
├── install_piper.go     # Piper download script
├── web/                 # Embedded web interface
├── piper/               # Auto-downloaded (gitignored)
└── models/              # Voice models (gitignored)
```

### Building

```bash
# Install dependencies
go mod download

# Format code
go fmt ./...

# Run tests
go test ./...

# Build
go build
```

### Make Commands

```bash
make build         # Build for current platform
make run           # Run without building
make windows       # Build for Windows
make linux         # Build for Linux x86_64
make linux-arm64   # Build for Linux ARM64
make linux-arm     # Build for Linux ARM
make build-all     # Build for all platforms
make clean         # Clean build artifacts
make help          # Show all commands
```

## 📝 Voice Models

### Where to Get Models

Download from [Piper Voices on HuggingFace](https://huggingface.co/rhasspy/piper-voices)

**Popular Voices:**

🇺🇸 **English (US)**
- `en_US-lessac-medium` - Clear, neutral voice
- `en_US-amy-medium` - Female voice

🇬🇧 **English (UK)**
- `en_GB-alan-medium` - British male voice

🇪🇸 **Spanish**
- `es_ES-mls_10246-low` - Spain Spanish
- `es_MX-cortana-medium` - Mexican Spanish

🇫🇷 **French**
- `fr_FR-upmc-medium` - French voice

🇩🇪 **German**
- `de_DE-thorsten-medium` - German voice

🇮🇹 **Italian**
- `it_IT-riccardo-medium` - Italian voice

*And 40+ more languages available!*

### Model Format

Each model requires **two files**:
- 📦 `model_name.onnx` - The neural network model
- ⚙️ `model_name.onnx.json` - Model configuration

Place both files in the `models/` directory.

## ⚙️ Configuration

### Environment Variables

**`PORT`** (default: `3000`)
- Server port number

**`HOST`** (default: `127.0.0.1`)
- Server host address
- Use `0.0.0.0` to listen on all interfaces

**`MAX_TEXT`** (default: `0`)
- Maximum text length in characters
- `0` means no limit

### Audio Settings

Adjust in API requests:

- `speaker`: Speaker ID (for multi-speaker models)
- `noise_scale`: Variability in speech (0.0-1.0)
- `length_scale`: Speed of speech (0.5-2.0)
- `noise_w`: Variation in phoneme duration (0.0-1.0)

## 🐛 Troubleshooting

### Piper not found

```bash
# Manually download Piper
go run install_piper.go
```

### Permission denied (Linux)

```bash
chmod +x piper/piper
chmod +x gopiper
```

### Port already in use

```bash
# Use a different port
PORT=8080 ./gopiper
```

### ARM64 build error with resource.syso

The `resource.syso` file is Windows-only. On Linux, it's automatically ignored. If you get errors, ensure you're using the latest code.

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [Piper TTS](https://github.com/rhasspy/piper) - The amazing TTS engine
- [Rhasspy](https://rhasspy.readthedocs.io/) - Voice assistant framework
- Go community for excellent libraries

## 📧 Support

- 🐛 [Report a bug](https://github.com/HirCoir/gopiper/issues)
- 💡 [Request a feature](https://github.com/HirCoir/gopiper/issues)
- 📖 [Documentation](https://github.com/HirCoir/gopiper/wiki)

---

<div align="center">

Made with ❤️ using Go and Piper TTS

⭐ Star this repo if you find it useful!

</div>
