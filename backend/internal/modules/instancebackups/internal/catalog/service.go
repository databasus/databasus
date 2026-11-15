// Package catalog makes Databasus backups and keeps their history: it starts a dump on the
// schedule, on a new destination or on root admin's request, delivers the encrypted file to every
// destination independently, applies retention and serves downloads of stored copies.
package catalog

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/audit"
	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/lib/recoverypause"
	"databasus-backend/internal/modules/instancebackups/internal/configs"
	"databasus-backend/internal/modules/instancebackups/internal/snapshot"
	"databasus-backend/internal/modules/instancebackups/internal/tokens"
	"databasus-backend/internal/modules/storages"
	"databasus-backend/internal/modules/users"
)

type Deps struct {
	Repositories Repositories
	Transactor   *database.Transactor
	Clock        clock.Source
	Logger       *slog.Logger
	AuditLog     *audit.Log
	Users        users.Module
	Storages     storages.Module
	Configs      *configs.Service
	Producer     *snapshot.Producer
	Tokens       *tokens.Service
	Pause        *recoverypause.Gate
	Release      string
}

type Service struct {
	backups          *backupRepository
	copies           *copyRepository
	transactor       *database.Transactor
	clock            clock.Source
	logger           *slog.Logger
	auditLog         *audit.Log
	users            users.Module
	storages         storages.Module
	configs          *configs.Service
	producer         *snapshot.Producer
	tokens           *tokens.Service
	pause            *recoverypause.Gate
	release          string
	productionSignal chan struct{}
	deliverySignal   chan struct{}
}

func NewService(deps Deps) *Service {
	return &Service{
		backups:          deps.Repositories.backups,
		copies:           deps.Repositories.copies,
		transactor:       deps.Transactor,
		clock:            deps.Clock,
		logger:           deps.Logger,
		auditLog:         deps.AuditLog,
		users:            deps.Users,
		storages:         deps.Storages,
		configs:          deps.Configs,
		producer:         deps.Producer,
		tokens:           deps.Tokens,
		pause:            deps.Pause,
		release:          deps.Release,
		productionSignal: make(chan struct{}, 1),
		deliverySignal:   make(chan struct{}, 1),
	}
}

// HistoryQuery is one page request of the history.
type HistoryQuery struct {
	Limit int
	After *string
}

// StartBackup begins a manual backup and returns it while its dump runs. Only one dump runs at a time.
func (s *Service) StartBackup(ctx context.Context, actor users.Actor) (BackupWithCopies, error) {
	rootAdmin, err := s.users.RequireRootAdmin(ctx, actor)
	if err != nil {
		return BackupWithCopies{}, err
	}

	if err := s.pause.Check(); err != nil {
		return BackupWithCopies{}, err
	}

	destinations, err := s.configs.GetDestinations(ctx)
	if err != nil {
		return BackupWithCopies{}, err
	}

	if len(destinations) == 0 {
		return BackupWithCopies{}, ErrNoDestinations
	}

	backup := s.newRunningBackup(TriggerManual)

	err = s.transactor.Transaction(ctx, func(txCtx context.Context) error {
		isStarted, err := s.backups.insertRunning(txCtx, backup)
		if err != nil {
			return err
		}

		if !isStarted {
			return ErrAlreadyRunning
		}

		return s.auditLog.Record(txCtx, audit.Event{
			Author:      rootAdmin.GetAuditAuthor(),
			WorkspaceID: nil,
			Message:     "Root admin started a Databasus backup",
		})
	})
	if err != nil {
		return BackupWithCopies{}, err
	}

	s.WakeProduction()

	return BackupWithCopies{Backup: backup, Copies: []Copy{}}, nil
}

func (s *Service) GetBackup(ctx context.Context, actor users.Actor, backupID uuid.UUID) (BackupWithCopies, error) {
	if _, err := s.users.RequireRootAdmin(ctx, actor); err != nil {
		return BackupWithCopies{}, err
	}

	backup, isFound, err := s.backups.get(ctx, backupID)
	if err != nil {
		return BackupWithCopies{}, err
	}

	if !isFound {
		return BackupWithCopies{}, ErrBackupNotFound
	}

	withCopies, err := s.attachCopies(ctx, []Backup{backup})
	if err != nil {
		return BackupWithCopies{}, err
	}

	return withCopies[0], nil
}

func (s *Service) ListBackups(ctx context.Context, actor users.Actor, query HistoryQuery) (BackupPage, error) {
	if _, err := s.users.RequireRootAdmin(ctx, actor); err != nil {
		return BackupPage{}, err
	}

	var after *database.TimeCursor

	if query.After != nil {
		cursor, err := database.DecodeTimeCursor(*query.After)
		if err != nil {
			return BackupPage{}, newInvalidCursorProblem()
		}

		after = &cursor
	}

	backups, err := s.backups.listPage(ctx, after, query.Limit+1)
	if err != nil {
		return BackupPage{}, err
	}

	var nextCursor *string

	if len(backups) > query.Limit {
		backups = backups[:query.Limit]
		last := backups[len(backups)-1]
		encoded := database.TimeCursor{Time: getHistoryTime(last), ID: last.ID}.Encode()
		nextCursor = &encoded
	}

	items, err := s.attachCopies(ctx, backups)
	if err != nil {
		return BackupPage{}, err
	}

	return BackupPage{Items: items, NextCursor: nextCursor}, nil
}

// OnStorageDeleting keeps the history of copies in storages a deletion removes; their files go
// with the storage, so the copies become unavailable. It runs in the deletion's transaction after
// the selection check refused deleting a current destination.
func (s *Service) OnStorageDeleting(ctx context.Context, storageIDs []uuid.UUID) error {
	return s.copies.detachStorages(ctx, storageIDs)
}

func (s *Service) GetProductionSignal() <-chan struct{} { return s.productionSignal }

func (s *Service) GetDeliverySignal() <-chan struct{} { return s.deliverySignal }

// WakeProduction asks the production job for a pass, such as after a settings change requested a
// run.
func (s *Service) WakeProduction() {
	wake(s.productionSignal)
}

func (s *Service) newRunningBackup(trigger Trigger) Backup {
	return Backup{
		ID:             uuid.New(),
		Status:         StatusInProgress,
		DumpStatus:     DumpStatusRunning,
		Trigger:        trigger,
		SourceRelease:  s.release,
		StartedAt:      s.clock.Now(),
		SnapshotAt:     nil,
		CompletedAt:    nil,
		SizeBytes:      nil,
		FailureReason:  nil,
		SpoolPath:      nil,
		SpoolExpiresAt: nil,
	}
}

func (s *Service) attachCopies(ctx context.Context, backups []Backup) ([]BackupWithCopies, error) {
	backupIDs := make([]uuid.UUID, 0, len(backups))
	for _, backup := range backups {
		backupIDs = append(backupIDs, backup.ID)
	}

	copies, err := s.copies.listByBackupIDs(ctx, backupIDs)
	if err != nil {
		return nil, err
	}

	copiesByBackupID := make(map[uuid.UUID][]Copy, len(backups))
	for _, backupCopy := range copies {
		copiesByBackupID[backupCopy.BackupID] = append(copiesByBackupID[backupCopy.BackupID], backupCopy)
	}

	items := make([]BackupWithCopies, 0, len(backups))
	for _, backup := range backups {
		backupCopies := copiesByBackupID[backup.ID]
		if backupCopies == nil {
			backupCopies = []Copy{}
		}

		items = append(items, BackupWithCopies{Backup: backup, Copies: backupCopies})
	}

	return items, nil
}

func getHistoryTime(backup Backup) time.Time {
	if backup.SnapshotAt != nil {
		return *backup.SnapshotAt
	}

	return backup.StartedAt
}

func wake(signal chan struct{}) {
	select {
	case signal <- struct{}{}:
	default:
	}
}
