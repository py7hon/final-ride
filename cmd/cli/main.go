package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"final-ride/internal/finalride"

	"github.com/schollz/progressbar/v3"
)

// Helper for progress bars
func createProgressBar(max int64, description string) *progressbar.ProgressBar {
	return progressbar.NewOptions64(max,
		progressbar.OptionSetDescription(description),
		progressbar.OptionSetWidth(30),
		progressbar.OptionShowBytes(true),
		progressbar.OptionShowCount(),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "=",
			SaucerHead:    ">",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
		progressbar.OptionOnCompletion(func() {
			fmt.Println()
		}),
	)
}

func createCountProgressBar(max int64, description string) *progressbar.ProgressBar {
	return progressbar.NewOptions64(max,
		progressbar.OptionSetDescription(description),
		progressbar.OptionSetWidth(30),
		progressbar.OptionShowCount(),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "=",
			SaucerHead:    ">",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
		progressbar.OptionOnCompletion(func() {
			fmt.Println()
		}),
	)
}

// Formatting helpers
func formatSpeed(bytesPerSec float64) string {
	if bytesPerSec >= 1024*1024*1024 {
		return fmt.Sprintf("%.2f GB/s", bytesPerSec/(1024*1024*1024))
	} else if bytesPerSec >= 1024*1024 {
		return fmt.Sprintf("%.2f MB/s", bytesPerSec/(1024*1024))
	} else if bytesPerSec >= 1024 {
		return fmt.Sprintf("%.2f KB/s", bytesPerSec/1024)
	}
	return fmt.Sprintf("%.2f B/s", bytesPerSec)
}

func formatSize(bytes int64) string {
	if bytes >= 1024*1024*1024 {
		return fmt.Sprintf("%.2f GB", float64(bytes)/(1024*1024*1024))
	} else if bytes >= 1024*1024 {
		return fmt.Sprintf("%.2f MB", float64(bytes)/(1024*1024))
	} else if bytes >= 1024 {
		return fmt.Sprintf("%.2f KB", float64(bytes)/1024)
	}
	return fmt.Sprintf("%d B", bytes)
}

func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	} else if d < time.Minute {
		return fmt.Sprintf("%.2fs", d.Seconds())
	}
	return fmt.Sprintf("%.2fm", d.Minutes())
}

func hasFlag(args []string, flag string) bool {
	for _, arg := range args {
		if arg == flag {
			return true
		}
	}
	return false
}

func removeFlags(args []string) []string {
	var clean []string
	for _, arg := range args {
		if !strings.HasPrefix(arg, "--") {
			clean = append(clean, arg)
		}
	}
	return clean
}

func printUsage(execName string) {
	fmt.Printf(`Usage: %s <command> [options]

Commands:
  upload <file> [options]    Upload file to Swarm
  download <cid>             Download file from Swarm (auto-detects encryption)
  help                       Show this help message

Options:
  --encrypt       Force upload with encryption
  --no-encrypt    Force upload without encryption (default: respects config.yaml)
  --help          Show this help message

Examples:
  %s upload myfile.txt                  # Upload (uses config.yaml default)
  %s upload myfile.txt --encrypt        # Force encryption
  %s upload myfile.txt --no-encrypt     # Force no-encryption
  %s download QmXxxx...                 # Download (auto-detects encryption)

`, execName, execName, execName, execName)
}

func main() {
	execName := filepath.Base(os.Args[0])

	// Load configuration
	config, err := finalride.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Convert chunk size from MB to bytes
	chunkSizeBytes := config.ChunkSizeMB * 1024 * 1024

	if len(os.Args) < 2 {
		printUsage(execName)
		return
	}

	action := os.Args[1]

	fmt.Println("========================================")
	fmt.Println("             FINAL RIDE CLI             ")
	fmt.Println("========================================")

	switch action {
	case "upload":
		cleanArgs := removeFlags(os.Args)
		if len(cleanArgs) < 3 {
			fmt.Printf("Usage: %s upload <file> [--no-encrypt]\n", execName)
			return
		}

		noEncrypt := hasFlag(os.Args, "--no-encrypt")
		forceEncrypt := hasFlag(os.Args, "--encrypt")
		
		shouldEncrypt := config.EncryptDefault
		if forceEncrypt { shouldEncrypt = true }
		if noEncrypt { shouldEncrypt = false }

		file := cleanArgs[2]
		totalStart := time.Now()

		fileInfo, err := os.Stat(file)
		if os.IsNotExist(err) {
			log.Fatalf("File does not exist: %s", file)
		}

		fileSize := fileInfo.Size()
		fmt.Println("========================================")
		fmt.Printf("File: %s\n", filepath.Base(file))
		fmt.Printf("Size: %s (%d bytes)\n", formatSize(fileSize), fileSize)
		fmt.Printf("Encryption: %v\n", shouldEncrypt)
		fmt.Println("========================================")

		fmt.Println("\n[1/4] Reading file...")
		readStart := time.Now()
		plaintext, err := os.ReadFile(file)
		if err != nil {
			log.Fatalf("Failed to read file: %v", err)
		}
		readDuration := time.Since(readStart)
		readSpeed := float64(len(plaintext)) / readDuration.Seconds()
		fmt.Printf("      Read complete: %s in %s (%s)\n", formatSize(int64(len(plaintext))), formatDuration(readDuration), formatSpeed(readSpeed))

		metadata := finalride.Metadata{
			Filename:  filepath.Base(file),
			Encrypted: shouldEncrypt,
		}

		var dataToUpload []byte

		if shouldEncrypt {
			encryptionKey, err := finalride.GenerateKey()
			if err != nil {
				log.Fatalf("Failed to generate encryption key: %v", err)
			}

			fmt.Println("\n[2/4] Encrypting file...")
			encryptStart := time.Now()
			dataToUpload, err = finalride.EncryptData(plaintext, encryptionKey)
			if err != nil {
				log.Fatalf("Encryption failed: %v", err)
			}
			encryptDuration := time.Since(encryptStart)
			encryptSpeed := float64(len(plaintext)) / encryptDuration.Seconds()
			fmt.Printf("      Encryption complete: %s in %s (%s)\n", formatSize(int64(len(dataToUpload))), formatDuration(encryptDuration), formatSpeed(encryptSpeed))

			metadata.Key = base64.StdEncoding.EncodeToString(encryptionKey)
		} else {
			fmt.Println("\n[2/4] Skipping encryption (--no-encrypt)")
			dataToUpload = plaintext
		}

		var uploadStart time.Time
		var uploadDuration time.Duration
		var totalUploaded int64

		if len(dataToUpload) > chunkSizeBytes {
			fmt.Printf("\n[3/4] Chunking file (size > %d MB)...\n", config.ChunkSizeMB)
			chunkStart := time.Now()
			chunks, chunkHashes := finalride.SplitIntoChunks(dataToUpload, chunkSizeBytes)
			chunkDuration := time.Since(chunkStart)
			chunkSpeed := float64(len(dataToUpload)) / chunkDuration.Seconds()
			fmt.Printf("      Chunking complete: %d chunks in %s (%s)\n", len(chunks), formatDuration(chunkDuration), formatSpeed(chunkSpeed))

			fmt.Println("\n[4/4] Uploading chunks...")
			uploadStart = time.Now()

			bar := createProgressBar(int64(len(dataToUpload)), "Uploading       ")
			chunkIDs := make(map[string]string)

			for k, chunk := range chunks {
				ref, err := finalride.UploadToSwarm(chunk, config.SwarmAPI)
				if err != nil {
					log.Fatalf("\nFailed to upload chunk %s: %v", k, err)
				}
				chunkIDs[k] = ref
				totalUploaded += int64(len(chunk))
				bar.Add(len(chunk))
			}

			uploadDuration = time.Since(uploadStart)
			metadata.Chunked = true
			metadata.ChunkIDs = chunkIDs
			metadata.ChunkHashes = chunkHashes

		} else {
			fmt.Println("\n[3/4] Skipping chunking (file size <= threshold)")
			fmt.Println("\n[4/4] Uploading file...")

			uploadStart = time.Now()
			bar := createProgressBar(int64(len(dataToUpload)), "Uploading       ")

			fileID, err := finalride.UploadToSwarm(dataToUpload, config.SwarmAPI)
			if err != nil {
				log.Fatalf("\nFailed to upload file: %v", err)
			}
			bar.Add(len(dataToUpload))
			totalUploaded = int64(len(dataToUpload))

			uploadDuration = time.Since(uploadStart)

			hash := sha256.Sum256(dataToUpload)
			metadata.Chunked = false
			metadata.FileID = fileID
			metadata.FileHash = fmt.Sprintf("%x", hash)
		}

		uploadSpeed := float64(totalUploaded) / uploadDuration.Seconds()
		fmt.Printf("      Upload complete: %s in %s (%s)\n", formatSize(totalUploaded), formatDuration(uploadDuration), formatSpeed(uploadSpeed))

		fmt.Println("\n      Uploading metadata...")
		metadataJSON, err := json.MarshalIndent(metadata, "", "  ")
		if err != nil {
			log.Fatalf("Failed to create metadata JSON: %v", err)
		}

		metadataCID, err := finalride.UploadToSwarm(metadataJSON, config.SwarmAPI)
		if err != nil {
			log.Fatalf("Failed to upload metadata: %v", err)
		}

		totalDuration := time.Since(totalStart)
		avgSpeed := float64(fileSize) / totalDuration.Seconds()

		fmt.Println("\n========================================")
		fmt.Println("UPLOAD SUCCESSFUL!")
		fmt.Println("========================================")
		fmt.Printf("Metadata CID: %s\n", metadataCID)
		fmt.Printf("Encrypted: %v\n", metadata.Encrypted)
		fmt.Printf("Chunked: %v\n", metadata.Chunked)
		if metadata.Chunked {
			fmt.Printf("Chunks: %d\n", len(metadata.ChunkIDs))
		}
		fmt.Println("----------------------------------------")
		fmt.Printf("Total time: %s\n", formatDuration(totalDuration))
		fmt.Printf("Average speed: %s\n", formatSpeed(avgSpeed))
		fmt.Println("----------------------------------------")
		fmt.Printf("Shareable Download Link:\n%s\n", fmt.Sprintf(config.DownloadLink, metadataCID))

	case "download":
		if len(os.Args) < 3 {
			fmt.Printf("Usage: %s download <metadata_cid>\n", execName)
			return
		}

		metadataCID := os.Args[2]

		// URL Extraction Logic
		if strings.Contains(metadataCID, "download=") {
			parts := strings.Split(metadataCID, "download=")
			if len(parts) > 1 {
				extracted := strings.Split(parts[1], "&")[0]
				fmt.Printf("Extracted CID from URL: %s\n", extracted)
				metadataCID = extracted
			}
		}

		totalStart := time.Now()

		fmt.Println("========================================")
		fmt.Println("DOWNLOAD STARTED")
		fmt.Println("========================================")
		fmt.Printf("Metadata CID: %s\n", metadataCID)

		fmt.Println("\n[1/4] Downloading metadata...")
		metadataStart := time.Now()
		metadataJSON, err := finalride.DownloadFromSwarm(metadataCID, config.SwarmAPI)
		if err != nil {
			log.Fatalf("Failed to download metadata: %v", err)
		}
		metadataDuration := time.Since(metadataStart)
		fmt.Printf("      Metadata downloaded in %s\n", formatDuration(metadataDuration))

		var metadata finalride.Metadata
		if err := json.Unmarshal(metadataJSON, &metadata); err != nil {
			log.Fatalf("Failed to parse metadata: %v", err)
		}

		fmt.Println("\n----------------------------------------")
		fmt.Println("FILE INFORMATION")
		fmt.Println("----------------------------------------")
		fmt.Printf("Filename:    %s\n", metadata.Filename)
		fmt.Printf("Encrypted:   %v\n", metadata.Encrypted)
		fmt.Printf("Chunked:     %v\n", metadata.Chunked)
		if metadata.Chunked {
			fmt.Printf("Chunks:      %d\n", len(metadata.ChunkIDs))
		}
		fmt.Println("----------------------------------------")

		var downloadedData []byte
		var downloadDuration time.Duration
		var totalDownloaded int64

		if metadata.Chunked {
			fmt.Printf("\n[2/4] Downloading %d chunks...\n", len(metadata.ChunkIDs))
			downloadStart := time.Now()

			downloadedChunks := make(map[string][]byte)
			bar := createCountProgressBar(int64(len(metadata.ChunkIDs)), "Downloading     ")

			for k, reference := range metadata.ChunkIDs {
				chunkData, err := finalride.DownloadFromSwarm(reference, config.SwarmAPI)
				if err != nil {
					log.Fatalf("\nFailed to download chunk %s: %v", k, err)
				}

				hash := sha256.Sum256(chunkData)
				expectedHash := metadata.ChunkHashes[k]
				if expectedHash != fmt.Sprintf("%x", hash) {
					log.Fatalf("\nChunk %s integrity check failed", k)
				}

				downloadedChunks[k] = chunkData
				totalDownloaded += int64(len(chunkData))
				bar.Add(1)
			}

			downloadDuration = time.Since(downloadStart)
			downloadSpeed := float64(totalDownloaded) / downloadDuration.Seconds()
			fmt.Printf("      Download complete: %s in %s (%s)\n", formatSize(totalDownloaded), formatDuration(downloadDuration), formatSpeed(downloadSpeed))

			fmt.Println("\n[3/4] Reassembling chunks...")
			reassembleStart := time.Now()
			downloadedData = finalride.ReassembleChunks(downloadedChunks)
			reassembleDuration := time.Since(reassembleStart)
			reassembleSpeed := float64(len(downloadedData)) / reassembleDuration.Seconds()
			fmt.Printf("      Reassemble complete: %s in %s (%s)\n", formatSize(int64(len(downloadedData))), formatDuration(reassembleDuration), formatSpeed(reassembleSpeed))

		} else {
			fmt.Println("\n[2/4] Downloading file...")
			downloadStart := time.Now()

			downloadedData, err = finalride.DownloadFromSwarm(metadata.FileID, config.SwarmAPI)
			if err != nil {
				log.Fatalf("\nFailed to download file: %v", err)
			}
			totalDownloaded = int64(len(downloadedData))

			downloadDuration = time.Since(downloadStart)
			downloadSpeed := float64(totalDownloaded) / downloadDuration.Seconds()
			fmt.Printf("      Download complete: %s in %s (%s)\n", formatSize(totalDownloaded), formatDuration(downloadDuration), formatSpeed(downloadSpeed))

			hash := sha256.Sum256(downloadedData)
			if metadata.FileHash != fmt.Sprintf("%x", hash) {
				log.Fatalf("File integrity check failed")
			}
			fmt.Println("      Integrity check: PASSED")

			fmt.Println("\n[3/4] Skipping reassembly (single file)")
		}

		var finalData []byte

		if metadata.Encrypted {
			encryptionKey, err := base64.StdEncoding.DecodeString(metadata.Key)
			if err != nil {
				log.Fatalf("Failed to decode encryption key: %v", err)
			}

			fmt.Println("\n[4/4] Decrypting file...")
			decryptStart := time.Now()
			finalData, err = finalride.DecryptData(downloadedData, encryptionKey)
			if err != nil {
				log.Fatalf("Decryption failed: %v", err)
			}
			decryptDuration := time.Since(decryptStart)
			decryptSpeed := float64(len(downloadedData)) / decryptDuration.Seconds()
			fmt.Printf("      Decryption complete: %s in %s (%s)\n", formatSize(int64(len(finalData))), formatDuration(decryptDuration), formatSpeed(decryptSpeed))
		} else {
			fmt.Println("\n[4/4] Skipping decryption (not encrypted)")
			finalData = downloadedData
		}

		fmt.Println("\n      Saving file...")
		writeStart := time.Now()
		outputFile := metadata.Filename
		if err := os.WriteFile(outputFile, finalData, 0644); err != nil {
			log.Fatalf("Failed to save file: %v", err)
		}
		writeDuration := time.Since(writeStart)
		writeSpeed := float64(len(finalData)) / writeDuration.Seconds()
		fmt.Printf("      Save complete: %s in %s (%s)\n", formatSize(int64(len(finalData))), formatDuration(writeDuration), formatSpeed(writeSpeed))

		totalDuration := time.Since(totalStart)
		avgSpeed := float64(len(finalData)) / totalDuration.Seconds()

		fmt.Println("\n========================================")
		fmt.Println("DOWNLOAD SUCCESSFUL!")
		fmt.Println("========================================")
		fmt.Printf("File saved: %s\n", outputFile)
		fmt.Printf("Size: %s\n", formatSize(int64(len(finalData))))
		fmt.Printf("Encrypted: %v\n", metadata.Encrypted)
		fmt.Println("----------------------------------------")
		fmt.Printf("Total time: %s\n", formatDuration(totalDuration))
		fmt.Printf("Average speed: %s\n", formatSpeed(avgSpeed))
		fmt.Println("========================================")

	case "help":
		printUsage(execName)

	default:
		fmt.Printf("Invalid action '%s'. Use '%s help' for usage.\n", action, execName)
	}
}
