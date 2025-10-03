// +build ignore

package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
)

const (
	piperVersion    = "2023.11.14-2"
	baseURL         = "https://github.com/rhasspy/piper/releases/download/2023.11.14-2"
	piperDir        = "piper"
)

func main() {
	fmt.Println("🚀 Installing Piper TTS...")
	
	// Check if piper directory already exists
	if _, err := os.Stat(piperDir); err == nil {
		fmt.Println("✅ Piper directory already exists, skipping download")
		return
	}

	// Determine download URL based on OS and architecture
	var downloadURL string
	var isZip bool
	
	switch runtime.GOOS {
	case "windows":
		if runtime.GOARCH != "amd64" {
			fmt.Printf("❌ Unsupported Windows architecture: %s (only amd64 is supported)\n", runtime.GOARCH)
			os.Exit(1)
		}
		downloadURL = baseURL + "/piper_windows_amd64.zip"
		isZip = true
		fmt.Println("📦 Detected Windows amd64 - downloading...")
		
	case "linux":
		isZip = false
		switch runtime.GOARCH {
		case "amd64":
			downloadURL = baseURL + "/piper_linux_x86_64.tar.gz"
			fmt.Println("📦 Detected Linux x86_64 - downloading...")
		case "arm64":
			downloadURL = baseURL + "/piper_linux_aarch64.tar.gz"
			fmt.Println("📦 Detected Linux aarch64 (ARM64) - downloading...")
		case "arm":
			downloadURL = baseURL + "/piper_linux_armv7l.tar.gz"
			fmt.Println("📦 Detected Linux armv7l (ARM 32-bit) - downloading...")
		default:
			fmt.Printf("❌ Unsupported Linux architecture: %s\n", runtime.GOARCH)
			fmt.Println("Supported: amd64, arm64, arm")
			os.Exit(1)
		}
		
	default:
		fmt.Printf("❌ Unsupported OS: %s\n", runtime.GOOS)
		fmt.Println("Supported: windows, linux")
		os.Exit(1)
	}

	// Download file
	fmt.Printf("⬇️  Downloading from: %s\n", downloadURL)
	tmpFile, err := downloadFile(downloadURL)
	if err != nil {
		fmt.Printf("❌ Download failed: %v\n", err)
		os.Exit(1)
	}
	defer os.Remove(tmpFile)

	// Extract archive
	fmt.Println("📂 Extracting files...")
	if isZip {
		if err := extractZip(tmpFile, "."); err != nil {
			fmt.Printf("❌ Extraction failed: %v\n", err)
			os.Exit(1)
		}
	} else {
		if err := extractTarGz(tmpFile, "."); err != nil {
			fmt.Printf("❌ Extraction failed: %v\n", err)
			os.Exit(1)
		}
	}

	fmt.Println("✅ Piper installed successfully!")
}

func downloadFile(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bad status: %s", resp.Status)
	}

	tmpFile, err := os.CreateTemp("", "piper-*.tmp")
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()

	_, err = io.Copy(tmpFile, resp.Body)
	if err != nil {
		os.Remove(tmpFile.Name())
		return "", err
	}

	return tmpFile.Name(), nil
}

func extractZip(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(destDir, f.Name)

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}
	return nil
}

func extractTarGz(tarGzPath, destDir string) error {
	file, err := os.Open(tarGzPath)
	if err != nil {
		return err
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target := filepath.Join(destDir, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}

			outFile, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return err
			}

			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()
		}
	}

	return nil
}
