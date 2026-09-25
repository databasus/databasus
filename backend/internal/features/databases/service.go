package databases

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"

	audit_logs "databasus-backend/internal/features/audit_logs"
	audit_logs_models "databasus-backend/internal/features/audit_logs/models"
	physical_core_service "databasus-backend/internal/features/backups/backups/core/physical/service"
	postgresql_physical "databasus-backend/internal/features/databases/databases/postgresql/physical"
	"databasus-backend/internal/features/notifiers"
	users_models "databasus-backend/internal/features/users/models"
	"databasus-backend/internal/util/encryption"
)

type DatabaseService struct {
	dbRepository    databaseStore
	notifierService *notifiers.NotifierService
	logger          *slog.Logger

	dbCreationListener         []DatabaseCreationListener
	dbRemoveListener           []DatabaseRemoveListener
	dbCopyListener             []DatabaseCopyListener
	dbBackupTypeChangeListener []DatabaseBackupTypeChangeListener

	workspaceService      workspaceService
	auditLogService       *audit_logs.AuditLogService
	fieldEncryptor        encryption.FieldEncryptor
	physicalBackupService *physical_core_service.PhysicalBackupService
}

func (s *DatabaseService) AddDbCreationListener(
	dbCreationListener DatabaseCreationListener,
) {
	s.dbCreationListener = append(s.dbCreationListener, dbCreationListener)
}

func (s *DatabaseService) AddDbRemoveListener(
	dbRemoveListener DatabaseRemoveListener,
) {
	s.dbRemoveListener = append(s.dbRemoveListener, dbRemoveListener)
}

func (s *DatabaseService) AddDbCopyListener(
	dbCopyListener DatabaseCopyListener,
) {
	s.dbCopyListener = append(s.dbCopyListener, dbCopyListener)
}

func (s *DatabaseService) AddDbBackupTypeChangeListener(
	dbBackupTypeChangeListener DatabaseBackupTypeChangeListener,
) {
	s.dbBackupTypeChangeListener = append(s.dbBackupTypeChangeListener, dbBackupTypeChangeListener)
}

func (s *DatabaseService) GetNotifierAttachedDatabasesIDs(
	notifierID uuid.UUID,
) ([]uuid.UUID, error) {
	databasesIDs, err := s.dbRepository.GetDatabasesIDsByNotifierID(notifierID)
	if err != nil {
		return nil, err
	}

	return databasesIDs, nil
}

func (s *DatabaseService) CreateDatabase(
	ctx context.Context,
	user *users_models.User,
	workspaceID uuid.UUID,
	database *Database,
) (*Database, error) {
	canManage, err := s.workspaceService.CanUserManageDBs(ctx, workspaceID, user)
	if err != nil {
		return nil, err
	}
	if !canManage {
		return nil, errors.New("insufficient permissions to create database in this workspace")
	}

	database.WorkspaceID = &workspaceID

	if err := database.Validate(); err != nil {
		return nil, err
	}

	// The database has no ID until it is saved, so the workspace is all there is to correlate on.
	logger := s.logger.With("workspace_id", workspaceID)

	tunneledDatabase, err := OpenTunnel(ctx, OpenTunnelSpec{
		Database:  database,
		Logger:    logger,
		Encryptor: s.fieldEncryptor,
	})
	if err != nil {
		return nil, err
	}

	defer tunneledDatabase.Close()

	// Never reassign database here: the original is what gets encrypted and saved below, and the
	// copy carries a loopback address, an ephemeral port and a disabled tunnel.
	databaseThroughTunnel := tunneledDatabase.GetDatabaseThroughTunnel()

	if err := databaseThroughTunnel.PopulateDbData(logger, s.fieldEncryptor); err != nil {
		return nil, fmt.Errorf("failed to auto-detect database data: %w", err)
	}

	if err := databaseThroughTunnel.TestConnection(logger, s.fieldEncryptor); err != nil {
		return nil, err
	}

	tunneledDatabase.CopyDiscoveredMetadataToOriginal()

	if err := database.EncryptSensitiveFields(s.fieldEncryptor); err != nil {
		return nil, fmt.Errorf("failed to encrypt sensitive fields: %w", err)
	}

	database, err = s.dbRepository.Save(database)
	if err != nil {
		return nil, err
	}

	for _, listener := range s.dbCreationListener {
		listener.OnDatabaseCreated(ctx, database.ID)
	}

	s.auditLogService.WriteAuditLog(ctx, audit_logs_models.AuditEntry{
		Message:     fmt.Sprintf("Database created: %s", database.Name),
		UserID:      &user.ID,
		WorkspaceID: &workspaceID,
	})

	return database, nil
}

func (s *DatabaseService) UpdateDatabase(
	ctx context.Context,
	user *users_models.User,
	database *Database,
) error {
	if database.ID == uuid.Nil {
		return errors.New("database ID is required for update")
	}

	existingDatabase, err := s.dbRepository.FindByID(database.ID)
	if err != nil {
		return err
	}

	if existingDatabase.WorkspaceID == nil {
		return errors.New("cannot update database without workspace")
	}

	canManage, err := s.workspaceService.CanUserManageDBs(ctx, *existingDatabase.WorkspaceID, user)
	if err != nil {
		return err
	}
	if !canManage {
		return errors.New("insufficient permissions to update this database")
	}

	if err := database.ValidateUpdate(*existingDatabase, *database); err != nil {
		return err
	}

	for _, notifier := range database.Notifiers {
		if notifier.WorkspaceID != *existingDatabase.WorkspaceID {
			return errors.New("notifier does not belong to this workspace")
		}
	}

	oldBackupType := getPhysicalBackupType(existingDatabase)

	existingDatabase.Update(database)

	if err := existingDatabase.Validate(); err != nil {
		return err
	}

	logger := s.logger.With("database_id", existingDatabase.ID)

	tunneledDatabase, err := OpenTunnel(ctx, OpenTunnelSpec{
		Database:  existingDatabase,
		Logger:    logger,
		Encryptor: s.fieldEncryptor,
	})
	if err != nil {
		return err
	}

	defer tunneledDatabase.Close()

	// Never reassign existingDatabase here: it is what gets encrypted and saved below.
	databaseThroughTunnel := tunneledDatabase.GetDatabaseThroughTunnel()

	if err := databaseThroughTunnel.PopulateDbData(logger, s.fieldEncryptor); err != nil {
		return fmt.Errorf("failed to auto-detect database data: %w", err)
	}

	if err := databaseThroughTunnel.TestConnection(logger, s.fieldEncryptor); err != nil {
		return err
	}

	tunneledDatabase.CopyDiscoveredMetadataToOriginal()

	oldName := existingDatabase.Name

	if err := existingDatabase.EncryptSensitiveFields(s.fieldEncryptor); err != nil {
		return fmt.Errorf("failed to encrypt sensitive fields: %w", err)
	}

	_, err = s.dbRepository.Save(existingDatabase)
	if err != nil {
		return err
	}

	if newBackupType := getPhysicalBackupType(existingDatabase); newBackupType != oldBackupType {
		for _, listener := range s.dbBackupTypeChangeListener {
			listener.OnBackupTypeChanged(ctx, BackupTypeChange{
				DatabaseID:    existingDatabase.ID,
				OldBackupType: oldBackupType,
				NewBackupType: newBackupType,
			})
		}
	}

	if oldName != existingDatabase.Name {
		s.auditLogService.WriteAuditLog(ctx, audit_logs_models.AuditEntry{
			Message:     fmt.Sprintf("Database updated and renamed from '%s' to '%s'", oldName, existingDatabase.Name),
			UserID:      &user.ID,
			WorkspaceID: existingDatabase.WorkspaceID,
		})
	} else {
		s.auditLogService.WriteAuditLog(ctx, audit_logs_models.AuditEntry{
			Message:     fmt.Sprintf("Database updated: %s", existingDatabase.Name),
			UserID:      &user.ID,
			WorkspaceID: existingDatabase.WorkspaceID,
		})
	}

	return nil
}

func (s *DatabaseService) DeleteDatabase(
	ctx context.Context,
	user *users_models.User,
	id uuid.UUID,
) error {
	existingDatabase, err := s.dbRepository.FindByID(id)
	if err != nil {
		return err
	}

	if existingDatabase.WorkspaceID == nil {
		return errors.New("cannot delete database without workspace")
	}

	canManage, err := s.workspaceService.CanUserManageDBs(ctx, *existingDatabase.WorkspaceID, user)
	if err != nil {
		return err
	}
	if !canManage {
		return errors.New("insufficient permissions to delete this database")
	}

	for _, listener := range s.dbRemoveListener {
		if err := listener.OnBeforeDatabaseRemove(ctx, id); err != nil {
			return err
		}
	}

	s.auditLogService.WriteAuditLog(ctx, audit_logs_models.AuditEntry{
		Message:     fmt.Sprintf("Database deleted: %s", existingDatabase.Name),
		UserID:      &user.ID,
		WorkspaceID: existingDatabase.WorkspaceID,
	})

	return s.dbRepository.Delete(id)
}

// DeleteForTest removes a database row through the listener chain without
// permission checks or audit logging. The path matters because listeners
// own external cleanup (backup rows, replication slots on source PG); tests
// that DELETE'd through raw SQL used to leak those resources.
//
// Idempotent: a missing row is treated as success so a test that already
// deleted the database through the public API does not crash on the fixture's
// teardown re-delete.
// DeleteForTest removes a database and its dependent rows during test teardown. Under the
// parallel test suite (go test -p=N) the cascade can deadlock against a concurrent writer in
// the same process (a backup scheduler still settling), surfacing as SQLSTATE 40P01. The
// aborted transaction is transient, so the whole delete — idempotent on a re-run — is retried.
func (s *DatabaseService) DeleteForTest(ctx context.Context, id uuid.UUID) error {
	const maxAttempts = 5

	deleteOnce := func() error {
		for _, listener := range s.dbRemoveListener {
			if err := listener.OnBeforeDatabaseRemove(ctx, id); err != nil {
				return err
			}
		}

		if err := s.dbRepository.Delete(id); err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		return nil
	}

	var err error
	for attempt := range maxAttempts {
		err = deleteOnce()
		if err == nil || !isTransientSerializationError(err) {
			return err
		}

		time.Sleep(time.Duration(attempt+1) * 20 * time.Millisecond)
	}

	return err
}

// isTransientSerializationError reports whether err is a PostgreSQL deadlock (40P01) or
// serialization failure (40001) — both transient, where retrying the aborted transaction is
// the correct response rather than failing.
func isTransientSerializationError(err error) bool {
	var pgErr *pgconn.PgError

	return errors.As(err, &pgErr) && (pgErr.Code == "40P01" || pgErr.Code == "40001")
}

func (s *DatabaseService) GetDatabase(
	ctx context.Context,
	user *users_models.User,
	id uuid.UUID,
) (*Database, error) {
	database, err := s.dbRepository.FindByID(id)
	if err != nil {
		return nil, err
	}

	if database.WorkspaceID == nil {
		return nil, errors.New("cannot access database without workspace")
	}

	canAccess, _, err := s.workspaceService.CanUserAccessWorkspace(ctx, *database.WorkspaceID, user)
	if err != nil {
		return nil, err
	}
	if !canAccess {
		return nil, errors.New("insufficient permissions to access this database")
	}

	database.HideSensitiveData()
	return database, nil
}

func (s *DatabaseService) GetDatabasesByWorkspace(
	ctx context.Context,
	user *users_models.User,
	workspaceID uuid.UUID,
) ([]*Database, error) {
	canAccess, _, err := s.workspaceService.CanUserAccessWorkspace(ctx, workspaceID, user)
	if err != nil {
		return nil, err
	}
	if !canAccess {
		return nil, errors.New("insufficient permissions to access this workspace")
	}

	databases, err := s.dbRepository.FindByWorkspaceID(workspaceID)
	if err != nil {
		return nil, err
	}

	for _, database := range databases {
		database.HideSensitiveData()
	}

	s.fillPhysicalLastBackupTimes(ctx, databases)

	return databases, nil
}

func (s *DatabaseService) IsNotifierUsing(
	ctx context.Context,
	user *users_models.User,
	notifierID uuid.UUID,
) (bool, error) {
	_, err := s.notifierService.GetNotifier(ctx, user, notifierID)
	if err != nil {
		return false, err
	}

	return s.dbRepository.IsNotifierUsing(notifierID)
}

func (s *DatabaseService) CountDatabasesByNotifier(
	ctx context.Context,
	user *users_models.User,
	notifierID uuid.UUID,
) (int, error) {
	_, err := s.notifierService.GetNotifier(ctx, user, notifierID)
	if err != nil {
		return 0, err
	}

	databaseIDs, err := s.dbRepository.GetDatabasesIDsByNotifierID(notifierID)
	if err != nil {
		return 0, err
	}

	return len(databaseIDs), nil
}

func (s *DatabaseService) TestDatabaseConnection(
	ctx context.Context,
	user *users_models.User,
	databaseID uuid.UUID,
) error {
	database, err := s.dbRepository.FindByID(databaseID)
	if err != nil {
		return err
	}

	if database.WorkspaceID == nil {
		return errors.New("cannot test connection for database without workspace")
	}

	canAccess, _, err := s.workspaceService.CanUserAccessWorkspace(ctx, *database.WorkspaceID, user)
	if err != nil {
		return err
	}
	if !canAccess {
		return errors.New("insufficient permissions to test connection for this database")
	}

	logger := s.logger.With("database_id", database.ID)

	tunneledDatabase, err := OpenTunnel(ctx, OpenTunnelSpec{
		Database:  database,
		Logger:    logger,
		Encryptor: s.fieldEncryptor,
	})
	if err != nil {
		return err
	}

	defer tunneledDatabase.Close()

	if err := tunneledDatabase.GetDatabaseThroughTunnel().TestConnection(logger, s.fieldEncryptor); err != nil {
		return err
	}

	tunneledDatabase.CopyDiscoveredMetadataToOriginal()

	database.LastBackupErrorMessage = nil

	_, err = s.dbRepository.Save(database)
	if err != nil {
		return err
	}

	return nil
}

func (s *DatabaseService) TestDatabaseConnectionDirect(
	ctx context.Context,
	user *users_models.User,
	database *Database,
) error {
	usingDatabase, err := s.resolveAuthorizedConnectionTarget(ctx, user, database)
	if err != nil {
		return err
	}

	return s.testDatabaseConnection(ctx, usingDatabase)
}

func (s *DatabaseService) TestTrustedDatabaseConnection(
	ctx context.Context,
	database *Database,
) error {
	return s.testDatabaseConnection(ctx, database)
}

func (s *DatabaseService) GetDatabaseByID(
	id uuid.UUID,
) (*Database, error) {
	return s.dbRepository.FindByID(id)
}

func (s *DatabaseService) GetAllDatabases() ([]*Database, error) {
	return s.dbRepository.GetAllDatabases()
}

func (s *DatabaseService) CountDatabases() (int64, error) {
	return s.dbRepository.CountAll()
}

func (s *DatabaseService) SetBackupError(databaseID uuid.UUID, errorMessage string) error {
	database, err := s.dbRepository.FindByID(databaseID)
	if err != nil {
		return err
	}

	database.LastBackupErrorMessage = &errorMessage
	_, err = s.dbRepository.Save(database)
	if err != nil {
		return err
	}

	return nil
}

func (s *DatabaseService) SetLastBackupTime(databaseID uuid.UUID, backupTime time.Time) error {
	database, err := s.dbRepository.FindByID(databaseID)
	if err != nil {
		return err
	}

	database.LastBackupTime = &backupTime
	database.LastBackupErrorMessage = nil // Clear any previous error
	_, err = s.dbRepository.Save(database)
	if err != nil {
		return err
	}

	return nil
}

func (s *DatabaseService) CopyDatabase(
	ctx context.Context,
	user *users_models.User,
	databaseID uuid.UUID,
) (*Database, error) {
	existingDatabase, err := s.dbRepository.FindByID(databaseID)
	if err != nil {
		return nil, err
	}

	if existingDatabase.WorkspaceID == nil {
		return nil, errors.New("cannot copy database without workspace")
	}

	canManage, err := s.workspaceService.CanUserManageDBs(ctx, *existingDatabase.WorkspaceID, user)
	if err != nil {
		return nil, err
	}
	if !canManage {
		return nil, errors.New("insufficient permissions to copy this database")
	}

	newDatabase := existingDatabase.CopyForNewDatabase()
	newDatabase.Name = existingDatabase.Name + " (Copy)"

	if err := newDatabase.Validate(); err != nil {
		return nil, err
	}

	copiedDatabase, err := s.dbRepository.Save(newDatabase)
	if err != nil {
		return nil, err
	}

	for _, listener := range s.dbCreationListener {
		listener.OnDatabaseCreated(ctx, copiedDatabase.ID)
	}

	for _, listener := range s.dbCopyListener {
		listener.OnDatabaseCopied(ctx, databaseID, copiedDatabase.ID)
	}

	s.auditLogService.WriteAuditLog(ctx, audit_logs_models.AuditEntry{
		Message:     fmt.Sprintf("Database copied: %s to %s", existingDatabase.Name, copiedDatabase.Name),
		UserID:      &user.ID,
		WorkspaceID: existingDatabase.WorkspaceID,
	})

	return copiedDatabase, nil
}

func (s *DatabaseService) TransferDatabaseToWorkspace(
	ctx context.Context,
	databaseID uuid.UUID,
	targetWorkspaceID uuid.UUID,
) error {
	database, err := s.dbRepository.FindByID(databaseID)
	if err != nil {
		return err
	}

	sourceWorkspaceID := database.WorkspaceID
	database.WorkspaceID = &targetWorkspaceID

	_, err = s.dbRepository.Save(database)
	if err != nil {
		return err
	}

	sourceWorkspace, err := s.workspaceService.GetWorkspaceByID(*sourceWorkspaceID)
	if err != nil {
		return fmt.Errorf("failed to get source workspace: %w", err)
	}

	targetWorkspace, err := s.workspaceService.GetWorkspaceByID(targetWorkspaceID)
	if err != nil {
		return fmt.Errorf("failed to get target workspace: %w", err)
	}

	s.auditLogService.WriteAuditLog(ctx, audit_logs_models.AuditEntry{
		Message: fmt.Sprintf(
			"Database transferred: %s from workspace '%s' to workspace '%s'",
			database.Name,
			sourceWorkspace.Name,
			targetWorkspace.Name,
		),
		UserID:      nil,
		WorkspaceID: &targetWorkspaceID,
	})

	return nil
}

func (s *DatabaseService) UpdateDatabaseNotifiers(
	databaseID uuid.UUID,
	newNotifiers []notifiers.Notifier,
) error {
	database, err := s.dbRepository.FindByID(databaseID)
	if err != nil {
		return err
	}

	database.Notifiers = newNotifiers

	_, err = s.dbRepository.Save(database)
	return err
}

func (s *DatabaseService) SetHealthStatus(
	databaseID uuid.UUID,
	healthStatus *HealthStatus,
) error {
	database, err := s.dbRepository.FindByID(databaseID)
	if err != nil {
		return err
	}

	database.HealthStatus = healthStatus
	_, err = s.dbRepository.Save(database)
	if err != nil {
		return err
	}

	return nil
}

func (s *DatabaseService) OnBeforeWorkspaceDeletion(workspaceID uuid.UUID) error {
	databases, err := s.dbRepository.FindByWorkspaceID(workspaceID)
	if err != nil {
		return err
	}

	if len(databases) > 0 {
		return fmt.Errorf(
			"workspace contains %d databases that must be deleted",
			len(databases),
		)
	}

	return nil
}

func (s *DatabaseService) ShouldSuggestReadOnlyUser(
	ctx context.Context,
	user *users_models.User,
	database *Database,
) (bool, []string, error) {
	var usingDatabase *Database

	if database.ID != uuid.Nil {
		existingDatabase, err := s.dbRepository.FindByID(database.ID)
		if err != nil {
			return false, nil, err
		}

		if existingDatabase.WorkspaceID == nil {
			return false, nil, errors.New("cannot check user for database without workspace")
		}

		canAccess, _, err := s.workspaceService.CanUserAccessWorkspace(
			ctx,
			*existingDatabase.WorkspaceID,
			user,
		)
		if err != nil {
			return false, nil, err
		}
		if !canAccess {
			return false, nil, errors.New("insufficient permissions to access this database")
		}

		if database.WorkspaceID != nil && *existingDatabase.WorkspaceID != *database.WorkspaceID {
			return false, nil, errors.New("database does not belong to this workspace")
		}

		existingDatabase.Update(database)

		if err := existingDatabase.Validate(); err != nil {
			return false, nil, err
		}

		usingDatabase = existingDatabase
	} else {
		if database.WorkspaceID != nil {
			canAccess, _, err := s.workspaceService.CanUserAccessWorkspace(
				ctx,
				*database.WorkspaceID,
				user,
			)
			if err != nil {
				return false, nil, err
			}
			if !canAccess {
				return false, nil, errors.New("insufficient permissions to access this workspace")
			}
		}

		usingDatabase = database
	}

	tunnelCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	logger := s.logger.With("database_id", usingDatabase.ID)

	tunneledDatabase, err := OpenTunnel(tunnelCtx, OpenTunnelSpec{
		Database:  usingDatabase,
		Logger:    logger,
		Encryptor: s.fieldEncryptor,
	})
	if err != nil {
		return false, nil, err
	}

	defer tunneledDatabase.Close()

	return tunneledDatabase.GetDatabaseThroughTunnel().ShouldSuggestReadOnlyUser(tunnelCtx, logger, s.fieldEncryptor)
}

func (s *DatabaseService) CreateReadOnlyUser(
	ctx context.Context,
	user *users_models.User,
	database *Database,
) (string, string, error) {
	var usingDatabase *Database

	if database.ID != uuid.Nil {
		existingDatabase, err := s.dbRepository.FindByID(database.ID)
		if err != nil {
			return "", "", err
		}

		if existingDatabase.WorkspaceID == nil {
			return "", "", errors.New("cannot create user for database without workspace")
		}

		canManage, err := s.workspaceService.CanUserManageDBs(ctx, *existingDatabase.WorkspaceID, user)
		if err != nil {
			return "", "", err
		}
		if !canManage {
			return "", "", errors.New("insufficient permissions to manage this database")
		}

		if database.WorkspaceID != nil && *existingDatabase.WorkspaceID != *database.WorkspaceID {
			return "", "", errors.New("database does not belong to this workspace")
		}

		existingDatabase.Update(database)

		if err := existingDatabase.Validate(); err != nil {
			return "", "", err
		}

		usingDatabase = existingDatabase
	} else {
		if database.WorkspaceID != nil {
			canManage, err := s.workspaceService.CanUserManageDBs(ctx, *database.WorkspaceID, user)
			if err != nil {
				return "", "", err
			}
			if !canManage {
				return "", "", errors.New("insufficient permissions to manage this workspace")
			}
		}

		usingDatabase = database
	}

	// MySQL and MariaDB commit CREATE USER and each GRANT implicitly, so their transaction cannot
	// roll a half-granted user back. WithoutCancel keeps the request_id while stopping a client
	// disconnect from cutting the sequence in half.
	tunnelCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
	defer cancel()

	logger := s.logger.With("database_id", usingDatabase.ID)

	tunneledDatabase, err := OpenTunnel(tunnelCtx, OpenTunnelSpec{
		Database:  usingDatabase,
		Logger:    logger,
		Encryptor: s.fieldEncryptor,
	})
	if err != nil {
		return "", "", err
	}

	defer tunneledDatabase.Close()

	databaseThroughTunnel := tunneledDatabase.GetDatabaseThroughTunnel()

	var username, password string

	switch usingDatabase.Type {
	case DatabaseTypePostgresLogical:
		username, password, err = databaseThroughTunnel.PostgresqlLogical.CreateReadOnlyUser(
			tunnelCtx, logger, s.fieldEncryptor,
		)
	case DatabaseTypeMysql:
		username, password, err = databaseThroughTunnel.Mysql.CreateReadOnlyUser(
			tunnelCtx, logger, s.fieldEncryptor,
		)
	case DatabaseTypeMariadb:
		username, password, err = databaseThroughTunnel.Mariadb.CreateReadOnlyUser(
			tunnelCtx, logger, s.fieldEncryptor,
		)
	case DatabaseTypeMongodb:
		username, password, err = databaseThroughTunnel.Mongodb.CreateReadOnlyUser(
			tunnelCtx, logger, s.fieldEncryptor,
		)
	default:
		return "", "", errors.New("read-only user creation not supported for this database type")
	}

	if err != nil {
		return "", "", err
	}

	if usingDatabase.WorkspaceID != nil {
		s.auditLogService.WriteAuditLog(ctx, audit_logs_models.AuditEntry{
			Message: fmt.Sprintf(
				"Read-only user created for database: %s (username: %s)",
				usingDatabase.Name,
				username,
			),
			UserID:      &user.ID,
			WorkspaceID: usingDatabase.WorkspaceID,
		})
	}

	return username, password, nil
}

func (s *DatabaseService) CreateReplicationOnlyUser(
	ctx context.Context,
	user *users_models.User,
	database *Database,
) (*postgresql_physical.ReplicationOnlyUser, error) {
	var usingDatabase *Database

	if database.ID != uuid.Nil {
		existingDatabase, err := s.dbRepository.FindByID(database.ID)
		if err != nil {
			return nil, err
		}

		if existingDatabase.WorkspaceID == nil {
			return nil, errors.New("cannot create user for database without workspace")
		}

		canManage, err := s.workspaceService.CanUserManageDBs(ctx, *existingDatabase.WorkspaceID, user)
		if err != nil {
			return nil, err
		}
		if !canManage {
			return nil, errors.New("insufficient permissions to manage this database")
		}

		if database.WorkspaceID != nil && *existingDatabase.WorkspaceID != *database.WorkspaceID {
			return nil, errors.New("database does not belong to this workspace")
		}

		existingDatabase.Update(database)

		if err := existingDatabase.Validate(); err != nil {
			return nil, err
		}

		usingDatabase = existingDatabase
	} else {
		if database.WorkspaceID != nil {
			canManage, err := s.workspaceService.CanUserManageDBs(ctx, *database.WorkspaceID, user)
			if err != nil {
				return nil, err
			}
			if !canManage {
				return nil, errors.New("insufficient permissions to manage this workspace")
			}
		}

		usingDatabase = database
	}

	if usingDatabase.Type != DatabaseTypePostgresPhysical {
		return nil, errors.New(
			"replication-only user creation is only supported for POSTGRES_PHYSICAL databases",
		)
	}

	if usingDatabase.PostgresqlPhysical == nil {
		return nil, errors.New("physical database details are missing")
	}

	tunnelCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	logger := s.logger.With("database_id", usingDatabase.ID)

	tunneledDatabase, err := OpenTunnel(tunnelCtx, OpenTunnelSpec{
		Database:  usingDatabase,
		Logger:    logger,
		Encryptor: s.fieldEncryptor,
	})
	if err != nil {
		return nil, err
	}

	defer tunneledDatabase.Close()

	createdUser, err := tunneledDatabase.GetDatabaseThroughTunnel().
		PostgresqlPhysical.CreateReplicationOnlyUser(tunnelCtx, logger, s.fieldEncryptor)
	if err != nil {
		return nil, err
	}

	if usingDatabase.WorkspaceID != nil {
		s.auditLogService.WriteAuditLog(ctx, audit_logs_models.AuditEntry{
			Message: fmt.Sprintf(
				"Replication-only user created for database: %s (username: %s)",
				usingDatabase.Name,
				createdUser.Username,
			),
			UserID:      &user.ID,
			WorkspaceID: usingDatabase.WorkspaceID,
		})
	}

	return createdUser, nil
}

func (s *DatabaseService) testDatabaseConnection(ctx context.Context, database *Database) error {
	if database == nil {
		return errors.New("database is required")
	}

	logger := s.logger.With("database_id", database.ID)

	tunneledDatabase, err := OpenTunnel(ctx, OpenTunnelSpec{
		Database:  database,
		Logger:    logger,
		Encryptor: s.fieldEncryptor,
	})
	if err != nil {
		return err
	}

	defer tunneledDatabase.Close()

	return tunneledDatabase.GetDatabaseThroughTunnel().TestConnection(logger, s.fieldEncryptor)
}

// fillPhysicalLastBackupTimes populates LastBackupTime for physical databases,
// whose backups (FULL / INCREMENTAL / WAL) live outside the databases table and so
// are not denormalized onto the row the way logical backups are. A failure here is
// non-fatal: the list must still render, just without the last-backup decoration.
func (s *DatabaseService) fillPhysicalLastBackupTimes(ctx context.Context, databases []*Database) {
	var physicalDatabaseIDs []uuid.UUID

	for _, database := range databases {
		if database.Type == DatabaseTypePostgresPhysical {
			physicalDatabaseIDs = append(physicalDatabaseIDs, database.ID)
		}
	}

	if len(physicalDatabaseIDs) == 0 {
		return
	}

	lastBackupTimes, err := s.physicalBackupService.GetLastBackupTimesByDatabaseIDs(physicalDatabaseIDs)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to load physical last backup times", "error", err)

		return
	}

	for _, database := range databases {
		if lastBackupTime, hasBackup := lastBackupTimes[database.ID]; hasBackup {
			database.LastBackupTime = &lastBackupTime
		}
	}
}

func (s *DatabaseService) resolveAuthorizedConnectionTarget(
	ctx context.Context,
	user *users_models.User,
	database *Database,
) (*Database, error) {
	if database == nil {
		return nil, errors.New("database is required")
	}

	if database.ID == uuid.Nil {
		if database.WorkspaceID == nil {
			return nil, errors.New("workspaceId is required")
		}

		canManage, err := s.workspaceService.CanUserManageDBs(ctx, *database.WorkspaceID, user)
		if err != nil {
			return nil, errors.Join(ErrDatabaseConnectionAuthorizationLookup, err)
		}
		if !canManage {
			return nil, ErrInsufficientPermissionsToTestDatabaseConnection
		}

		return database, nil
	}

	existingDatabase, err := s.dbRepository.FindByID(database.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInsufficientPermissionsToTestDatabaseConnection
		}

		return nil, errors.Join(ErrDatabaseConnectionTargetLookup, err)
	}

	if existingDatabase.WorkspaceID == nil {
		return nil, ErrInsufficientPermissionsToTestDatabaseConnection
	}

	canManage, err := s.workspaceService.CanUserManageDBs(ctx, *existingDatabase.WorkspaceID, user)
	if err != nil {
		return nil, errors.Join(ErrDatabaseConnectionAuthorizationLookup, err)
	}
	if !canManage {
		return nil, ErrInsufficientPermissionsToTestDatabaseConnection
	}

	if database.WorkspaceID != nil && existingDatabase.WorkspaceID != nil &&
		*existingDatabase.WorkspaceID != *database.WorkspaceID {
		return nil, errors.New("database does not belong to this workspace")
	}

	existingDatabase.Update(database)

	if err := existingDatabase.Validate(); err != nil {
		return nil, err
	}

	return existingDatabase, nil
}

func getPhysicalBackupType(database *Database) postgresql_physical.BackupType {
	if database.PostgresqlPhysical == nil {
		return ""
	}

	return database.PostgresqlPhysical.BackupType
}
