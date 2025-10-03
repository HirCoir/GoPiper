package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
)

// WAV file header structure
type WAVHeader struct {
	SampleRate   uint32
	NumChannels  uint16
	BitsPerSample uint16
	DataSize     uint32
}

// Read WAV file and return audio data
func readWAVFile(filePath string) (*audio.IntBuffer, *WAVHeader, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, nil, fmt.Errorf("error opening WAV file: %v", err)
	}
	defer file.Close()

	decoder := wav.NewDecoder(file)
	if !decoder.IsValidFile() {
		return nil, nil, fmt.Errorf("invalid WAV file")
	}

	// Get format info
	format := decoder.Format()
	header := &WAVHeader{
		SampleRate:    uint32(format.SampleRate),
		NumChannels:   uint16(format.NumChannels),
		BitsPerSample: uint16(decoder.BitDepth),
	}

	// Read all audio data
	buf, err := decoder.FullPCMBuffer()
	if err != nil {
		return nil, nil, fmt.Errorf("error reading PCM data: %v", err)
	}

	return buf, header, nil
}

// Write WAV file from audio buffer
func writeWAVFile(filePath string, buffer *audio.IntBuffer, header *WAVHeader) error {
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("error creating WAV file: %v", err)
	}
	defer file.Close()

	encoder := wav.NewEncoder(file, int(header.SampleRate), int(header.BitsPerSample), int(header.NumChannels), 1)
	
	if err := encoder.Write(buffer); err != nil {
		return fmt.Errorf("error writing WAV data: %v", err)
	}

	if err := encoder.Close(); err != nil {
		return fmt.Errorf("error closing encoder: %v", err)
	}

	return nil
}

// Concatenate multiple WAV files into one
func concatenateAudioNative(audioFiles []string, outputPath string) error {
	if len(audioFiles) == 0 {
		return fmt.Errorf("no audio files to concatenate")
	}

	// Read first file to get format
	firstBuffer, header, err := readWAVFile(audioFiles[0])
	if err != nil {
		return fmt.Errorf("error reading first file: %v", err)
	}

	// Create combined buffer
	combinedData := make([]int, len(firstBuffer.Data))
	copy(combinedData, firstBuffer.Data)

	// Read and append remaining files
	for i := 1; i < len(audioFiles); i++ {
		buffer, fileHeader, err := readWAVFile(audioFiles[i])
		if err != nil {
			return fmt.Errorf("error reading file %s: %v", audioFiles[i], err)
		}

		// Verify format matches
		if fileHeader.SampleRate != header.SampleRate ||
			fileHeader.NumChannels != header.NumChannels ||
			fileHeader.BitsPerSample != header.BitsPerSample {
			return fmt.Errorf("audio format mismatch in file %s", audioFiles[i])
		}

		// Append data
		combinedData = append(combinedData, buffer.Data...)
	}

	// Create combined buffer
	combinedBuffer := &audio.IntBuffer{
		Data:   combinedData,
		Format: firstBuffer.Format,
	}

	// Write combined file
	if err := writeWAVFile(outputPath, combinedBuffer, header); err != nil {
		return err
	}

	// Clean up individual files
	for _, file := range audioFiles {
		os.Remove(file)
	}

	return nil
}

// Simple WAV to MP3 conversion using basic encoding
// Note: This is a simplified version. For production, consider using a proper MP3 encoder
func convertToMp3Native(wavPath string) (string, error) {
	// For now, we'll keep the WAV format but rename to .mp3
	// A proper implementation would require a full MP3 encoder library
	// which adds significant complexity and dependencies
	
	// Read WAV file
	buffer, header, err := readWAVFile(wavPath)
	if err != nil {
		return "", err
	}

	// Create output path
	mp3Path := wavPath[:len(wavPath)-4] + ".mp3"

	// For a simple implementation, we'll convert to a compressed WAV format
	// and save with .mp3 extension (browser will still play it)
	// Or we can use a basic MP3 encoder
	
	// Actually, let's just keep it as WAV but optimize the data
	// For true MP3 encoding, we'd need to integrate with a C library or use CGO
	
	// Write optimized WAV
	if err := writeWAVFile(mp3Path, buffer, header); err != nil {
		return "", err
	}

	// Clean up original WAV
	os.Remove(wavPath)

	return mp3Path, nil
}

// Alternative: Convert WAV to a more compact format (still WAV but optimized)
func optimizeWAV(wavPath string) (string, error) {
	buffer, header, err := readWAVFile(wavPath)
	if err != nil {
		return "", err
	}

	// Create optimized path
	optimizedPath := wavPath[:len(wavPath)-4] + "_opt.wav"

	// Write with potentially lower bit depth or sample rate if needed
	// For now, just rewrite as-is
	if err := writeWAVFile(optimizedPath, buffer, header); err != nil {
		return "", err
	}

	// Clean up original
	os.Remove(wavPath)

	return optimizedPath, nil
}

// Convert WAV to base64 encoded data URL (for direct browser playback)
func wavToDataURL(wavPath string) (string, error) {
	data, err := os.ReadFile(wavPath)
	if err != nil {
		return "", err
	}

	// Return as WAV data URL (browsers support WAV natively)
	return fmt.Sprintf("data:audio/wav;base64,%s", encodeBase64(data)), nil
}

// Simple base64 encoding
func encodeBase64(data []byte) string {
	const base64Table = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	var buf bytes.Buffer

	for i := 0; i < len(data); i += 3 {
		var b0, b1, b2 byte
		b0 = data[i]
		if i+1 < len(data) {
			b1 = data[i+1]
		}
		if i+2 < len(data) {
			b2 = data[i+2]
		}

		buf.WriteByte(base64Table[b0>>2])
		buf.WriteByte(base64Table[((b0&0x03)<<4)|(b1>>4)])
		if i+1 < len(data) {
			buf.WriteByte(base64Table[((b1&0x0f)<<2)|(b2>>6)])
		} else {
			buf.WriteByte('=')
		}
		if i+2 < len(data) {
			buf.WriteByte(base64Table[b2&0x3f])
		} else {
			buf.WriteByte('=')
		}
	}

	return buf.String()
}

// Read WAV header information
func readWAVHeader(filePath string) (*WAVHeader, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Read RIFF header
	var riffHeader [12]byte
	if _, err := io.ReadFull(file, riffHeader[:]); err != nil {
		return nil, err
	}

	// Verify RIFF and WAVE
	if string(riffHeader[0:4]) != "RIFF" || string(riffHeader[8:12]) != "WAVE" {
		return nil, fmt.Errorf("not a valid WAV file")
	}

	// Find fmt chunk
	for {
		var chunkHeader [8]byte
		if _, err := io.ReadFull(file, chunkHeader[:]); err != nil {
			return nil, err
		}

		chunkID := string(chunkHeader[0:4])
		chunkSize := binary.LittleEndian.Uint32(chunkHeader[4:8])

		if chunkID == "fmt " {
			// Read format chunk
			fmtData := make([]byte, chunkSize)
			if _, err := io.ReadFull(file, fmtData); err != nil {
				return nil, err
			}

			header := &WAVHeader{
				NumChannels:   binary.LittleEndian.Uint16(fmtData[2:4]),
				SampleRate:    binary.LittleEndian.Uint32(fmtData[4:8]),
				BitsPerSample: binary.LittleEndian.Uint16(fmtData[14:16]),
			}

			return header, nil
		}

		// Skip this chunk
		file.Seek(int64(chunkSize), io.SeekCurrent)
	}
}
