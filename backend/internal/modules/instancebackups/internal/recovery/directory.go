// Package recovery replaces the internal database and the master key with a recovered Databasus
// backup: the lifecycle lock every process holds, the journal that makes publication resumable,
// the checks of a startup recovery target and the publication itself.
package recovery

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

const (
	privateDirectoryMode = 0o700
	privateFileMode      = 0o600
)

// Directory is the private recovery directory in the data volume. It holds the lifecycle lock, the
// journal and the staged copies of one operation's archive and key, outside the temporary folder
// whose cleanup runs at every start.
type Directory struct {
	root string
}

func NewDirectory(root string) Directory {
	return Directory{root: root}
}

// Ensure creates the directory and narrows its mode, in case an operator created it wider.
func (d Directory) Ensure() error {
	if err := os.MkdirAll(d.root, privateDirectoryMode); err != nil {
		return fmt.Errorf("create the recovery directory: %w", err)
	}

	if err := os.Chmod(d.root, privateDirectoryMode); err != nil {
		return fmt.Errorf("restrict the recovery directory: %w", err)
	}

	return nil
}

func (d Directory) getLockPath() string {
	return filepath.Join(d.root, "lifecycle.lock")
}

func (d Directory) getJournalPath() string {
	return filepath.Join(d.root, "journal.json")
}

func (d Directory) getJournalLockPath() string {
	return filepath.Join(d.root, "journal.lock")
}

func (d Directory) getOperationPath(operationID uuid.UUID) string {
	return filepath.Join(d.root, "operations", operationID.String())
}

func (d Directory) getStagedArchivePath(operationID uuid.UUID) string {
	return filepath.Join(d.getOperationPath(operationID), "archive.dbbackup")
}

func (d Directory) getStagedKeyPath(operationID uuid.UUID) string {
	return filepath.Join(d.getOperationPath(operationID), "master.key")
}

func (d Directory) getPreviousKeyPath(operationID uuid.UUID) string {
	return filepath.Join(d.getOperationPath(operationID), "previous.key")
}

func (d Directory) createOperationDirectory(operationID uuid.UUID) error {
	if err := os.MkdirAll(d.getOperationPath(operationID), privateDirectoryMode); err != nil {
		return fmt.Errorf("create the recovery operation directory: %w", err)
	}

	return nil
}

func (d Directory) removeOperationDirectory(operationID uuid.UUID) error {
	if err := os.RemoveAll(d.getOperationPath(operationID)); err != nil {
		return fmt.Errorf("remove the recovery operation directory: %w", err)
	}

	return nil
}
