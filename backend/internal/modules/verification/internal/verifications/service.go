package verifications

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/apierror"
	"databasus-backend/internal/lib/audit"
	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/modules/backups"
	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/modules/notifiers"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/modules/verification/internal/agents"
	"databasus-backend/internal/modules/verification/internal/configs"
	"databasus-backend/internal/modules/workspaces"
)

type Deps struct {
	Repositories Repositories
	Agents       *agents.Service
	Configs      *configs.Service
	Transactor   *database.Transactor
	Clock        clock.Source
	AuditLog     *audit.Log
	Users        users.Module
	Workspaces   workspaces.Module
	Databases    databases.Module
	Backups      backups.Module
	Notifiers    notifiers.Module
	Logger       *slog.Logger
}
type Service struct {
	verificationRepository *verificationRepository
	agents                 *agents.Service
	configService          *configs.Service
	transactor             *database.Transactor
	clock                  clock.Source
	auditLog               *audit.Log
	users                  users.Module
	workspaceService       workspaces.Module
	databases              databases.Module
	backups                backups.Module
	notifierService        notifiers.Module
	logger                 *slog.Logger
}

func NewService(deps Deps) *Service {
	return &Service{
		verificationRepository: deps.Repositories.verifications,
		agents:                 deps.Agents,
		configService:          deps.Configs,
		transactor:             deps.Transactor,
		clock:                  deps.Clock,
		auditLog:               deps.AuditLog,
		users:                  deps.Users,
		workspaceService:       deps.Workspaces,
		databases:              deps.Databases,
		backups:                deps.Backups,
		notifierService:        deps.Notifiers,
		logger:                 deps.Logger,
	}
}

type ListQuery struct {
	DatabaseID uuid.UUID
	Limit      *int
	After      *string
}
type Page struct {
	Verifications []*Verification
	NextCursor    *string
}

func (s *Service) StartVerification(ctx context.Context, actor users.Actor, backupID uuid.UUID) (*Verification, error) {
	backup, isFound, err := s.getLogicalBackup(ctx, backupID)
	if err != nil {
		return nil, err
	}
	if !isFound {
		return nil, ErrBackupNotFound
	}

	database, err := s.authorizeDatabase(ctx, actor, backup.DatabaseID, workspaces.ActionEditContent)
	if err != nil {
		return nil, hideNotFound(err, ErrBackupNotFound)
	}

	if backup.Status != backups.LogicalBackupStatusCompleted {
		return nil, ErrBackupNotCompleted
	}

	verification := s.newPending(backup, VerificationTriggerManual)
	err = s.transactor.Transaction(ctx, func(ctx context.Context) error {
		existing, err := s.verificationRepository.GetActiveForDatabase(ctx, backup.DatabaseID)
		if err != nil {
			return err
		}
		for _, verification := range existing {
			if verification.Trigger == VerificationTriggerManual {
				return ErrAlreadyRunning
			}
		}
		if err := s.verificationRepository.Create(ctx, verification); err != nil {
			return err
		}
		return s.record(ctx, actor, database, "manual backup verification enqueued for database: "+database.Name)
	})
	return verification, err
}

func (s *Service) CancelVerification(ctx context.Context, actor users.Actor, id uuid.UUID) (*Verification, error) {
	var canceled *Verification
	err := s.transactor.Transaction(ctx, func(ctx context.Context) error {
		verification, err := s.GetVerification(ctx, actor, id)
		if err != nil {
			return err
		}
		database, err := s.authorizeDatabase(ctx, actor, verification.DatabaseID, workspaces.ActionEditContent)
		if err != nil {
			return hideNotFound(err, ErrVerificationNotFound)
		}
		if verification.Status != VerificationStatusPending && verification.Status != VerificationStatusRunning {
			return ErrNotRunning
		}
		if err := s.verificationRepository.MarkTerminal(
			ctx,
			TerminalUpdate{
				VerificationID: id,
				Status:         VerificationStatusCanceled,
				Fields:         map[string]any{"fail_message": cancelMessageByUser},
				FinishedAt:     s.clock.Now(),
			},
		); err != nil {
			return err
		}
		canceled, _, err = s.verificationRepository.GetVerification(ctx, id)
		if err != nil {
			return err
		}
		return s.record(ctx, actor, database, "backup verification canceled for database: "+database.Name)
	})
	return canceled, err
}

func (s *Service) GetVerification(ctx context.Context, actor users.Actor, id uuid.UUID) (*Verification, error) {
	verification, isFound, err := s.verificationRepository.GetVerification(ctx, id)
	if err != nil {
		return nil, err
	}
	if !isFound {
		return nil, ErrVerificationNotFound
	}
	if _, err := s.authorizeDatabase(ctx, actor, verification.DatabaseID, workspaces.ActionView); err != nil {
		return nil, hideNotFound(err, ErrVerificationNotFound)
	}
	return verification, nil
}

func (s *Service) ListVerifications(ctx context.Context, actor users.Actor, query ListQuery) (Page, error) {
	if _, err := s.authorizeDatabase(ctx, actor, query.DatabaseID, workspaces.ActionView); err != nil {
		return Page{}, err
	}

	limit := 20
	if query.Limit != nil {
		limit = *query.Limit
	}

	if limit < 1 || limit > 100 {
		return Page{}, apierror.NewInvalidFieldsProblem(
			[]apierror.FieldError{{Field: "limit", Detail: "must be between 1 and 100"}},
		)
	}

	var after *database.TimeCursor
	if query.After != nil {
		cursor, err := database.DecodeTimeCursor(*query.After)
		if err != nil {
			return Page{}, err
		}
		after = &cursor
	}

	verifications, err := s.verificationRepository.ListPage(
		ctx,
		VerificationPageQuery{DatabaseID: query.DatabaseID, Limit: limit + 1, After: after},
	)
	if err != nil {
		return Page{}, err
	}

	page := Page{Verifications: verifications[:min(len(verifications), limit)], NextCursor: nil}
	if len(verifications) > limit {
		lastVerification := verifications[limit-1]
		cursor := database.TimeCursor{Time: lastVerification.CreatedAt, ID: lastVerification.ID}.Encode()
		page.NextCursor = &cursor
	}
	return page, nil
}

func (s *Service) authorizeDatabase(
	ctx context.Context,
	actor users.Actor,
	id uuid.UUID,
	action workspaces.Action,
) (databases.Database, error) {
	database, err := s.getDatabase(ctx, id)
	if err != nil {
		return databases.Database{}, err
	}
	if err := s.workspaceService.Authorize(ctx, actor, database.WorkspaceID, action); err != nil {
		return databases.Database{}, hideNotFound(err, ErrDatabaseNotFound)
	}
	return database, nil
}

func (s *Service) record(ctx context.Context, actor users.Actor, database databases.Database, message string) error {
	author, isFound, err := s.users.GetUser(ctx, actor.UserID)
	if err != nil {
		return err
	}
	if !isFound {
		return ErrVerificationNotFound
	}
	return s.auditLog.Record(
		ctx,
		audit.Event{Author: author.GetAuditAuthor(), WorkspaceID: &database.WorkspaceID, Message: message},
	)
}

func (s *Service) newPending(backup backups.LogicalBackup, trigger VerificationTrigger) *Verification {
	return &Verification{
		ID:                        uuid.New(),
		DatabaseID:                backup.DatabaseID,
		BackupID:                  backup.ID,
		AgentID:                   nil,
		Trigger:                   trigger,
		Status:                    VerificationStatusPending,
		AttemptCount:              1,
		CreatedAt:                 s.clock.Now(),
		StartedAt:                 nil,
		FinishedAt:                nil,
		RestoreDurationMs:         nil,
		VerifyDurationMs:          nil,
		PgRestoreExitCode:         nil,
		RestoredDatabaseSizeBytes: nil,
		TableCount:                nil,
		SchemaCount:               nil,
		FailMessage:               nil,
		TableStats:                nil,
	}
}
