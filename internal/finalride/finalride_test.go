package finalride

import (
	"bytes"
	"os"
	"testing"
)

func TestEncryptionDecryption(t *testing.T) {
	plaintext := []byte("Hello, Swarm! This is a secret message.")
	key, err := GenerateKey()
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	encrypted, err := EncryptData(plaintext, key)
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}

	if bytes.Equal(plaintext, encrypted) {
		t.Fatal("Encrypted data is identical to plaintext")
	}

	decrypted, err := DecryptData(encrypted, key)
	if err != nil {
		t.Fatalf("Decryption failed: %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Fatalf("Decrypted data does not match plaintext. Got %s, want %s", string(decrypted), string(plaintext))
	}
}

func TestChunkingReassembly(t *testing.T) {
	data := make([]byte, 1024*1024*25) // 25MB
	for i := range data {
		data[i] = byte(i % 256)
	}

	chunkSize := 1024 * 1024 * 10 // 10MB
	chunks, hashes := SplitIntoChunks(data, chunkSize)

	if len(chunks) != 3 {
		t.Fatalf("Expected 3 chunks, got %d", len(chunks))
	}

	if len(hashes) != 3 {
		t.Fatalf("Expected 3 hashes, got %d", len(hashes))
	}

	reassembled := ReassembleChunks(chunks)

	if !bytes.Equal(data, reassembled) {
		t.Fatal("Reassembled data does not match original data")
	}
}

func TestConfigLoadSave(t *testing.T) {
	tempFile := "test_config.yaml"
	defer os.Remove(tempFile)

	originalConfig := &Config{
		SwarmAPI:       "http://localhost:1633",
		WebURL:         "http://localhost:8080",
		DownloadLink:   "http://localhost:8080?download=%s",
		ChunkSizeMB:    10,
		Theme:          "dark",
		DownloadDir:    "./test_downloads",
		EncryptDefault: true,
	}

	if err := SaveConfig(tempFile, originalConfig); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	loadedConfig, err := LoadConfig(tempFile)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if loadedConfig.SwarmAPI != originalConfig.SwarmAPI {
		t.Errorf("SwarmAPI mismatch. Got %s, want %s", loadedConfig.SwarmAPI, originalConfig.SwarmAPI)
	}
	if loadedConfig.WebURL != originalConfig.WebURL {
		t.Errorf("WebURL mismatch. Got %s, want %s", loadedConfig.WebURL, originalConfig.WebURL)
	}
	if loadedConfig.ChunkSizeMB != originalConfig.ChunkSizeMB {
		t.Errorf("ChunkSizeMB mismatch. Got %d, want %d", loadedConfig.ChunkSizeMB, originalConfig.ChunkSizeMB)
	}
	if loadedConfig.EncryptDefault != originalConfig.EncryptDefault {
		t.Errorf("EncryptDefault mismatch. Got %v, want %v", loadedConfig.EncryptDefault, originalConfig.EncryptDefault)
	}
}
