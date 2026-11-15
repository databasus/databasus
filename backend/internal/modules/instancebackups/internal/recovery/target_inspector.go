package recovery

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	"databasus-backend/internal/lib/database"
)

const (
	userStateNextStep = "Mounted-file recovery cannot replace an installation that holds user state. " +
		"Use the root-confirmed import in Settings, Databasus backups, or recover into a separate fresh " +
		"installation. To start the current installation, remove RECOVERY_DUMP_PATH and RECOVERY_MASTER_KEY_PATH"
	instanceBackupsConfigTable = "public.instancebackups_configs"
)

// TargetInspector decides whether a startup recovery may replace the current installation: it may
// hold only what automatic initialization creates. The owners of the tables with default rows
// explain those rows through their public boundaries; any other row is user data, and a row
// outside the application's schema is an unknown state.
type TargetInspector struct {
	dataFolder string
	bundle     database.MigrationBundle
}

func NewTargetInspector(dataFolder string, bundle database.MigrationBundle) *TargetInspector {
	return &TargetInspector{dataFolder: dataFolder, bundle: bundle}
}

// Inspect returns a *RefusalError for an ineligible target. It reads db without migrating or
// changing it.
func (i *TargetInspector) Inspect(ctx context.Context, target *databaseRepository, newerSchemaGuidance string) error {
	recordedVersions, err := target.readRecordedVersions(ctx)
	if err != nil {
		return newUnknownStateRefusal(err)
	}

	if err := i.bundle.CheckNotNewer(recordedVersions); err != nil {
		return &RefusalError{
			Reason:      RefusalNewerSchema,
			Explanation: "the current internal database was migrated by a newer Databasus version",
			NextStep:    newerSchemaGuidance,
			Cause:       err,
		}
	}

	usersState, err := target.inspectUsers(ctx)
	if err != nil {
		return newUnknownStateRefusal(err)
	}

	if usersState.HasAccounts {
		return &RefusalError{
			Reason:      RefusalExistingAccounts,
			Explanation: "the current installation has user accounts, a sole root account included",
			NextStep:    userStateNextStep,
			Cause:       nil,
		}
	}

	instanceBackupSettings, err := target.inspectInstanceBackupSettings(ctx)
	if err != nil {
		return newUnknownStateRefusal(err)
	}

	if usersState.HasChangedSettings || (instanceBackupSettings.IsPresent && !instanceBackupSettings.IsDefault) {
		return &RefusalError{
			Reason:      RefusalChangedSettings,
			Explanation: "persisted settings of the current installation were changed from their defaults",
			NextStep:    userStateNextStep,
			Cause:       nil,
		}
	}

	return i.inspectRemainingContents(ctx, target, usersState.AutomaticTables)
}

func (i *TargetInspector) inspectRemainingContents(
	ctx context.Context,
	target *databaseRepository,
	automaticTables []string,
) error {
	populatedTables, err := target.listPopulatedTables(ctx)
	if err != nil {
		return newUnknownStateRefusal(err)
	}

	automaticTables = append(slices.Clone(automaticTables), instanceBackupsConfigTable)

	var userTables, unknownTables []string

	for _, table := range populatedTables {
		switch {
		case slices.Contains(automaticTables, table):
		case strings.HasPrefix(table, "public."):
			userTables = append(userTables, table)
		default:
			unknownTables = append(unknownTables, table)
		}
	}

	if len(unknownTables) > 0 {
		return newUnknownStateRefusal(fmt.Errorf("tables outside the application schema hold rows: %s",
			strings.Join(unknownTables, ", ")))
	}

	if len(userTables) > 0 {
		return newUserDataRefusal(fmt.Sprintf("tables holding user data: %s", strings.Join(userTables, ", ")))
	}

	hasLocalFiles, err := isDirectoryNonEmpty(i.dataFolder)
	if err != nil {
		return newUnknownStateRefusal(err)
	}

	if hasLocalFiles {
		return newUserDataRefusal("the local backup folder " + i.dataFolder + " holds files")
	}

	return nil
}

func newUserDataRefusal(detail string) *RefusalError {
	return &RefusalError{
		Reason:      RefusalUserData,
		Explanation: "the current installation holds user data; " + detail,
		NextStep:    userStateNextStep,
		Cause:       nil,
	}
}

func newUnknownStateRefusal(cause error) *RefusalError {
	return &RefusalError{
		Reason:      RefusalUnknownState,
		Explanation: "the state of the current installation could not be verified as automatically initialized",
		NextStep:    userStateNextStep,
		Cause:       cause,
	}
}

func isDirectoryNonEmpty(path string) (bool, error) {
	directory, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}

	if err != nil {
		return false, fmt.Errorf("open %s: %w", path, err)
	}
	defer func() { _ = directory.Close() }()

	_, err = directory.Readdirnames(1)
	if errors.Is(err, io.EOF) {
		return false, nil
	}

	if err != nil {
		return false, fmt.Errorf("read %s: %w", path, err)
	}

	return true, nil
}
