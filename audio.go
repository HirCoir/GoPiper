package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
)

type AudioSettings struct {
	Speaker     int     `json:"speaker"`
	NoiseScale  float64 `json:"noise_scale"`
	LengthScale float64 `json:"length_scale"`
	NoiseW      float64 `json:"noise_w"`
}

type SentenceResult struct {
	Index     int
	AudioFile string
	Sentence  string
	Error     error
}

// Generate audio using Piper
func generateAudio(text, modelPath string, settings AudioSettings) (string, error) {
	outputFile := filepath.Join(os.TempDir(), fmt.Sprintf("tts_%s.wav", generateRandomString(8)))

	args := []string{
		"-m", modelPath,
		"-f", outputFile,
		"--speaker", strconv.Itoa(settings.Speaker),
		"--noise-scale", fmt.Sprintf("%.3f", settings.NoiseScale),
		"--length-scale", fmt.Sprintf("%.3f", settings.LengthScale),
		"--noise-w", fmt.Sprintf("%.3f", settings.NoiseW),
	}

	log.Printf("Piper command: %s %v", piperPath, args)
	log.Printf("Input text: %s", text)

	cmd := exec.Command(piperPath, args...)
	
	// Set LD_LIBRARY_PATH for Linux to find shared libraries
	if tempPiperDir != "" {
		// Get current environment
		env := os.Environ()
		
		// Get existing LD_LIBRARY_PATH
		existingPath := os.Getenv("LD_LIBRARY_PATH")
		
		// Build new LD_LIBRARY_PATH with temp piper directory first
		var newPath string
		if existingPath != "" {
			newPath = tempPiperDir + ":" + existingPath
		} else {
			newPath = tempPiperDir + ":/usr/local/lib:/usr/lib:/lib"
		}
		
		// Add LD_LIBRARY_PATH to command environment
		env = append(env, "LD_LIBRARY_PATH="+newPath)
		cmd.Env = env
		
		log.Printf("[LIBRARY] Setting LD_LIBRARY_PATH to: %s", newPath)
	}
	
	// Create stdin pipe
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", fmt.Errorf("error creating stdin pipe: %v", err)
	}

	// Capture stderr
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	// Start the command
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("error starting piper: %v", err)
	}

	// Write text to stdin
	if _, err := stdin.Write([]byte(text)); err != nil {
		return "", fmt.Errorf("error writing to stdin: %v", err)
	}
	stdin.Close()

	// Wait for command to finish
	if err := cmd.Wait(); err != nil {
		return "", fmt.Errorf("piper failed: %v - %s", err, stderr.String())
	}

	// Check if output file exists
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		return "", fmt.Errorf("output file not created: %s", outputFile)
	}

	return outputFile, nil
}

// Generate audio for multiple sentences in parallel
func generateAudioParallel(sentences []string, modelPath string, settings AudioSettings) ([]string, error) {
	queueStatus := processQueue.GetStatus()
	log.Printf("[PARALLEL] Processing %d sentences with max %d concurrent processes", len(sentences), queueStatus.MaxConcurrent)
	log.Printf("[PARALLEL] Queue status - Running: %d, Queued: %d", queueStatus.Running, queueStatus.Queued)

	results := make([]SentenceResult, len(sentences))
	var wg sync.WaitGroup
	var mu sync.Mutex

	for i, sentence := range sentences {
		wg.Add(1)
		index := i
		sent := sentence

		go func() {
			defer wg.Done()

			log.Printf("[PARALLEL] Starting sentence %d/%d: \"%s...\"", index+1, len(sentences), truncateString(sent, 50))

			// Add task to queue
			result, err := processQueue.Add(func() (interface{}, error) {
				return generateAudio(sent, modelPath, settings)
			})

			mu.Lock()
			if err != nil {
				log.Printf("[PARALLEL] Error processing sentence %d: %v", index+1, err)
				results[index] = SentenceResult{
					Index:    index,
					Sentence: sent,
					Error:    err,
				}
			} else {
				audioFile := result.(string)
				log.Printf("[PARALLEL] Completed sentence %d/%d", index+1, len(sentences))
				results[index] = SentenceResult{
					Index:     index,
					AudioFile: audioFile,
					Sentence:  sent,
					Error:     nil,
				}
			}
			mu.Unlock()
		}()
	}

	wg.Wait()

	// Check for errors and collect audio files
	audioFiles := []string{}
	for _, result := range results {
		if result.Error != nil {
			// Clean up any generated files
			for _, file := range audioFiles {
				os.Remove(file)
			}
			return nil, fmt.Errorf("error processing sentence %d: %v", result.Index+1, result.Error)
		}
		audioFiles = append(audioFiles, result.AudioFile)
	}

	log.Printf("[PARALLEL] All %d sentences processed successfully", len(sentences))
	return audioFiles, nil
}


// Concatenate multiple audio files using native Go
func concatenateAudio(audioFiles []string, outputPath string) error {
	// Use native Go concatenation only
	if err := concatenateAudioNative(audioFiles, outputPath); err != nil {
		log.Printf("[CONCAT] ❌ Native concatenation failed: %v", err)
		return fmt.Errorf("audio concatenation failed: %v", err)
	}
	
	log.Printf("[CONCAT] ✅ Native Go concatenation successful")
	return nil
}

// Get default audio settings
func getDefaultSettings() AudioSettings {
	return AudioSettings{
		Speaker:     0,
		NoiseScale:  0.667,
		LengthScale: 1.0,
		NoiseW:      0.8,
	}
}

// Parse audio settings from request
func parseAudioSettings(data map[string]interface{}) AudioSettings {
	settings := getDefaultSettings()

	if speaker, ok := data["speaker"].(float64); ok {
		settings.Speaker = int(speaker)
	}
	if noiseScale, ok := data["noise_scale"].(float64); ok {
		settings.NoiseScale = noiseScale
	}
	if lengthScale, ok := data["length_scale"].(float64); ok {
		settings.LengthScale = lengthScale
	}
	if noiseW, ok := data["noise_w"].(float64); ok {
		settings.NoiseW = noiseW
	}

	return settings
}
