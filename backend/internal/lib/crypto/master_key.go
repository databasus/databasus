// Package crypto holds the master key of the instance and the encryption of secrets with it.
package crypto

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

const (
	minMasterKeyBytes    = 32
	masterKeyFileMode    = 0o600
	masterKeyDirMode     = 0o755
	temporaryKeyFileName = ".secret.key-*"
)

type MasterKey struct {
	secret string
}

func (k *MasterKey) GetSecret() string {
	return k.secret
}

// OpenOrCreateMasterKey reads the key from path, or creates it there when the file is missing. Several
// processes that create it at once end up with one key: each links a fully written temporary file
// into place, and every one that loses the race reads the winner's file.
func OpenOrCreateMasterKey(path string) (*MasterKey, error) {
	masterKey, err := readMasterKey(path)
	if err == nil {
		return masterKey, nil
	}

	if !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}

	err = createMasterKeyFile(path)
	if err != nil && !errors.Is(err, fs.ErrExist) {
		return nil, err
	}

	return readMasterKey(path)
}

func readMasterKey(path string) (*MasterKey, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read the master key file %s: %w", path, err)
	}

	if len(content) < minMasterKeyBytes {
		return nil, fmt.Errorf("check the master key file %s: %w", path, ErrMasterKeyTooShort)
	}

	return &MasterKey{secret: string(content)}, nil
}

func createMasterKeyFile(path string) error {
	temporaryPath, err := writeTemporaryKeyFile(path, uuid.NewString()+uuid.NewString())
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(temporaryPath) }()

	err = os.Link(temporaryPath, path)
	if err != nil {
		return fmt.Errorf("create the master key file %s: %w", path, err)
	}

	return nil
}

// writeTemporaryKeyFile writes the key into a new file next to path and returns that file's path, so
// the key reaches path only whole: by a link or a rename that never exposes a partial file.
func writeTemporaryKeyFile(path, secret string) (string, error) {
	directory := filepath.Dir(path)

	err := os.MkdirAll(directory, masterKeyDirMode)
	if err != nil {
		return "", fmt.Errorf("create the directory of the master key file %s: %w", path, err)
	}

	temporaryFile, err := os.CreateTemp(directory, temporaryKeyFileName)
	if err != nil {
		return "", fmt.Errorf("create a temporary file for the master key file %s: %w", path, err)
	}

	temporaryPath := temporaryFile.Name()

	err = writeAndCloseKeyFile(temporaryFile, secret)
	if err != nil {
		_ = os.Remove(temporaryPath)

		return "", fmt.Errorf("write the master key file %s: %w", path, err)
	}

	return temporaryPath, nil
}

func writeAndCloseKeyFile(file *os.File, secret string) error {
	err := file.Chmod(masterKeyFileMode)
	if err == nil {
		_, err = file.WriteString(secret)
	}

	if err == nil {
		err = file.Sync()
	}

	return errors.Join(err, file.Close())
}
