package apptest

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/testwait"
)

func SuspendPhysicalBackupProcess(t *testing.T, backupID uuid.UUID) func() {
	t.Helper()
	var resume func()
	testwait.Until(t, "the backup's pg_basebackup process starts", func(context.Context) error {
		processes, err := os.ReadDir("/proc")
		if err != nil {
			return err
		}

		for _, entry := range processes {
			processID, err := strconv.Atoi(entry.Name())
			if err != nil {
				continue
			}
			arguments, err := os.ReadFile(filepath.Join("/proc", entry.Name(), "cmdline"))
			if err != nil || !strings.Contains(string(arguments), "pg_basebackup") ||
				!strings.Contains(string(arguments), backupID.String()) {
				continue
			}

			if err := syscall.Kill(processID, syscall.SIGSTOP); err != nil {
				return err
			}

			resume = func() {
				currentArguments, err := os.ReadFile(filepath.Join("/proc", entry.Name(), "cmdline"))
				if err == nil && string(currentArguments) == string(arguments) {
					_ = syscall.Kill(processID, syscall.SIGCONT)
				}
			}
			return nil
		}
		return fmt.Errorf("pg_basebackup for backup %s has not started", backupID)
	})
	t.Cleanup(resume)
	return resume
}
