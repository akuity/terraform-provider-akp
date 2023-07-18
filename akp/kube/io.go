package kube

import (
	"fmt"
	"os"
)

func writeFile(bytes []byte) (string, error) {
	f, err := os.CreateTemp("", "")
	if err != nil {
		return "", fmt.Errorf("failed to generate temp file for manifest: %w", err)
	}
	if _, err = f.Write(bytes); err != nil {
		return "", fmt.Errorf("failed to write manifest: %w", err)
	}
	if err = f.Close(); err != nil {
		return "", fmt.Errorf("failed to close manifest: %w", err)
	}
	return f.Name(), nil
}

func deleteFile(path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return
	}
	_ = os.Remove(path)
}
