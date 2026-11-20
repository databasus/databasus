package userdatabase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"unicode/utf8"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/apierror"
	"databasus-backend/internal/lib/audit"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/lib/logging"
	"databasus-backend/internal/modules/databases/internal/providers"
	"databasus-backend/internal/modules/notifiers"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/modules/workspaces"
)

const maxDatabaseNameLength = 255

type Deps struct {
	Repositories Repositories
	Registry     *providers.Registry
	Transactor   *database.Transactor
	Logger       *slog.Logger
	AuditLog     *audit.Log
	Users        users.Module
	Workspaces   workspaces.Module
	Notifiers    notifiers.Module
}

// Service keeps databases with their provider settings and notifier attachments, and publishes the
// events of their lifecycle in the transactions that change them.
type Service struct {
	repository               *databaseRepository
	notifierAttachments      *notifierAttachmentRepository
	registry                 *providers.Registry
	transactor               *database.Transactor
	logger                   *slog.Logger
	audit                    auditRecorder
	workspaces               workspaces.Module
	notifiers                notifiers.Module
	createdHandlers          []CreatedHandler
	copiedHandlers           []CopiedHandler
	backupTypeChangeHandlers []BackupTypeChangedHandler
	deletingHandlers         []DeletingHandler
}

func NewService(deps Deps) *Service {
	return &Service{
		repository:               deps.Repositories.databases,
		notifierAttachments:      deps.Repositories.notifierAttachments,
		registry:                 deps.Registry,
		transactor:               deps.Transactor,
		logger:                   deps.Logger,
		audit:                    auditRecorder{auditLog: deps.AuditLog, users: deps.Users},
		workspaces:               deps.Workspaces,
		notifiers:                deps.Notifiers,
		createdHandlers:          nil,
		copiedHandlers:           nil,
		backupTypeChangeHandlers: nil,
		deletingHandlers:         nil,
	}
}

// The subscriptions are made by internal/app while it builds the application, before any request
// runs. The handlers of one event run in the order they subscribed.

func (s *Service) SubscribeToCreated(handler CreatedHandler) {
	s.createdHandlers = append(s.createdHandlers, handler)
}

func (s *Service) SubscribeToCopied(handler CopiedHandler) {
	s.copiedHandlers = append(s.copiedHandlers, handler)
}

func (s *Service) SubscribeToBackupTypeChanged(handler BackupTypeChangedHandler) {
	s.backupTypeChangeHandlers = append(s.backupTypeChangeHandlers, handler)
}

func (s *Service) SubscribeToDeleting(handler DeletingHandler) {
	s.deletingHandlers = append(s.deletingHandlers, handler)
}

// ListDatabases pages through the databases of a workspace, ordered by name, optionally only those
// that use one notifier.
func (s *Service) ListDatabases(ctx context.Context, actor users.Actor, query ListQuery) (Page, error) {
	limit := defaultDatabasePageLimit
	if query.Limit != nil {
		limit = *query.Limit
	}

	if limit < 1 || limit > maxDatabasePageLimit {
		return Page{}, apierror.NewInvalidFieldsProblem([]apierror.FieldError{{
			Field:  "limit",
			Detail: fmt.Sprintf("must be between 1 and %d", maxDatabasePageLimit),
		}})
	}

	var after *database.NameCursor

	if query.After != nil {
		cursor, err := database.DecodeNameCursor(*query.After)
		if err != nil {
			return Page{}, err
		}

		after = &cursor
	}

	err := s.workspaces.Authorize(ctx, actor, query.WorkspaceID, workspaces.ActionView)
	if err != nil {
		return Page{}, err
	}

	listedDatabases, err := s.repository.list(ctx, databaseListFilter{
		workspaceID: query.WorkspaceID,
		notifierID:  query.NotifierID,
		limit:       limit + 1,
		after:       after,
	})
	if err != nil {
		return Page{}, err
	}

	pageDatabases, err := s.loadWithSettings(ctx, listedDatabases[:min(len(listedDatabases), limit)])
	if err != nil {
		return Page{}, err
	}

	page := Page{Databases: pageDatabases, NextCursor: nil}

	if len(listedDatabases) > limit {
		lastDatabase := listedDatabases[limit-1]
		nextCursor := database.NameCursor{Name: lastDatabase.Name, ID: lastDatabase.ID}.Encode()
		page.NextCursor = &nextCursor
	}

	return page, nil
}

// CreateDatabase connects and detects outside the transaction, so a slow server holds no pooled
// connection, and checks the notifiers after the attachment rows are inserted, so a notifier transfer
// running at the same time either waits for the attachment or is seen moved.
func (s *Service) CreateDatabase(ctx context.Context, actor users.Actor, creation Creation) (WithSettings, error) {
	err := validateName(creation.Name)
	if err != nil {
		return WithSettings{}, err
	}

	err = s.workspaces.Authorize(ctx, actor, creation.WorkspaceID, workspaces.ActionEditContent)
	if err != nil {
		return WithSettings{}, err
	}

	provider, err := s.registry.GetProvider(creation.Settings.GetType())
	if err != nil {
		return WithSettings{}, err
	}

	err = provider.Validate(creation.Settings)
	if err != nil {
		return WithSettings{}, err
	}

	ctx = logging.With(ctx, slog.String("workspace_id", creation.WorkspaceID.String()))

	detectedSettings, err := s.connectAndDetect(ctx, provider, provider.PrepareForCreate(creation.Settings))
	if err != nil {
		return WithSettings{}, err
	}

	var createdDatabase WithSettings

	err = s.transactor.Transaction(ctx, func(ctx context.Context) error {
		insertedDatabase, err := s.repository.insert(ctx, creation.WorkspaceID, creation.Name, provider.GetType())
		if err != nil {
			return err
		}

		err = provider.GetRepository().Upsert(ctx, insertedDatabase.ID, detectedSettings)
		if err != nil {
			return err
		}

		err = s.attachNotifiers(ctx, insertedDatabase, creation.NotifierIDs)
		if err != nil {
			return err
		}

		err = s.publishCreated(ctx, insertedDatabase.ID)
		if err != nil {
			return err
		}

		err = s.audit.record(ctx, actor, insertedDatabase.WorkspaceID, "Database created: "+insertedDatabase.Name)
		if err != nil {
			return err
		}

		createdDatabase, err = s.loadOneWithSettings(ctx, insertedDatabase)

		return err
	})
	if err != nil {
		return WithSettings{}, err
	}

	return createdDatabase, nil
}

func (s *Service) GetDatabase(ctx context.Context, actor users.Actor, databaseID uuid.UUID) (WithSettings, error) {
	foundDatabase, err := s.getAuthorized(ctx, actor, databaseID, workspaces.ActionView, noRowLock)
	if err != nil {
		return WithSettings{}, err
	}

	return s.loadOneWithSettings(ctx, foundDatabase)
}

// UpdateDatabase keeps the type of the database and refuses settings of another type before anything
// connects. It connects with the merged settings outside the transaction, then writes them under the
// row lock.
func (s *Service) UpdateDatabase(ctx context.Context, actor users.Actor, change Change) (WithSettings, error) {
	err := validateName(change.Name)
	if err != nil {
		return WithSettings{}, err
	}

	storedDatabase, err := s.getAuthorized(ctx, actor, change.DatabaseID, workspaces.ActionEditContent, noRowLock)
	if err != nil {
		return WithSettings{}, err
	}

	ctx = logging.With(ctx, slog.String("database_id", storedDatabase.ID.String()))

	storedWithSettings, err := s.loadOneWithSettings(ctx, storedDatabase)
	if err != nil {
		return WithSettings{}, err
	}

	mergedSettings, err := s.mergeAndValidate(storedWithSettings, change.Settings)
	if err != nil {
		return WithSettings{}, err
	}

	provider, err := s.registry.GetProvider(storedDatabase.Type)
	if err != nil {
		return WithSettings{}, err
	}

	detectedSettings, err := s.connectAndDetect(ctx, provider, mergedSettings)
	if err != nil {
		return WithSettings{}, err
	}

	var updatedDatabase WithSettings

	err = s.transactor.Transaction(ctx, func(ctx context.Context) error {
		lockedDatabase, err := s.getAuthorized(
			ctx,
			actor,
			change.DatabaseID,
			workspaces.ActionEditContent,
			forUpdateRowLock,
		)
		if err != nil {
			return err
		}

		lockedWithSettings, err := s.loadOneWithSettings(ctx, lockedDatabase)
		if err != nil {
			return err
		}

		err = s.repository.updateName(ctx, lockedDatabase.ID, change.Name)
		if err != nil {
			return err
		}

		err = provider.GetRepository().Upsert(ctx, lockedDatabase.ID, detectedSettings)
		if err != nil {
			return err
		}

		err = s.attachNotifiers(ctx, lockedDatabase, change.NotifierIDs)
		if err != nil {
			return err
		}

		err = s.publishBackupTypeChange(ctx, lockedWithSettings, detectedSettings)
		if err != nil {
			return err
		}

		message := "Database updated: " + change.Name
		if lockedDatabase.Name != change.Name {
			message = fmt.Sprintf("Database updated and renamed from '%s' to '%s'", lockedDatabase.Name, change.Name)
		}

		err = s.audit.record(ctx, actor, lockedDatabase.WorkspaceID, message)
		if err != nil {
			return err
		}

		lockedDatabase.Name = change.Name

		updatedDatabase, err = s.loadOneWithSettings(ctx, lockedDatabase)

		return err
	})
	if err != nil {
		return WithSettings{}, err
	}

	return updatedDatabase, nil
}

// CopyDatabase creates a database with the original's type, settings, secrets and notifiers, but its
// own replication slot and no backup history. It does not connect, as copying never has.
func (s *Service) CopyDatabase(ctx context.Context, actor users.Actor, databaseID uuid.UUID) (WithSettings, error) {
	var copiedDatabase WithSettings

	err := s.transactor.Transaction(ctx, func(ctx context.Context) error {
		originalDatabase, err := s.getAuthorized(ctx, actor, databaseID, workspaces.ActionEditContent, noRowLock)
		if err != nil {
			return err
		}

		originalWithSettings, err := s.loadOneWithSettings(ctx, originalDatabase)
		if err != nil {
			return err
		}

		provider, err := s.registry.GetProvider(originalDatabase.Type)
		if err != nil {
			return err
		}

		insertedDatabase, err := s.repository.insertCopy(ctx, originalDatabase, originalDatabase.Name+" (Copy)")
		if err != nil {
			return err
		}

		err = provider.GetRepository().Upsert(
			ctx, insertedDatabase.ID, provider.CopyForNewDatabase(originalWithSettings.Settings),
		)
		if err != nil {
			return err
		}

		err = s.attachNotifiers(ctx, insertedDatabase, originalWithSettings.NotifierIDs)
		if err != nil {
			return err
		}

		err = s.publishCreated(ctx, insertedDatabase.ID)
		if err != nil {
			return err
		}

		for _, handler := range s.copiedHandlers {
			err := handler(ctx, Copied{OriginalDatabaseID: originalDatabase.ID, CopiedDatabaseID: insertedDatabase.ID})
			if err != nil {
				return err
			}
		}

		err = s.audit.record(ctx, actor, originalDatabase.WorkspaceID,
			fmt.Sprintf("Database copied: %s to %s", originalDatabase.Name, insertedDatabase.Name))
		if err != nil {
			return err
		}

		copiedDatabase, err = s.loadOneWithSettings(ctx, insertedDatabase)

		return err
	})
	if err != nil {
		return WithSettings{}, err
	}

	return copiedDatabase, nil
}

// getAuthorized answers a database of a workspace the actor is not a member of exactly as a missing
// database, so its existence stays hidden.
func (s *Service) getAuthorized(
	ctx context.Context,
	actor users.Actor,
	databaseID uuid.UUID,
	action workspaces.Action,
	lock rowLock,
) (Database, error) {
	foundDatabase, isFound, err := s.repository.get(ctx, databaseID, lock)
	if err != nil {
		return Database{}, err
	}

	if !isFound {
		return Database{}, ErrDatabaseNotFound
	}

	err = s.workspaces.Authorize(ctx, actor, foundDatabase.WorkspaceID, action)

	var problem *apierror.Problem
	if errors.As(err, &problem) && problem.Code == apierror.CodeNotFound {
		return Database{}, ErrDatabaseNotFound
	}

	if err != nil {
		return Database{}, err
	}

	return foundDatabase, nil
}

// mergeAndValidate fills the secrets the incoming settings leave out from the stored ones, then
// validates the result. Settings of another type than the stored database's are refused.
func (s *Service) mergeAndValidate(
	storedDatabase WithSettings,
	incomingSettings providers.Settings,
) (providers.Settings, error) {
	if incomingSettings.GetType() != storedDatabase.Type {
		return nil, ErrTypeChanged
	}

	provider, err := s.registry.GetProvider(storedDatabase.Type)
	if err != nil {
		return nil, err
	}

	mergedSettings, err := provider.MergeStoredSettings(providers.SettingsMerge{
		Incoming: incomingSettings,
		Stored:   storedDatabase.Settings,
	})
	if err != nil {
		return nil, err
	}

	err = provider.Validate(mergedSettings)
	if err != nil {
		return nil, err
	}

	return mergedSettings, nil
}

// connectAndDetect returns the settings with what the server reports, and refuses a server that
// cannot be reached or would not let a backup run.
func (s *Service) connectAndDetect(
	ctx context.Context,
	provider providers.Provider,
	settings providers.Settings,
) (providers.Settings, error) {
	connectedDatabase, err := provider.Connect(ctx, settings)
	if err != nil {
		return nil, ToConnectionError(err)
	}

	defer connectedDatabase.Close()

	detectedSettings, err := connectedDatabase.Detect(ctx, s.logger)
	if err != nil {
		return nil, ToConnectionError(err)
	}

	err = connectedDatabase.TestConnection(ctx, s.logger)
	if err != nil {
		return nil, ToConnectionError(err)
	}

	return detectedSettings, nil
}

// attachNotifiers replaces the attachments and only then reads the notifiers, so the check sees a
// transfer that committed while the insert waited for it.
func (s *Service) attachNotifiers(ctx context.Context, attachedDatabase Database, notifierIDs []uuid.UUID) error {
	uniqueNotifierIDs := uniqueIDs(notifierIDs)

	err := s.notifierAttachments.replace(ctx, attachedDatabase.ID, uniqueNotifierIDs)
	if err != nil {
		return err
	}

	if len(uniqueNotifierIDs) == 0 {
		return nil
	}

	attachedNotifiers, err := s.notifiers.GetNotifiers(ctx, uniqueNotifierIDs)
	if err != nil {
		return err
	}

	if len(attachedNotifiers) != len(uniqueNotifierIDs) {
		return ErrNotifierNotInWorkspace
	}

	for _, attachedNotifier := range attachedNotifiers {
		if attachedNotifier.WorkspaceID != attachedDatabase.WorkspaceID {
			return ErrNotifierNotInWorkspace
		}
	}

	return nil
}

func (s *Service) publishCreated(ctx context.Context, databaseID uuid.UUID) error {
	for _, handler := range s.createdHandlers {
		err := handler(ctx, Created{DatabaseID: databaseID})
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) publishBackupTypeChange(
	ctx context.Context,
	storedDatabase WithSettings,
	changedSettings providers.Settings,
) error {
	provider, err := s.registry.GetProvider(storedDatabase.Type)
	if err != nil {
		return err
	}

	storedFacts, err := provider.GetTelemetryFacts(storedDatabase.Settings)
	if err != nil {
		return err
	}

	changedFacts, err := provider.GetTelemetryFacts(changedSettings)
	if err != nil {
		return err
	}

	if storedFacts.PhysicalBackupType == changedFacts.PhysicalBackupType {
		return nil
	}

	for _, handler := range s.backupTypeChangeHandlers {
		err := handler(ctx, BackupTypeChanged{
			DatabaseID:    storedDatabase.ID,
			OldBackupType: storedFacts.PhysicalBackupType,
			NewBackupType: changedFacts.PhysicalBackupType,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) loadOneWithSettings(ctx context.Context, foundDatabase Database) (WithSettings, error) {
	databasesWithSettings, err := s.loadWithSettings(ctx, []Database{foundDatabase})
	if err != nil {
		return WithSettings{}, err
	}

	return databasesWithSettings[0], nil
}

// loadWithSettings reads the attachments of the databases in one query and their settings with one
// query per type among them, keeping their order. A database without its settings row is an internal
// error.
func (s *Service) loadWithSettings(ctx context.Context, databases []Database) ([]WithSettings, error) {
	if len(databases) == 0 {
		return []WithSettings{}, nil
	}

	attachedDatabases, err := s.loadNotifierIDs(ctx, databases)
	if err != nil {
		return nil, err
	}

	settingsByDatabaseID, err := s.loadSettingsByDatabaseID(ctx, attachedDatabases)
	if err != nil {
		return nil, err
	}

	databasesWithSettings := make([]WithSettings, 0, len(attachedDatabases))
	for _, listedDatabase := range attachedDatabases {
		settings, isFound := settingsByDatabaseID[listedDatabase.ID]
		if !isFound {
			return nil, fmt.Errorf("%w: %s", errSettingsMissing, listedDatabase.ID)
		}

		databasesWithSettings = append(
			databasesWithSettings,
			WithSettings{Database: listedDatabase, Settings: settings},
		)
	}

	return databasesWithSettings, nil
}

// loadSettingsByDatabaseID leaves out the databases without a settings row.
func (s *Service) loadSettingsByDatabaseID(
	ctx context.Context,
	databases []Database,
) (map[uuid.UUID]providers.Settings, error) {
	databaseIDsByType := map[Type][]uuid.UUID{}
	for _, listedDatabase := range databases {
		databaseIDsByType[listedDatabase.Type] = append(databaseIDsByType[listedDatabase.Type], listedDatabase.ID)
	}

	settingsByDatabaseID := map[uuid.UUID]providers.Settings{}

	for databaseType, databaseIDs := range databaseIDsByType {
		provider, err := s.registry.GetProvider(databaseType)
		if err != nil {
			return nil, err
		}

		settingsOfType, err := provider.GetRepository().ListByDatabaseIDs(ctx, databaseIDs)
		if err != nil {
			return nil, err
		}

		maps.Copy(settingsByDatabaseID, settingsOfType)
	}

	return settingsByDatabaseID, nil
}

func (s *Service) loadNotifierIDs(ctx context.Context, databases []Database) ([]Database, error) {
	if len(databases) == 0 {
		return databases, nil
	}

	databaseIDs := make([]uuid.UUID, 0, len(databases))
	for _, listedDatabase := range databases {
		databaseIDs = append(databaseIDs, listedDatabase.ID)
	}

	notifierIDsByDatabaseID, err := s.notifierAttachments.listByDatabaseIDs(ctx, databaseIDs)
	if err != nil {
		return nil, err
	}

	attachedDatabases := make([]Database, 0, len(databases))
	for _, listedDatabase := range databases {
		if notifierIDs, isAttached := notifierIDsByDatabaseID[listedDatabase.ID]; isAttached {
			listedDatabase.NotifierIDs = notifierIDs
		}

		attachedDatabases = append(attachedDatabases, listedDatabase)
	}

	return attachedDatabases, nil
}

// ToConnectionError answers the refusal of the user's server as database_connection_failed, and keeps
// an internal failure, such as a secret that does not decrypt, internal.
func ToConnectionError(err error) error {
	var problem *apierror.Problem
	if errors.As(err, &problem) || errors.Is(err, providers.ErrSecretUnreadable) ||
		errors.Is(err, providers.ErrSettingsOfAnotherType) {
		return err
	}

	return NewConnectionFailedError(err)
}

func validateName(name string) error {
	nameLength := utf8.RuneCountInString(name)
	if nameLength < 1 || nameLength > maxDatabaseNameLength {
		return ErrInvalidName
	}

	return nil
}

func uniqueIDs(ids []uuid.UUID) []uuid.UUID {
	seenIDs := make(map[uuid.UUID]struct{}, len(ids))
	uniqueIDs := make([]uuid.UUID, 0, len(ids))

	for _, id := range ids {
		if _, isSeen := seenIDs[id]; isSeen {
			continue
		}

		seenIDs[id] = struct{}{}
		uniqueIDs = append(uniqueIDs, id)
	}

	return uniqueIDs
}
