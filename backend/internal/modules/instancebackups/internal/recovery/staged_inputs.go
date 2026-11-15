package recovery

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/google/uuid"

	"databasus-backend/internal/engines/postgresql/internaldb"
)

// hashFile returns the lowercase hexadecimal SHA-256 of a whole file: the fingerprint that
// identifies a Databasus backup file for startup consumption.
func hashFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = file.Close() }()

	digest := sha256.New()
	if _, err := io.Copy(digest, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(digest.Sum(nil)), nil
}

// copyFileDurably copies source to a new private file and returns the copy's fingerprint, so the
// recovery reads an immutable copy whose bytes are the ones it hashed.
func copyFileDurably(sourcePath, targetPath string) (string, error) {
	source, err := os.Open(sourcePath)
	if err != nil {
		return "", err
	}
	defer func() { _ = source.Close() }()

	target, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, privateFileMode)
	if err != nil {
		return "", fmt.Errorf("create the staged copy: %w", err)
	}

	digest := sha256.New()
	_, copyErr := io.Copy(io.MultiWriter(target, digest), source)
	syncErr := target.Sync()
	closeErr := target.Close()

	if err := errors.Join(copyErr, syncErr, closeErr); err != nil {
		return "", fmt.Errorf("write the staged copy: %w", err)
	}

	return hex.EncodeToString(digest.Sum(nil)), nil
}

func hashBytes(content []byte) string {
	digest := sha256.Sum256(content)

	return hex.EncodeToString(digest[:])
}

// getDatabaseNames names the staging and replaced databases of one operation. The scope derived
// from the active database's name tells the recovery databases of installations sharing one
// server apart.
func getDatabaseNames(activeDatabase string, operationID uuid.UUID) (string, string) {
	suffix := getDatabaseScope(activeDatabase) + "_" + strings.ReplaceAll(operationID.String(), "-", "")[:16]

	return internaldb.StagingPrefix + suffix, internaldb.PreviousPrefix + suffix
}

func getDatabaseScope(activeDatabase string) string {
	return hashBytes([]byte(activeDatabase))[:8]
}
