package finalride

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// UploadToSwarm uploads data to Ethereum Swarm and returns its reference
func UploadToSwarm(data []byte, apiEndpoint string) (string, error) {
	resp, err := http.Post(apiEndpoint+"/bzz", "application/octet-stream", bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to upload to Swarm: %s - %s", resp.Status, string(body))
	}

	var response struct {
		Reference string `json:"reference"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", err
	}

	return response.Reference, nil
}

// DownloadFromSwarm downloads data from Ethereum Swarm using its reference
func DownloadFromSwarm(reference string, apiEndpoint string) ([]byte, error) {
	resp, err := http.Get(fmt.Sprintf("%s/bzz/%s", apiEndpoint, reference))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to download from Swarm: %s - %s", resp.Status, string(body))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return data, nil
}
