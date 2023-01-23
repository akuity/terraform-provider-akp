package kube

import (
	"fmt"
	"os"
)

func writeFile(bytes []byte) (string, error) {
	f, err := os.CreateTemp("", "")
	if err != nil {
		return "", fmt.Errorf("Failed to generate temp file for manifest: %v", err)
	}
	if _, err = f.Write(bytes); err != nil {
		return "", fmt.Errorf("Failed to write manifest: %v", err)
	}
	if err = f.Close(); err != nil {
		return "", fmt.Errorf("Failed to close manifest: %v", err)
	}
	return f.Name(), nil
}

func deleteFile(path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return
	}
	_ = os.Remove(path)
}
