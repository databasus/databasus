package features

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"databasus-backend/internal/features/audit_logs"
	backuping_logical "databasus-backend/internal/features/backups/backups/backuping/logical"
	backuping_physical "databasus-backend/internal/features/backups/backups/backuping/physical"
	backups_services "databasus-backend/internal/features/backups/backups/services"
	backups_config_logical "databasus-backend/internal/features/backups/config/logical"
	backups_config_physical "databasus-backend/internal/features/backups/config/physical"
	"databasus-backend/internal/features/databases"
	healthcheck_config "databasus-backend/internal/features/healthcheck/config"
	"databasus-backend/internal/features/notifiers"
	"databasus-backend/internal/features/restores"
	"databasus-backend/internal/features/restores/restoring"
	"databasus-backend/internal/features/storages"
	task_cancellation "databasus-backend/internal/features/tasks/cancellation"
)

// Test_SetupDependencies_CalledTwice_LogsWarning verifies SetupDependencies is idempotent
func Test_SetupDependencies_CalledTwice_LogsWarning(t *testing.T) {
	// Call each SetupDependencies twice - should not panic, only log warnings
	audit_logs.SetupDependencies()
	audit_logs.SetupDependencies()

	backups_services.SetupDependencies()
	backups_services.SetupDependencies()

	backups_config_logical.SetupDependencies()
	backups_config_logical.SetupDependencies()

	backups_config_physical.SetupDependencies()
	backups_config_physical.SetupDependencies()

	backuping_physical.SetupDependencies()
	backuping_physical.SetupDependencies()

	databases.SetupDependencies()
	databases.SetupDependencies()

	healthcheck_config.SetupDependencies()
	healthcheck_config.SetupDependencies()

	notifiers.SetupDependencies()
	notifiers.SetupDependencies()

	restores.SetupDependencies()
	restores.SetupDependencies()

	storages.SetupDependencies()
	storages.SetupDependencies()

	task_cancellation.SetupDependencies()
	task_cancellation.SetupDependencies()

	// If we reach here without panic, test passes
	t.Log("All SetupDependencies calls completed successfully (idempotent)")
}

// Test_SetupDependencies_ConcurrentCalls_Safe verifies thread safety
func Test_SetupDependencies_ConcurrentCalls_Safe(t *testing.T) {
	var wg sync.WaitGroup

	// Call SetupDependencies concurrently from 10 goroutines
	for range 10 {
		wg.Go(func() {
			audit_logs.SetupDependencies()
		})
	}

	wg.Wait()
	t.Log("Concurrent SetupDependencies calls completed successfully")
}

// Test_BackgroundService_Run_CalledTwice_Panics verifies Run() panics on duplicate calls
func Test_BackgroundService_Run_CalledTwice_Panics(t *testing.T) {
	ctx := t.Context()

	// Create a test background service
	backgroundService := audit_logs.GetAuditLogBackgroundService()

	// Start first Run() in goroutine
	go func() {
		backgroundService.Run(ctx)
	}()

	// Give first call time to initialize
	time.Sleep(100 * time.Millisecond)

	// Second call should panic
	defer func() {
		if r := recover(); r != nil {
			expectedMsg := "*audit_logs.AuditLogBackgroundService.Run() called multiple times"
			panicMsg := fmt.Sprintf("%v", r)
			if panicMsg == expectedMsg {
				t.Logf("Successfully caught panic: %v", r)
			} else {
				t.Errorf("Expected panic message '%s', got '%s'", expectedMsg, panicMsg)
			}
		} else {
			t.Error("Expected panic on second Run() call, but did not panic")
		}
	}()

	backgroundService.Run(ctx)
}

// Test_BackupsScheduler_Run_CalledTwice_Panics verifies scheduler panics on duplicate calls
func Test_BackupsScheduler_Run_CalledTwice_Panics(t *testing.T) {
	ctx := t.Context()

	scheduler := backuping_logical.GetBackupsScheduler()

	// Start first Run() in goroutine
	go func() {
		scheduler.Run(ctx)
	}()

	// Give first call time to initialize
	time.Sleep(100 * time.Millisecond)

	// Second call should panic
	defer func() {
		if r := recover(); r != nil {
			t.Logf("Successfully caught panic: %v", r)
		} else {
			t.Error("Expected panic on second Run() call, but did not panic")
		}
	}()

	scheduler.Run(ctx)
}

// Test_RestoresScheduler_Run_CalledTwice_Panics verifies restore scheduler panics on duplicate calls
func Test_RestoresScheduler_Run_CalledTwice_Panics(t *testing.T) {
	ctx := t.Context()

	scheduler := restoring.GetRestoresScheduler()

	// Start first Run() in goroutine
	go func() {
		scheduler.Run(ctx)
	}()

	// Give first call time to initialize
	time.Sleep(100 * time.Millisecond)

	// Second call should panic
	defer func() {
		if r := recover(); r != nil {
			t.Logf("Successfully caught panic: %v", r)
		} else {
			t.Error("Expected panic on second Run() call, but did not panic")
		}
	}()

	scheduler.Run(ctx)
}
