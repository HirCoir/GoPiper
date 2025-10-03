package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type Model struct {
	ID           string          `json:"id"`
	Name         string          `json:"name"`
	Description  string          `json:"description"`
	Language     string          `json:"language"`
	VoicePrompt  string          `json:"voiceprompt"`
	JSONPath     string          `json:"jsonPath"`
	OnnxPath     string          `json:"onnxPath"`
	Image        string          `json:"image,omitempty"`
	Replacements [][]string      `json:"replacements"`
	Source       string          `json:"source"`
}

type ModelCard struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	Description  string     `json:"description"`
	Language     string     `json:"language"`
	VoicePrompt  string     `json:"voiceprompt"`
	Image        string     `json:"image"`
	Replacements [][]string `json:"replacements"`
}

type ModelData struct {
	ModelCard ModelCard `json:"modelcard"`
}

func scanModels() error {
	log.Printf("[SCAN] 🔍 Starting model scan...")
	availableModels = []Model{}

	for _, modelPath := range modelPaths {
		log.Printf("[SCAN] 📁 Scanning directory: %s", modelPath)
		
		if _, err := os.Stat(modelPath); os.IsNotExist(err) {
			log.Printf("[SCAN] ❌ Model path does not exist: %s", modelPath)
			continue
		}

		files, err := os.ReadDir(modelPath)
		if err != nil {
			log.Printf("[SCAN] ❌ Error reading directory %s: %v", modelPath, err)
			continue
		}

		log.Printf("[SCAN] 📄 Found %d files in %s", len(files), filepath.Base(modelPath))

		for _, file := range files {
			if file.IsDir() {
				continue
			}

			fileName := file.Name()
			if !strings.HasSuffix(fileName, ".onnx.json") {
				continue
			}

			jsonPath := filepath.Join(modelPath, fileName)
			onnxPath := filepath.Join(modelPath, strings.TrimSuffix(fileName, ".json"))

			// Check if corresponding .onnx file exists
			if _, err := os.Stat(onnxPath); os.IsNotExist(err) {
				log.Printf("[SCAN] ⚠️  Missing .onnx file for %s", fileName)
				continue
			}

			// Read and parse model data
			model, err := loadModel(jsonPath, onnxPath, modelPath)
			if err != nil {
				log.Printf("[SCAN] ❌ Error reading model %s: %v", fileName, err)
				continue
			}

			availableModels = append(availableModels, model)
			log.Printf("[SCAN] ✅ Found model: %s (%s) [%s]", model.Name, model.ID, model.Language)
		}
	}

	log.Printf("[SCAN] 🎯 Total models found: %d", len(availableModels))
	return nil
}

func loadModel(jsonPath, onnxPath, source string) (Model, error) {
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return Model{}, err
	}

	var modelData ModelData
	if err := json.Unmarshal(data, &modelData); err != nil {
		return Model{}, err
	}

	mc := modelData.ModelCard
	
	// Extract base64 image if it exists
	var imageBase64 string
	if mc.Image != "" {
		imageBase64 = processImageData(mc.Image)
	}

	// Process replacements
	replacements := mc.Replacements
	if len(replacements) == 0 {
		// Default replacements
		replacements = [][]string{
			{"\n", " . "},
			{"*", ""},
			{")", ","},
		}
	}

	// Get model ID from filename if not in modelcard
	modelID := mc.ID
	if modelID == "" {
		modelID = strings.TrimSuffix(filepath.Base(jsonPath), ".onnx.json")
	}

	// Get model name
	modelName := mc.Name
	if modelName == "" {
		modelName = modelID
	}

	model := Model{
		ID:           modelID,
		Name:         modelName,
		Description:  getOrDefault(mc.Description, "No description available"),
		Language:     getOrDefault(mc.Language, "Unknown"),
		VoicePrompt:  getOrDefault(mc.VoicePrompt, "Not available"),
		JSONPath:     jsonPath,
		OnnxPath:     onnxPath,
		Image:        imageBase64,
		Replacements: replacements,
		Source:       source,
	}

	return model, nil
}

func processImageData(imageData string) string {
	// Extract base64 data from data URI
	if strings.Contains(imageData, "base64,") {
		parts := strings.SplitN(imageData, "base64,", 2)
		if len(parts) == 2 {
			return parts[1]
		}
	}
	return imageData
}

func getOrDefault(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}

func findModelByPath(onnxPath string) (*Model, error) {
	for i := range availableModels {
		if availableModels[i].OnnxPath == onnxPath {
			return &availableModels[i], nil
		}
	}
	return nil, fmt.Errorf("model not found")
}
