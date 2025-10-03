package main

//go:generate go run install_piper.go

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/common-nighthawk/go-figure"
	"github.com/joho/godotenv"
)

//go:embed piper
var piperFS embed.FS

//go:embed web
var webFS embed.FS

var (
	modelPaths      []string
	availableModels []Model
	piperPath       string
	tempPiperDir    string
	processQueue    *ProcessQueue
	userSettings    Settings
	cpuCores        int
	maxTextLength   int = 0 // 0 means no limit
)

type Settings struct {
	MaxThreads        int  `json:"maxThreads"`
	AutoDetectThreads bool `json:"autoDetectThreads"`
}

func main() {
	// Seed random number generator
	rand.Seed(time.Now().UnixNano())
	cpuCores = runtime.NumCPU()
	log.Printf("[SERVER] 🚀 Starting Piper TTS Go Server...")
	log.Printf("[SERVER] 💻 CPU cores detected: %d", cpuCores)

	// Extract embedded piper files to temp directory
	if err := extractEmbeddedPiper(); err != nil {
		log.Printf("[SERVER] ⚠️  Could not extract embedded piper: %v", err)
		log.Printf("[SERVER] Will try to use local piper directory if available")
	}

	// Setup cleanup on exit
	setupCleanup()

	// Initialize process queue
	maxConcurrent := cpuCores * 2
	processQueue = NewProcessQueue(maxConcurrent)
	
	userSettings = Settings{
		MaxThreads:        maxConcurrent,
		AutoDetectThreads: true,
	}

	// Initialize paths
	initializePaths()

	// Initialize model paths
	if err := initializeModelPaths(); err != nil {
		log.Printf("[MODELS] Warning: %v", err)
	}

	// Scan models
	if err := scanModels(); err != nil {
		log.Printf("[SCAN] Warning: %v", err)
	}

	// Setup router
	router := mux.NewRouter()
	
	// Enable CORS
	router.Use(corsMiddleware)
	
	// Routes
	router.HandleFunc("/models", getModelsHandler).Methods("GET")
	router.HandleFunc("/set-model-paths", setModelPathsHandler).Methods("POST")
	router.HandleFunc("/convert", convertHandler).Methods("POST")
	router.HandleFunc("/rescan-models", rescanModelsHandler).Methods("GET")
	router.HandleFunc("/settings", getSettingsHandler).Methods("GET")
	router.HandleFunc("/settings", updateSettingsHandler).Methods("POST")
	router.HandleFunc("/queue-status", getQueueStatusHandler).Methods("GET")
	
	// Serve static files from embedded web directory
	webSubFS, err := fs.Sub(webFS, "web")
	if err != nil {
		log.Fatal(err)
	}
	fileServer := http.FileServer(http.FS(webSubFS))
	router.PathPrefix("/").Handler(fileServer)

	// Load environment variables
	loadEnv()
	
	// Start server
	port := getEnv("PORT", "3000")
	host := getEnv("HOST", "127.0.0.1")
	
	// Display stylized banner
	fmt.Println()
	myFigure := figure.NewFigure("GoPiper", "", true)
	myFigure.Print()
	fmt.Println()
	
	// Try to start server with port availability checking
	if err := startServer(router, host, port); err != nil {
		log.Fatal(err)
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

// Extract embedded piper files to temporary directory
func extractEmbeddedPiper() error {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "ttsgo-piper-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %v", err)
	}
	
	tempPiperDir = tempDir
	log.Printf("[EMBED] 📦 Extracting piper to: %s", tempPiperDir)

	// Walk through embedded files
	err = fs.WalkDir(piperFS, "piper", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Get relative path
		relPath, err := filepath.Rel("piper", path)
		if err != nil {
			return err
		}

		targetPath := filepath.Join(tempPiperDir, relPath)

		if d.IsDir() {
			// Create directory
			return os.MkdirAll(targetPath, 0755)
		}

		// Read embedded file
		data, err := piperFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read %s: %v", path, err)
		}

		// Write to temp location
		if err := os.WriteFile(targetPath, data, 0755); err != nil {
			return fmt.Errorf("failed to write %s: %v", targetPath, err)
		}

		log.Printf("[EMBED] ✅ Extracted: %s", relPath)
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to extract files: %v", err)
	}

	log.Printf("[EMBED] 🎉 All piper files extracted successfully")
	
	// Create symbolic links for shared libraries (Linux only)
	if runtime.GOOS == "linux" {
		if err := createLibrarySymlinks(); err != nil {
			log.Printf("[EMBED] ⚠️  Warning: Could not create library symlinks: %v", err)
		}
	}
	
	return nil
}

// Create symbolic links for shared libraries
func createLibrarySymlinks() error {
	// Define symlinks as pairs: [target, link_name]
	symlinks := [][2]string{
		{"libespeak-ng.so.1.52.0.1", "libespeak-ng.so.1"},
		{"libespeak-ng.so.1.52.0.1", "libespeak-ng.so"},
		{"libonnxruntime.so.1.14.1", "libonnxruntime.so.1"},
		{"libonnxruntime.so.1.14.1", "libonnxruntime.so"},
		{"libpiper_phonemize.so.1.2.0", "libpiper_phonemize.so.1"},
		{"libpiper_phonemize.so.1.2.0", "libpiper_phonemize.so"},
	}

	for _, pair := range symlinks {
		target := pair[0]
		linkName := pair[1]
		
		targetPath := filepath.Join(tempPiperDir, target)
		linkPath := filepath.Join(tempPiperDir, linkName)

		// Check if target exists
		if _, err := os.Stat(targetPath); os.IsNotExist(err) {
			continue
		}

		// Remove existing symlink if it exists
		os.Remove(linkPath)

		// Create symlink (relative path)
		if err := os.Symlink(target, linkPath); err != nil {
			log.Printf("[SYMLINK] ⚠️  Failed to create %s -> %s: %v", linkName, target, err)
		} else {
			log.Printf("[SYMLINK] ✅ Created: %s -> %s", linkName, target)
		}
	}

	return nil
}

// Setup cleanup handler for Ctrl+C and window close
func setupCleanup() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	
	go func() {
		<-c
		log.Printf("[SERVER] 🛑 Shutdown signal received, cleaning up...")
		cleanup()
		os.Exit(0)
	}()
}

// Cleanup temporary files
func cleanup() {
	if tempPiperDir != "" {
		log.Printf("[CLEANUP] 🧹 Removing temporary piper directory: %s", tempPiperDir)
		if err := os.RemoveAll(tempPiperDir); err != nil {
			log.Printf("[CLEANUP] ⚠️  Error removing temp directory: %v", err)
		} else {
			log.Printf("[CLEANUP] ✅ Temporary files cleaned up")
		}
	}
}

func initializePaths() {
	// Use temp directory if piper was extracted, otherwise use local
	var piperDir string
	
	if tempPiperDir != "" {
		piperDir = tempPiperDir
		log.Printf("[PATHS] Using extracted piper from: %s", piperDir)
	} else {
		// Fallback to local piper directory
		currentDir, err := os.Getwd()
		if err != nil {
			log.Printf("[PATHS] Error getting current directory: %v", err)
			currentDir = "."
		}
		piperDir = filepath.Join(currentDir, "piper")
		log.Printf("[PATHS] Using local piper from: %s", piperDir)
	}

	// Set paths based on OS
	if runtime.GOOS == "windows" {
		piperPath = filepath.Join(piperDir, "piper.exe")
	} else {
		piperPath = filepath.Join(piperDir, "piper")
	}

	log.Printf("[PATHS] Piper executable: %s", piperPath)
	log.Printf("[PATHS] Using native Go audio processing (no FFmpeg required)")
	
	// Verify piper exists
	if _, err := os.Stat(piperPath); os.IsNotExist(err) {
		log.Printf("[PATHS] ⚠️  WARNING: Piper executable not found at %s", piperPath)
	} else {
		log.Printf("[PATHS] ✅ Piper executable found")
	}
}

func initializeModelPaths() error {
	modelPaths = []string{}

	// Check local ./models directory
	localModelsPath := filepath.Join(".", "models")
	if _, err := os.Stat(localModelsPath); err == nil {
		modelPaths = append(modelPaths, localModelsPath)
		log.Printf("[MODELS] ✅ Found local models directory: %s", localModelsPath)
	} else {
		log.Printf("[MODELS] ❌ Local models directory not found: %s", localModelsPath)
	}

	// Add Documents path
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("error getting home directory: %v", err)
	}

	onnxTtsPath := filepath.Join(homeDir, "Documents", "onnx-tts")
	if _, err := os.Stat(onnxTtsPath); err == nil {
		modelPaths = append(modelPaths, onnxTtsPath)
		log.Printf("[MODELS] ✅ Found Documents models directory: %s", onnxTtsPath)
	} else {
		log.Printf("[MODELS] ❌ Documents models directory not found: %s", onnxTtsPath)
		// Still add it in case it gets created later
		modelPaths = append(modelPaths, onnxTtsPath)
	}

	log.Printf("[MODELS] Initialized model paths: %v", modelPaths)
	return nil
}

// Response helpers
func jsonResponse(w http.ResponseWriter, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

func errorResponse(w http.ResponseWriter, message string, statusCode int) {
	jsonResponse(w, map[string]interface{}{
		"success": false,
		"error":   message,
	}, statusCode)
}

// Load environment variables from .env file
func loadEnv() {
	// Try to load .env file
	if err := godotenv.Load(); err != nil {
		log.Printf("[ENV] ⚠️  No .env file found, using defaults: %v", err)
	} else {
		log.Println("[ENV] ✅ Environment variables loaded from .env file")
	}
	
	// Load MAX_TEXT if set
	if maxTextStr := os.Getenv("MAX_TEXT"); maxTextStr != "" {
		if maxText, err := strconv.Atoi(maxTextStr); err == nil {
			maxTextLength = maxText
			log.Printf("[ENV] ✅ Max text length set to %d characters", maxTextLength)
		} else {
			log.Printf("[ENV] ⚠️  Invalid MAX_TEXT value: %s", maxTextStr)
		}
	}
}

// Get environment variable with default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Start server with port availability checking
func startServer(router *mux.Router, host, port string) error {
	addr := host + ":" + port
	
	// Check if port is available before starting server
	if isPortAvailable(addr) {
		log.Printf("[SERVER] ✅ TTS Server running on http://%s", addr)
		log.Printf("[SERVER] 🌐 Open your browser and go to: http://%s:%s", host, port)
		return http.ListenAndServe(addr, router)
	}
	
	// If port is in use, try random ports
	log.Printf("[SERVER] ⚠️  Port %s is in use, trying random ports...", port)
	for i := 0; i < 10; i++ { // Try up to 10 random ports
		randomPort := getRandomPort()
		addr := host + ":" + randomPort
		
		if isPortAvailable(addr) {
			log.Printf("[SERVER] ✅ TTS Server running on http://%s", addr)
			log.Printf("[SERVER] 🌐 Open your browser and go to: http://%s:%s", host, randomPort)
			return http.ListenAndServe(addr, router)
		}
		
		log.Printf("[SERVER] ⚠️  Port %s is also in use, trying another...", randomPort)
	}
	
	return fmt.Errorf("no available ports found after 10 attempts")
}

// Check if a port is available by attempting to listen on it
func isPortAvailable(addr string) bool {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return false
	}
	listener.Close()
	return true
}

// Check if error indicates port is in use
func isPortInUse(err error) bool {
	// Check if the error indicates the port is in use
	return err != nil && (err.Error() == "listen tcp: address already in use" || 
		strings.Contains(err.Error(), "bind: address already in use") ||
		strings.Contains(err.Error(), "Only one usage of each socket address"))
}

// Generate a random port between 3001-9999
func getRandomPort() string {
	return strconv.Itoa(3001 + rand.Intn(6998))
}
