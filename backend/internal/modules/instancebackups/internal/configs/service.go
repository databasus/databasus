package configs

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/audit"
	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/lib/schedule"
	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/modules/storages"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/modules/workspaces"
)

// maxDestinations bounds a selection, so one run never fans out to an unbounded number of uploads.
const maxDestinations = 50

type Deps struct {
	Repositories Repositories
	Transactor   *database.Transactor
	Clock        clock.Source
	AuditLog     *audit.Log
	Users        users.Module
	Storages     storages.Module
	Workspaces   workspaces.Module
	Databases    databases.Module
	// OnSettingsCommitted runs after a settings update committed, so the work it wakes sees the change.
	OnSettingsCommitted func(change SettingsChange)
	// OnLastDestinationRemoved runs in the transaction of the settings update that removed the last
	// destination, so the stopped protection is recorded exactly when the removal commits.
	OnLastDestinationRemoved func(ctx context.Context) error
}

type Service struct {
	configs      *configRepository
	destinations *destinationRepository
	transactor   *database.Transactor
	clock        clock.Source
	auditLog     *audit.Log
	users        users.Module
	storages     storages.Module
	workspaces   workspaces.Module
	databases    databases.Module
	onCommitted  func(change SettingsChange)
	onStopped    func(ctx context.Context) error
}

func NewService(deps Deps) *Service {
	return &Service{
		configs:      deps.Repositories.configs,
		destinations: deps.Repositories.destinations,
		transactor:   deps.Transactor,
		clock:        deps.Clock,
		auditLog:     deps.AuditLog,
		users:        deps.Users,
		storages:     deps.Storages,
		workspaces:   deps.Workspaces,
		databases:    deps.Databases,
		onCommitted:  deps.OnSettingsCommitted,
		onStopped:    deps.OnLastDestinationRemoved,
	}
}

// SettingsChange tells what a committed settings update did to the selection, so the caller can
// request a run or report stopped protection.
type SettingsChange struct {
	Settings             Settings
	AddedStorageIDs      []uuid.UUID
	RemovedStorageIDs    []uuid.UUID
	IsFirstSelection     bool
	IsLastRemoval        bool
	PreviousDestinations []Destination
}

func (s *Service) GetSettings(ctx context.Context, actor users.Actor) (Settings, error) {
	if _, err := s.users.RequireRootAdmin(ctx, actor); err != nil {
		return Settings{}, err
	}

	return s.readSettings(ctx)
}

func (s *Service) GetConfig(ctx context.Context) (Config, error) {
	return s.configs.get(ctx)
}

func (s *Service) GetDestinations(ctx context.Context) ([]Destination, error) {
	return s.destinations.list(ctx)
}

// UpdateSettings replaces the schedule, the retention and the selection in one transaction. It
// locks the settings row before the storages, in ID order, so it serializes with storage deletion.
func (s *Service) UpdateSettings(ctx context.Context, actor users.Actor, settings Settings) (Settings, error) {
	rootAdmin, err := s.users.RequireRootAdmin(ctx, actor)
	if err != nil {
		return Settings{}, err
	}

	if err := validateSettings(settings); err != nil {
		return Settings{}, err
	}

	var change SettingsChange

	err = s.transactor.Transaction(ctx, func(txCtx context.Context) error {
		change, err = s.replaceSettings(txCtx, settings)
		if err != nil {
			return err
		}

		if change.IsLastRemoval {
			if err := s.onStopped(txCtx); err != nil {
				return err
			}
		}

		return s.auditLog.Record(txCtx, audit.Event{
			Author:      rootAdmin.GetAuditAuthor(),
			WorkspaceID: nil,
			Message:     describeChange(change),
		})
	})
	if err != nil {
		return Settings{}, err
	}

	s.onCommitted(change)

	return change.Settings, nil
}

// OnStorageDeleting refuses deleting a selected destination. The deletion holds the storages' rows,
// which a selection also locks, so a selection either committed before this check or waits and
// then finds the storage gone.
func (s *Service) OnStorageDeleting(ctx context.Context, storageIDs []uuid.UUID) error {
	isSelected, err := s.destinations.hasAnyOf(ctx, storageIDs)
	if err != nil {
		return err
	}

	if isSelected {
		return ErrDatabasusBackupInUse
	}

	return nil
}

// MarkProtected records the first successful copy after a selection or a stop.
func (s *Service) MarkProtected(ctx context.Context) error {
	return s.transactor.Transaction(ctx, func(txCtx context.Context) error {
		config, err := s.configs.getForUpdate(txCtx)
		if err != nil {
			return err
		}

		switch config.ProtectionState {
		case ProtectionStateInitializing, ProtectionStateResuming:
			return s.configs.updateProtectionState(txCtx, ProtectionStateProtected)
		case ProtectionStateNotConfigured, ProtectionStateProtected, ProtectionStateStopped:
			return nil
		}

		return nil
	})
}

// RequestRun asks for a Databasus backup at the next production pass.
func (s *Service) RequestRun(ctx context.Context) error {
	return s.configs.updateIsRunRequested(ctx, true)
}

// RecordOutstanding records, for every destination, the earliest scheduled occurrence due by now that
// no successful copy covers. An occurrence already recorded is kept, so later occurrences, polls and
// retries never move a destination's overdue deadline.
func (s *Service) RecordOutstanding(ctx context.Context, now time.Time) error {
	config, err := s.configs.get(ctx)
	if err != nil {
		return err
	}

	return s.recordOutstanding(ctx, config.GetSchedule(), now)
}

// RecordDelivery runs in the caller's transaction, which publishes the copy. It reports false and
// records nothing when the destination was deselected or selected again since the copy was planned,
// so a late upload never marks a destination protected.
func (s *Service) RecordDelivery(ctx context.Context, outcome DeliveryOutcome) (bool, error) {
	_, isCurrent, err := s.destinations.getCurrentForUpdate(ctx, outcome.StorageID, outcome.SelectionGeneration)
	if err != nil || !isCurrent {
		return false, err
	}

	if outcome.LastError != nil {
		return true, s.destinations.recordFailure(ctx, outcome.StorageID, *outcome.LastError)
	}

	if err := s.destinations.recordSuccess(ctx, outcome); err != nil {
		return false, err
	}

	return true, s.MarkProtected(ctx)
}

// TakeRunRequest clears a pending run request and reports whether there was one.
func (s *Service) TakeRunRequest(ctx context.Context) (bool, error) {
	isRequested := false

	err := s.transactor.Transaction(ctx, func(txCtx context.Context) error {
		config, err := s.configs.getForUpdate(txCtx)
		if err != nil {
			return err
		}

		isRequested = config.IsRunRequested
		if !isRequested {
			return nil
		}

		return s.configs.updateIsRunRequested(txCtx, false)
	})

	return isRequested, err
}

func validateSettings(settings Settings) error {
	if err := settings.Schedule.Validate(); err != nil {
		return newInvalidScheduleProblem(err.Error())
	}

	if settings.RetentionDays < 1 || settings.RetentionDays > 3650 {
		return newInvalidRetentionProblem()
	}

	if len(settings.StorageIDs) > maxDestinations {
		return newInvalidStorageIDsProblem(fmt.Sprintf("at most %d storages can be selected", maxDestinations))
	}

	sortedIDs := slices.Clone(settings.StorageIDs)
	slices.SortFunc(sortedIDs, compareIDs)

	if len(slices.Compact(sortedIDs)) != len(settings.StorageIDs) {
		return newInvalidStorageIDsProblem("a storage is selected more than once")
	}

	return nil
}

func diffSelection(current []Destination, selected []uuid.UUID) ([]uuid.UUID, []uuid.UUID) {
	currentIDs := make([]uuid.UUID, 0, len(current))
	for _, destination := range current {
		currentIDs = append(currentIDs, destination.StorageID)
	}

	addedIDs := []uuid.UUID{}
	for _, storageID := range selected {
		if !slices.Contains(currentIDs, storageID) {
			addedIDs = append(addedIDs, storageID)
		}
	}

	removedIDs := []uuid.UUID{}
	for _, storageID := range currentIDs {
		if !slices.Contains(selected, storageID) {
			removedIDs = append(removedIDs, storageID)
		}
	}

	slices.SortFunc(addedIDs, compareIDs)
	slices.SortFunc(removedIDs, compareIDs)

	return addedIDs, removedIDs
}

func toSettings(config Config, destinations []Destination) Settings {
	storageIDs := make([]uuid.UUID, 0, len(destinations))
	for _, destination := range destinations {
		storageIDs = append(storageIDs, destination.StorageID)
	}

	return Settings{Schedule: config.GetSchedule(), RetentionDays: config.RetentionDays, StorageIDs: storageIDs}
}

func describeChange(change SettingsChange) string {
	switch {
	case change.IsLastRemoval:
		return "Databasus backups stopped: the last destination was removed"
	case len(change.AddedStorageIDs) > 0 || len(change.RemovedStorageIDs) > 0:
		return fmt.Sprintf("Databasus backup destinations changed: %d added, %d removed",
			len(change.AddedStorageIDs), len(change.RemovedStorageIDs))
	default:
		return "Databasus backup schedule and retention updated"
	}
}

func compareIDs(left, right uuid.UUID) int {
	return strings.Compare(left.String(), right.String())
}

// AcknowledgeMasterKey records that root admin saved the key whose digest is masterKeySha256.
func (s *Service) AcknowledgeMasterKey(ctx context.Context, actor users.Actor, masterKeySha256 string) error {
	rootAdmin, err := s.users.RequireRootAdmin(ctx, actor)
	if err != nil {
		return err
	}

	now := s.clock.Now()

	return s.transactor.Transaction(ctx, func(txCtx context.Context) error {
		if err := s.configs.updateMasterKeyAcknowledgment(txCtx, &masterKeySha256, &now); err != nil {
			return err
		}

		return s.auditLog.Record(txCtx, audit.Event{
			Author:      rootAdmin.GetAuditAuthor(),
			WorkspaceID: nil,
			Message:     "Root admin confirmed that the master key is saved outside the server",
		})
	})
}

func (s *Service) recordOutstanding(ctx context.Context, currentSchedule schedule.Schedule, now time.Time) error {
	destinations, err := s.destinations.list(ctx)
	if err != nil {
		return err
	}

	for _, destination := range destinations {
		if destination.OutstandingDueAt != nil {
			continue
		}

		dueAt := destination.GetOutstandingDueAt(currentSchedule, now)
		if dueAt == nil {
			continue
		}

		if err := s.destinations.markOutstanding(ctx, destination.StorageID, *dueAt); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) readSettings(ctx context.Context) (Settings, error) {
	config, err := s.configs.get(ctx)
	if err != nil {
		return Settings{}, err
	}

	destinations, err := s.destinations.list(ctx)
	if err != nil {
		return Settings{}, err
	}

	return toSettings(config, destinations), nil
}

func (s *Service) replaceSettings(ctx context.Context, settings Settings) (SettingsChange, error) {
	config, err := s.configs.getForUpdate(ctx)
	if err != nil {
		return SettingsChange{}, err
	}

	lockedStorages, err := s.storages.GetStoragesBlockingChanges(ctx, settings.StorageIDs)
	if err != nil {
		return SettingsChange{}, err
	}

	if len(lockedStorages) != len(settings.StorageIDs) {
		return SettingsChange{}, newInvalidStorageIDsProblem("a selected storage does not exist")
	}

	currentDestinations, err := s.destinations.list(ctx)
	if err != nil {
		return SettingsChange{}, err
	}

	addedStorageIDs, removedStorageIDs := diffSelection(currentDestinations, settings.StorageIDs)

	if err := s.destinations.deleteByStorageIDs(ctx, removedStorageIDs); err != nil {
		return SettingsChange{}, err
	}

	now := s.clock.Now()

	// The occurrences already due under the old schedule stay owed, so a schedule edit never
	// erases or moves an overdue deadline.
	if err := s.recordOutstanding(ctx, config.GetSchedule(), now); err != nil {
		return SettingsChange{}, err
	}
	nextGeneration := config.NextSelectionGeneration

	for _, storageID := range addedStorageIDs {
		err := s.destinations.insert(ctx, Destination{
			StorageID:                storageID,
			SelectionGeneration:      nextGeneration,
			SelectedAt:               now,
			OutstandingDueAt:         &now,
			LastSuccessfulSnapshotAt: nil,
			LastError:                nil,
		})
		if err != nil {
			return SettingsChange{}, err
		}

		nextGeneration++
	}

	if err := s.configs.updateNextSelectionGeneration(ctx, nextGeneration); err != nil {
		return SettingsChange{}, err
	}

	config.ScheduleType = settings.Schedule.Type
	config.ScheduleTimeOfDay = settings.Schedule.TimeOfDay
	config.ScheduleWeekday = settings.Schedule.Weekday
	config.ScheduleDayOfMonth = settings.Schedule.DayOfMonth
	config.ScheduleCronExpression = settings.Schedule.CronExpression
	config.RetentionDays = settings.RetentionDays

	if err := s.configs.updateScheduleAndRetention(ctx, config); err != nil {
		return SettingsChange{}, err
	}

	change := SettingsChange{
		Settings:             Settings{},
		AddedStorageIDs:      addedStorageIDs,
		RemovedStorageIDs:    removedStorageIDs,
		IsFirstSelection:     len(currentDestinations) == 0 && len(settings.StorageIDs) > 0,
		IsLastRemoval:        len(currentDestinations) > 0 && len(settings.StorageIDs) == 0,
		PreviousDestinations: currentDestinations,
	}

	if err := s.applyProtectionTransition(ctx, config, change); err != nil {
		return SettingsChange{}, err
	}

	if len(addedStorageIDs) > 0 {
		if err := s.configs.updateIsRunRequested(ctx, true); err != nil {
			return SettingsChange{}, err
		}
	}

	change.Settings, err = s.readSettings(ctx)
	if err != nil {
		return SettingsChange{}, err
	}

	return change, nil
}

func (s *Service) applyProtectionTransition(ctx context.Context, config Config, change SettingsChange) error {
	nextState := config.ProtectionState

	switch {
	case change.IsLastRemoval:
		nextState = ProtectionStateStopped
	case change.IsFirstSelection && config.ProtectionState == ProtectionStateStopped:
		nextState = ProtectionStateResuming
	case change.IsFirstSelection:
		nextState = ProtectionStateInitializing
	}

	if nextState == config.ProtectionState {
		return nil
	}

	return s.configs.updateProtectionState(ctx, nextState)
}
