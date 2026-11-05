package upgrade

import (
	"fmt"
	"io"
	"os"
)

func installAgentBinary(binaryPath string, source io.Reader) error {
	temporaryPath := binaryPath + ".install"
	defer func() { _ = os.Remove(temporaryPath) }()
	binary, err := os.OpenFile(temporaryPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return fmt.Errorf("create replacement agent: %w", err)
	}

	_, copyErr := io.Copy(binary, source)
	closeErr := binary.Close()
	if copyErr != nil {
		return copyErr
	}

	if closeErr != nil {
		return closeErr
	}

	return os.Rename(temporaryPath, binaryPath)
}
