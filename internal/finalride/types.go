package finalride

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the structure of the config.yaml file
type Config struct {
	SwarmAPI       string `yaml:"swarm_api"`       // Swarm API endpoint
	DownloadLink   string `yaml:"download_link"`   // Download link template
	ChunkSizeMB    int    `yaml:"chunk_size_mb"`   // Chunk size in MB
	Theme          string `yaml:"theme"`           // UI Theme: "light" or "dark"
	DownloadDir    string `yaml:"download_dir"`    // Default download directory
	EncryptDefault bool   `yaml:"encrypt_default"` // Encrypt by default?
}

// Metadata represents the file metadata stored in Swarm
type Metadata struct {
	Filename    string            `json:"filename"`
	Encrypted   bool              `json:"encrypted"`
	Key         string            `json:"key,omitempty"`          // Encryption key (only if encrypted)
	Chunked     bool              `json:"chunked"`
	FileID      string            `json:"file_id,omitempty"`      // Single file reference (if not chunked)
	ChunkIDs    map[string]string `json:"chunk_ids,omitempty"`    // Chunk references (if chunked)
	ChunkHashes map[string]string `json:"chunk_hashes,omitempty"` // Chunk hashes for integrity
	FileHash    string            `json:"file_hash,omitempty"`    // File hash (if not chunked)
}

// LoadConfig reads and parses the config.yaml file
func LoadConfig(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %v", err)
	}

	return &config, nil
}

// SaveConfig writes the configuration to config.yaml
func SaveConfig(configPath string, config *Config) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %v", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}
	return nil
}
