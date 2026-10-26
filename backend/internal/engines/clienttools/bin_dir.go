package clienttools

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// CheckBinDir reports a missing binDir as a single error, and otherwise every required command
// that is missing or lacks the executable bit.
func CheckBinDir(binDir string, requiredCommands []string) []error {
	if _, err := os.Stat(binDir); errors.Is(err, fs.ErrNotExist) {
		return []error{fmt.Errorf("client tools bin directory not found: %s", binDir)}
	}

	var commandErrors []error

	for _, command := range requiredCommands {
		commandPath := filepath.Join(binDir, command)

		info, err := os.Stat(commandPath)
		if errors.Is(err, fs.ErrNotExist) {
			commandErrors = append(commandErrors, fmt.Errorf("client command not found: %s", commandPath))

			continue
		}

		if err != nil {
			commandErrors = append(commandErrors, fmt.Errorf("cannot stat %s: %w", commandPath, err))

			continue
		}

		if info.Mode().Perm()&0o111 == 0 {
			commandErrors = append(commandErrors, fmt.Errorf("client command not executable: %s", commandPath))
		}
	}

	return commandErrors
}
