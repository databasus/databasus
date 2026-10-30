package app

import (
	"fmt"
	"os"
	"path/filepath"
)

func cleanFolder(folder string) error {
	if _, err := os.Stat(folder); os.IsNotExist(err) {
		return nil
	}

	entries, err := os.ReadDir(folder)
	if err != nil {
		return fmt.Errorf("failed to read directory %s: %w", folder, err)
	}

	for _, entry := range entries {
		itemPath := filepath.Join(folder, entry.Name())
		if err := os.RemoveAll(itemPath); err != nil {
			return fmt.Errorf("failed to remove %s: %w", itemPath, err)
		}
	}

	return nil
}
