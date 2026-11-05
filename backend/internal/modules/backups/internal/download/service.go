package download

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/background"
	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/modules/backups/internal/auditing"
	"databasus-backend/internal/modules/backups/internal/logical"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/modules/workspaces"
)

const (
	TokenCleanupJobName  = "backups_download_tokens_cleanup"
	tokenCleanupInterval = time.Minute
)

type Deps struct {
	Repositories   Repositories
	Transactor     *database.Transactor
	Logger         *slog.Logger
	Clock          clock.Source
	LogicalBackups *logical.Service
	Streams        *UserStreams
	// Heartbeats runs the heartbeat of each running download, so the shutdown waits for it.
	Heartbeats *background.Group
	Audit      auditing.Recorder
}

// Service issues download tokens and opens the downloads they allow.
type Service struct {
	repository     *tokenRepository
	transactor     *database.Transactor
	logger         *slog.Logger
	clock          clock.Source
	logicalBackups *logical.Service
	streams        *UserStreams
	heartbeats     *background.Group
	audit          auditing.Recorder
}

func NewService(deps Deps) *Service {
	return &Service{
		repository:     deps.Repositories.tokens,
		transactor:     deps.Transactor,
		logger:         deps.Logger,
		clock:          deps.Clock,
		logicalBackups: deps.LogicalBackups,
		streams:        deps.Streams,
		heartbeats:     deps.Heartbeats,
		audit:          deps.Audit,
	}
}

// Download is an opened backup file. Closing File ends the user's stream.
type Download struct {
	File      io.ReadCloser
	FileName  string
	SizeBytes int64
}

// CreateToken issues a token that lets one request download the backup once within five minutes. A
// user whose stream runs gets no token, so the second download cannot start behind the first.
func (s *Service) CreateToken(ctx context.Context, actor users.Actor, backupID uuid.UUID) (IssuedToken, error) {
	backup, backupDatabase, err := s.logicalBackups.GetAuthorizedBackup(ctx, actor, backupID, workspaces.ActionView)
	if err != nil {
		return IssuedToken{}, err
	}

	isStreamHeld, err := s.streams.IsHeld(ctx, actor.UserID)
	if err != nil {
		return IssuedToken{}, err
	}

	if isStreamHeld {
		return IssuedToken{}, ErrUserStreamHeld
	}

	fileName, err := s.logicalBackups.GetDownloadFileName(backup, backupDatabase)
	if err != nil {
		return IssuedToken{}, err
	}

	now := s.clock.Now().UTC()
	downloadToken := DownloadToken{
		ID:        uuid.New(),
		Token:     generateTokenText(),
		BackupID:  backupID,
		UserID:    actor.UserID,
		ExpiresAt: now.Add(tokenLifetime),
		IsUsed:    false,
		CreatedAt: now,
	}

	err = s.transactor.Transaction(ctx, func(ctx context.Context) error {
		err := s.repository.insert(ctx, downloadToken)
		if err != nil {
			return err
		}

		return s.audit.Record(ctx, actor, backupDatabase.WorkspaceID,
			fmt.Sprintf("Download token generated for backup of database: %s", backupDatabase.Name))
	})
	if err != nil {
		return IssuedToken{}, err
	}

	return IssuedToken{Token: downloadToken.Token, FileName: fileName, ExpiresAt: downloadToken.ExpiresAt}, nil
}

// OpenDownload takes the user stream before it consumes the token: a request refused because another
// stream runs leaves the token usable, and of two requests with one token only the one that consumes
// it streams.
func (s *Service) OpenDownload(ctx context.Context, backupID uuid.UUID, token string) (Download, error) {
	downloadToken, isFound, err := s.repository.getUsable(ctx, token, backupID, s.clock.Now().UTC())
	if err != nil {
		return Download{}, err
	}

	if !isFound {
		return Download{}, ErrTokenNotUsable
	}

	userStream, err := s.streams.Acquire(ctx, downloadToken.UserID)
	if err != nil {
		return Download{}, err
	}

	download, err := s.openWithStream(ctx, downloadToken)
	if err != nil {
		s.releaseStream(ctx, userStream)

		return Download{}, err
	}

	heartbeatCtx, stopHeartbeat := context.WithCancel(context.WithoutCancel(ctx))
	s.heartbeats.Go(heartbeatCtx, "user stream heartbeat", func(heartbeatCtx context.Context) error {
		s.keepStreamAlive(heartbeatCtx, userStream)

		return nil
	})

	download.File = &streamedFile{
		ReadCloser: download.File,
		closeOnce:  sync.Once{},
		onClose: func() {
			stopHeartbeat()
			s.releaseStream(context.WithoutCancel(ctx), userStream)
		},
	}

	return download, nil
}

func (s *Service) GetTokenCleanupJob() background.Job {
	return background.Job{Name: TokenCleanupJobName, Interval: tokenCleanupInterval, Run: s.deleteExpiredTokens}
}

func (s *Service) openWithStream(ctx context.Context, downloadToken DownloadToken) (Download, error) {
	isConsumed, err := s.repository.consume(ctx, downloadToken, s.clock.Now().UTC())
	if err != nil {
		return Download{}, err
	}

	if !isConsumed {
		return Download{}, ErrTokenNotUsable
	}

	backup, backupDatabase, err := s.logicalBackups.GetBackupWithDatabase(ctx, downloadToken.BackupID)
	if err != nil {
		return Download{}, err
	}

	fileName, err := s.logicalBackups.GetDownloadFileName(backup, backupDatabase)
	if err != nil {
		return Download{}, err
	}

	backupFile, err := s.logicalBackups.OpenBackupFile(ctx, backup)
	if err != nil {
		return Download{}, err
	}

	err = s.audit.RecordByUser(ctx, downloadToken.UserID, backupDatabase.WorkspaceID,
		fmt.Sprintf("Backup file downloaded for database: %s", backupDatabase.Name))
	if err != nil {
		_ = backupFile.Close()

		return Download{}, err
	}

	return Download{
		File:      backupFile,
		FileName:  fileName,
		SizeBytes: int64(backup.BackupSizeMb * 1024 * 1024),
	}, nil
}

func (s *Service) keepStreamAlive(heartbeatCtx context.Context, userStream UserStream) {
	ticker := time.NewTicker(UserStreamHeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-heartbeatCtx.Done():
			return
		case <-ticker.C:
			err := userStream.Refresh(heartbeatCtx)
			if err != nil && heartbeatCtx.Err() == nil {
				s.logger.ErrorContext(heartbeatCtx, "failed to refresh the user stream", slog.Any("error", err))
			}
		}
	}
}

func (s *Service) releaseStream(ctx context.Context, userStream UserStream) {
	err := userStream.Release(ctx)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to release the user stream", slog.Any("error", err))
	}
}

func (s *Service) deleteExpiredTokens(ctx context.Context, now time.Time) error {
	deletedCount, err := s.repository.deleteExpired(ctx, now.UTC())
	if err != nil {
		return err
	}

	if deletedCount > 0 {
		s.logger.DebugContext(ctx, "expired download tokens deleted", slog.Int64("count", deletedCount))
	}

	return nil
}

// streamedFile ends the user stream once, when the download's reader is closed.
type streamedFile struct {
	io.ReadCloser

	closeOnce sync.Once
	onClose   func()
}

func (f *streamedFile) Close() error {
	err := f.ReadCloser.Close()
	f.closeOnce.Do(f.onClose)

	return err
}
