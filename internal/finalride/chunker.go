package finalride

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"strconv"
)

// SplitIntoChunks splits data into chunks
func SplitIntoChunks(data []byte, chunkSize int) (map[string][]byte, map[string]string) {
	chunks := make(map[string][]byte)
	hashes := make(map[string]string)
	chunkNum := 1

	for i := 0; i < len(data); i += chunkSize {
		end := i + chunkSize
		if end > len(data) {
			end = len(data)
		}
		chunk := data[i:end]
		chunkKey := fmt.Sprintf("%d", chunkNum)
		chunks[chunkKey] = chunk

		// Compute hash for the chunk
		hash := sha256.Sum256(chunk)
		hashes[chunkKey] = fmt.Sprintf("%x", hash)
		chunkNum++
	}
	return chunks, hashes
}

// ReassembleChunks reassembles chunks in order
func ReassembleChunks(chunks map[string][]byte) []byte {
	keys := make([]int, 0, len(chunks))
	for k := range chunks {
		num, _ := strconv.Atoi(k)
		keys = append(keys, num)
	}
	sort.Ints(keys)

	var result []byte
	for _, k := range keys {
		result = append(result, chunks[strconv.Itoa(k)]...)
	}
	return result
}
