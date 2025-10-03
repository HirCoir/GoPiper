package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

// maxTextLength is defined in main.go as a global variable
// It limits the maximum text length for TTS conversion (0 = no limit)

// GET /models - Get available models
func getModelsHandler(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, map[string]interface{}{
		"success": true,
		"models":  availableModels,
		"count":   len(availableModels),
	}, http.StatusOK)
}

// POST /set-model-paths - Set model paths
func setModelPathsHandler(w http.ResponseWriter, r *http.Request) {
	var requestData struct {
		Paths []string `json:"paths"`
	}

	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		errorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if requestData.Paths == nil {
		errorResponse(w, "Paths must be an array", http.StatusBadRequest)
		return
	}

	modelPaths = requestData.Paths
	if err := scanModels(); err != nil {
		errorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, map[string]interface{}{
		"success":    true,
		"message":    "Model paths updated",
		"modelCount": len(availableModels),
	}, http.StatusOK)
}

// POST /convert - Convert text to speech
func convertHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("[DEBUG] 🚀 /convert route called")

	var requestData struct {
		Text      string                 `json:"text"`
		ModelPath string                 `json:"modelPath"`
		Settings  map[string]interface{} `json:"settings"`
	}

	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		errorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	log.Printf("[DEBUG] 📥 Received request - text length: %d, modelPath: %s", len(requestData.Text), requestData.ModelPath)

	if requestData.Text == "" {
		errorResponse(w, "Text is required", http.StatusBadRequest)
		return
	}

	// Check MAX_TEXT limit if set
	if maxTextLength > 0 && len(requestData.Text) > maxTextLength {
		errorResponse(w, fmt.Sprintf("Text exceeds maximum length of %d characters", maxTextLength), http.StatusBadRequest)
		return
	}

	if requestData.ModelPath == "" {
		errorResponse(w, "Model path is required", http.StatusBadRequest)
		return
	}

	// Find model by path
	model, err := findModelByPath(requestData.ModelPath)
	if err != nil {
		errorResponse(w, "Model not found", http.StatusNotFound)
		return
	}

	log.Printf("[CONVERT] 🎤 Converting text with model: %s (%s)", model.Name, model.Language)
	log.Printf("[CONVERT] 📝 Input text length: %d characters", len(requestData.Text))
	log.Printf("[CONVERT] 🔧 About to start text filtering...")

	// Apply comprehensive text filtering and replacements
	processedText := filterTextSegment(requestData.Text, model.Replacements)

	log.Printf("[CONVERT] 🔧 Text filtering completed")

	if processedText == "" {
		log.Printf("[CONVERT] ❌ Text became empty after processing")
		errorResponse(w, "Text became empty after processing", http.StatusBadRequest)
		return
	}

	log.Printf("[CONVERT] ✅ Text ready for synthesis: '%s'", truncateString(processedText, 100))

	// Split into sentences
	sentences := splitSentences(processedText)
	log.Printf("[CONVERT] 📄 Split into %d sentences", len(sentences))

	if len(sentences) == 0 {
		errorResponse(w, "No valid sentences found in text", http.StatusBadRequest)
		return
	}

	// Parse audio settings
	settings := parseAudioSettings(requestData.Settings)

	// Generate audio for all sentences in parallel
	validSentences := []string{}
	for _, s := range sentences {
		if s != "" {
			validSentences = append(validSentences, s)
		}
	}

	audioFiles, err := generateAudioParallel(validSentences, requestData.ModelPath, settings)
	if err != nil {
		log.Printf("[CONVERT] ❌ Error generating audio: %v", err)
		errorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if len(audioFiles) == 0 {
		errorResponse(w, "Failed to generate any audio", http.StatusInternalServerError)
		return
	}

	var finalAudioPath string

	if len(audioFiles) == 1 {
		finalAudioPath = audioFiles[0]
		log.Printf("[CONVERT] 🎵 Using single audio file")
	} else {
		// Concatenate multiple audio files
		log.Printf("[CONVERT] 🔗 Concatenating %d audio files", len(audioFiles))
		concatenatedPath := filepath.Join(os.TempDir(), fmt.Sprintf("final_%s.wav", generateRandomString(8)))
		if err := concatenateAudio(audioFiles, concatenatedPath); err != nil {
			log.Printf("[CONVERT] ❌ Error concatenating audio: %v", err)
			errorResponse(w, err.Error(), http.StatusInternalServerError)
			return
		}
		finalAudioPath = concatenatedPath
	}

	// Read the WAV file and encode as base64 (no conversion needed, browsers support WAV)
	log.Printf("[CONVERT] 🎵 Reading audio file...")
	audioBuffer, err := os.ReadFile(finalAudioPath)
	if err != nil {
		log.Printf("[CONVERT] ❌ Error reading audio file: %v", err)
		errorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer os.Remove(finalAudioPath)

	audioBase64 := base64.StdEncoding.EncodeToString(audioBuffer)
	audioSizeKB := len(audioBuffer) / 1024

	log.Printf("[CONVERT] ✅ Conversion completed! Audio size: %dKB (WAV format)", audioSizeKB)

	jsonResponse(w, map[string]interface{}{
		"success":       true,
		"audio":         fmt.Sprintf("data:audio/wav;base64,%s", audioBase64),
		"model":         model.Name,
		"sentenceCount": len(sentences),
	}, http.StatusOK)
}

// GET /rescan-models - Rescan models
func rescanModelsHandler(w http.ResponseWriter, r *http.Request) {
	if err := scanModels(); err != nil {
		errorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, map[string]interface{}{
		"success":    true,
		"message":    "Models rescanned",
		"modelCount": len(availableModels),
	}, http.StatusOK)
}

// GET /settings - Get current settings
func getSettingsHandler(w http.ResponseWriter, r *http.Request) {
	queueStatus := processQueue.GetStatus()

	jsonResponse(w, map[string]interface{}{
		"success": true,
		"settings": map[string]interface{}{
			"maxThreads":           userSettings.MaxThreads,
			"autoDetectThreads":    userSettings.AutoDetectThreads,
			"cpuCores":             cpuCores,
			"currentMaxConcurrent": queueStatus.MaxConcurrent,
			"recommendedThreads":   cpuCores * 2,
		},
		"queueStatus": queueStatus,
	}, http.StatusOK)
}

// POST /settings - Update settings
func updateSettingsHandler(w http.ResponseWriter, r *http.Request) {
	var requestData struct {
		MaxThreads        *int  `json:"maxThreads"`
		AutoDetectThreads *bool `json:"autoDetectThreads"`
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		errorResponse(w, "Error reading request body", http.StatusBadRequest)
		return
	}

	if err := json.Unmarshal(body, &requestData); err != nil {
		errorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if requestData.AutoDetectThreads != nil {
		userSettings.AutoDetectThreads = *requestData.AutoDetectThreads
	}

	if requestData.MaxThreads != nil && *requestData.MaxThreads > 0 {
		maxThreads := *requestData.MaxThreads
		if maxThreads < 1 {
			maxThreads = 1
		}
		if maxThreads > 32 {
			maxThreads = 32
		}
		userSettings.MaxThreads = maxThreads

		if !userSettings.AutoDetectThreads {
			processQueue.SetMaxConcurrent(userSettings.MaxThreads)
		}
	}

	// If auto-detect is enabled, use CPU-based calculation
	if userSettings.AutoDetectThreads {
		autoThreads := cpuCores * 2
		processQueue.SetMaxConcurrent(autoThreads)
		userSettings.MaxThreads = autoThreads
	}

	queueStatus := processQueue.GetStatus()

	jsonResponse(w, map[string]interface{}{
		"success": true,
		"message": "Settings updated",
		"settings": map[string]interface{}{
			"maxThreads":           userSettings.MaxThreads,
			"autoDetectThreads":    userSettings.AutoDetectThreads,
			"cpuCores":             cpuCores,
			"currentMaxConcurrent": queueStatus.MaxConcurrent,
		},
		"queueStatus": queueStatus,
	}, http.StatusOK)
}

// GET /queue-status - Get queue status
func getQueueStatusHandler(w http.ResponseWriter, r *http.Request) {
	queueStatus := processQueue.GetStatus()

	jsonResponse(w, map[string]interface{}{
		"success": true,
		"status":  queueStatus,
	}, http.StatusOK)
}
