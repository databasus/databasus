package recovery

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// JournalStore reads and replaces the journal file. A replacement is written to a temporary file,
// synced and renamed over the journal, then the directory is synced, so a crash leaves either the
// old or the new journal.
type JournalStore struct {
	path     string
	lockPath string
}

func NewJournalStore(directory Directory) *JournalStore {
	return &JournalStore{path: directory.getJournalPath(), lockPath: directory.getJournalLockPath()}
}

// Lock holds the journal for one read, change and write, waiting for another request or process
// holding it. The returned function releases it.
func (s *JournalStore) Lock() (func(), error) {
	if err := os.MkdirAll(filepath.Dir(s.lockPath), privateDirectoryMode); err != nil {
		return nil, fmt.Errorf("create the recovery directory: %w", err)
	}

	file, err := os.OpenFile(s.lockPath, os.O_RDWR|os.O_CREATE, privateFileMode)
	if err != nil {
		return nil, fmt.Errorf("open the recovery journal lock: %w", err)
	}

	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX); err != nil {
		_ = file.Close()

		return nil, fmt.Errorf("take the recovery journal lock: %w", err)
	}

	return func() { _ = file.Close() }, nil
}

// Read returns an empty journal before the first recovery.
func (s *JournalStore) Read() (Journal, error) {
	content, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return Journal{
			PublicationGeneration:   0,
			PublishedKeySha256:      "",
			ConsumedStartupArchives: []string{},
			Operation:               nil,
			LastCompleted:           nil,
			LastEnded:               nil,
		}, nil
	}

	if err != nil {
		return Journal{}, fmt.Errorf("read the recovery journal: %w", err)
	}

	var journal Journal
	if err := json.Unmarshal(content, &journal); err != nil {
		return Journal{}, fmt.Errorf("%w: %w", ErrJournalUnreadable, err)
	}

	if journal.ConsumedStartupArchives == nil {
		journal.ConsumedStartupArchives = []string{}
	}

	return journal, nil
}

func (s *JournalStore) Write(journal Journal) error {
	content, err := json.MarshalIndent(journal, "", "  ")
	if err != nil {
		return fmt.Errorf("encode the recovery journal: %w", err)
	}

	return writeFileDurably(s.path, content)
}

// writeFileDurably replaces path with content so that a crash leaves either the old or the new
// file, never a partial one.
func writeFileDurably(path string, content []byte) error {
	temporaryPath := path + ".tmp"

	file, err := os.OpenFile(temporaryPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, privateFileMode)
	if err != nil {
		return fmt.Errorf("create %s: %w", filepath.Base(temporaryPath), err)
	}

	_, writeErr := file.Write(content)
	syncErr := file.Sync()
	closeErr := file.Close()

	if err := errors.Join(writeErr, syncErr, closeErr); err != nil {
		_ = os.Remove(temporaryPath)

		return fmt.Errorf("write %s: %w", filepath.Base(temporaryPath), err)
	}

	if err := os.Rename(temporaryPath, path); err != nil {
		_ = os.Remove(temporaryPath)

		return fmt.Errorf("replace %s: %w", filepath.Base(path), err)
	}

	return syncDirectory(filepath.Dir(path))
}

func syncDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open %s to sync it: %w", path, err)
	}
	defer func() { _ = directory.Close() }()

	if err := directory.Sync(); err != nil {
		return fmt.Errorf("sync %s: %w", path, err)
	}

	return nil
}
