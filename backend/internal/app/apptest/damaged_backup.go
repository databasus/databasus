package apptest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
)

func (a *App) DamageLocalBackup(tb testing.TB, backupID uuid.UUID, damagedBytes []byte) {
	tb.Helper()
	backups, err := a.app.GetLogicalBackups(tb.Context(), []uuid.UUID{backupID})
	if err != nil {
		tb.Fatalf("read the backup file name: %v", err)
	}
	if len(backups) != 1 || backups[0].FileName == "" {
		tb.Fatal("the backup has no stored file")
	}

	if err := os.WriteFile(
		filepath.Join(a.dataFolder, backups[0].FileName),
		damagedBytes,
		0o600,
	); err != nil {
		tb.Fatalf("damage the stored backup: %v", err)
	}
}

func (a *App) RemoveLocalBackupFile(tb testing.TB, backupID uuid.UUID) {
	tb.Helper()
	backups, err := a.app.GetLogicalBackups(tb.Context(), []uuid.UUID{backupID})
	if err != nil {
		tb.Fatalf("read the backup file name: %v", err)
	}
	if len(backups) != 1 || backups[0].FileName == "" {
		tb.Fatal("the backup has no stored file")
	}
	if err := os.Remove(filepath.Join(a.dataFolder, backups[0].FileName)); err != nil {
		tb.Fatalf("remove the stored backup: %v", err)
	}
}
