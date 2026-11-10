package notifier

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"unicode/utf8"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/apierror"
	"databasus-backend/internal/lib/audit"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/modules/notifiers/internal/notification"
	"databasus-backend/internal/modules/notifiers/internal/providers"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/modules/workspaces"
)

const maxNotifierNameLength = 255

type Deps struct {
	Repositories Repositories
	Registry     *providers.Registry
	Transactor   *database.Transactor
	AuditLog     *audit.Log
	Users        users.Module
	Workspaces   workspaces.Module
}

// Service keeps notifiers with their provider settings and records the outcome of every send.
type Service struct {
	repository *notifierRepository
	registry   *providers.Registry
	transactor *database.Transactor
	audit      auditRecorder
	workspaces workspaces.Module
}

func NewService(deps Deps) *Service {
	return &Service{
		repository: deps.Repositories.notifiers,
		registry:   deps.Registry,
		transactor: deps.Transactor,
		audit:      auditRecorder{auditLog: deps.AuditLog, users: deps.Users},
		workspaces: deps.Workspaces,
	}
}

// ListNotifiers pages through the notifiers of a workspace, ordered by name.
func (s *Service) ListNotifiers(ctx context.Context, actor users.Actor, query ListQuery) (Page, error) {
	limit := defaultNotifierPageLimit
	if query.Limit != nil {
		limit = *query.Limit
	}

	if limit < 1 || limit > maxNotifierPageLimit {
		return Page{}, apierror.NewInvalidFieldsProblem([]apierror.FieldError{{
			Field:  "limit",
			Detail: fmt.Sprintf("must be between 1 and %d", maxNotifierPageLimit),
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

	listedNotifiers, err := s.repository.list(ctx, notifierListFilter{
		workspaceID: query.WorkspaceID,
		limit:       limit + 1,
		after:       after,
	})
	if err != nil {
		return Page{}, err
	}

	pageNotifiers, err := s.loadSettings(ctx, listedNotifiers[:min(len(listedNotifiers), limit)])
	if err != nil {
		return Page{}, err
	}

	page := Page{Notifiers: pageNotifiers, NextCursor: nil}

	if len(listedNotifiers) > limit {
		lastNotifier := listedNotifiers[limit-1]
		nextCursor := database.NameCursor{Name: lastNotifier.Name, ID: lastNotifier.ID}.Encode()
		page.NextCursor = &nextCursor
	}

	return page, nil
}

func (s *Service) CreateNotifier(ctx context.Context, actor users.Actor, creation Creation) (WithSettings, error) {
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

	createdNotifier := WithSettings{
		ID:            uuid.New(),
		WorkspaceID:   creation.WorkspaceID,
		Name:          creation.Name,
		Type:          provider.GetType(),
		LastSendError: nil,
		Settings:      creation.Settings,
	}

	err = s.transactor.Transaction(ctx, func(ctx context.Context) error {
		err := s.repository.insertWithSettings(ctx, createdNotifier.Notifier, providerSettings{
			repository: provider.GetRepository(),
			settings:   createdNotifier.Settings,
		})
		if err != nil {
			return err
		}

		return s.audit.record(ctx, actor, createdNotifier.WorkspaceID, "Notifier created: "+createdNotifier.Name)
	})
	if err != nil {
		return WithSettings{}, err
	}

	return createdNotifier, nil
}

func (s *Service) GetNotifier(ctx context.Context, actor users.Actor, notifierID uuid.UUID) (WithSettings, error) {
	foundNotifier, err := s.getAuthorized(ctx, actor, notifierID, workspaces.ActionView)
	if err != nil {
		return WithSettings{}, err
	}

	return s.loadOneWithSettings(ctx, foundNotifier)
}

// UpdateNotifier keeps the type of the notifier: settings of another type are refused before
// anything is written.
func (s *Service) UpdateNotifier(ctx context.Context, actor users.Actor, change Change) (WithSettings, error) {
	err := validateName(change.Name)
	if err != nil {
		return WithSettings{}, err
	}

	storedNotifier, err := s.getAuthorized(ctx, actor, change.NotifierID, workspaces.ActionEditContent)
	if err != nil {
		return WithSettings{}, err
	}

	if change.Settings.GetType() != storedNotifier.Type {
		return WithSettings{}, ErrTypeChanged
	}

	mergedSettings, err := s.mergeAndValidate(ctx, storedNotifier, change.Settings)
	if err != nil {
		return WithSettings{}, err
	}

	provider, err := s.registry.GetProvider(storedNotifier.Type)
	if err != nil {
		return WithSettings{}, err
	}

	err = s.transactor.Transaction(ctx, func(ctx context.Context) error {
		err := s.repository.updateNameAndSettings(ctx, storedNotifier.ID, change.Name, providerSettings{
			repository: provider.GetRepository(),
			settings:   mergedSettings,
		})
		if err != nil {
			return err
		}

		message := "Notifier updated: " + change.Name
		if storedNotifier.Name != change.Name {
			message = fmt.Sprintf("Notifier updated and renamed from '%s' to '%s'", storedNotifier.Name, change.Name)
		}

		return s.audit.record(ctx, actor, storedNotifier.WorkspaceID, message)
	})
	if err != nil {
		return WithSettings{}, err
	}

	updatedNotifier := storedNotifier
	updatedNotifier.Name = change.Name

	return WithSettings{Notifier: updatedNotifier, Settings: mergedSettings}, nil
}

// DeleteNotifier answers ErrNotifierInUse when a database still uses the notifier, which the
// reference of databases_notifiers enforces.
func (s *Service) DeleteNotifier(ctx context.Context, actor users.Actor, notifierID uuid.UUID) error {
	return s.transactor.Transaction(ctx, func(ctx context.Context) error {
		deletedNotifier, err := s.getAuthorized(ctx, actor, notifierID, workspaces.ActionEditContent)
		if err != nil {
			return err
		}

		err = s.repository.delete(ctx, notifierID)
		if database.IsForeignKeyViolation(err) {
			return ErrNotifierInUse
		}

		if err != nil {
			return err
		}

		return s.audit.record(ctx, actor, deletedNotifier.WorkspaceID, "Notifier deleted: "+deletedNotifier.Name)
	})
}

// GetNotifiers leaves out the IDs that match no notifier.
func (s *Service) GetNotifiers(ctx context.Context, notifierIDs []uuid.UUID) ([]Notifier, error) {
	if len(notifierIDs) == 0 {
		return []Notifier{}, nil
	}

	return s.repository.listByIDs(ctx, notifierIDs)
}

func (s *Service) GetNotifierTypes(ctx context.Context) ([]notification.NotifierType, error) {
	return s.repository.listTypes(ctx)
}

// ListForDelivery loads the notifiers with their settings, one query per provider type. It leaves
// out the IDs that match no notifier, such as one deleted after the notification was asked for.
func (s *Service) ListForDelivery(ctx context.Context, notifierIDs []uuid.UUID) ([]WithSettings, error) {
	foundNotifiers, err := s.GetNotifiers(ctx, notifierIDs)
	if err != nil {
		return nil, err
	}

	return s.loadSettings(ctx, foundNotifiers)
}

// RecordSendOutcome stores the error of the latest send, or clears it when sendErr is nil. It
// changes nothing else about the notifier.
func (s *Service) RecordSendOutcome(ctx context.Context, notifierID uuid.UUID, sendErr error) error {
	var lastSendError *string

	if sendErr != nil {
		errorText := sendErr.Error()
		lastSendError = &errorText
	}

	return s.repository.updateLastSendError(ctx, notifierID, lastSendError)
}

// OnWorkspaceDeleting deletes the notifiers of the workspace in the deletion's transaction. The
// databases of the workspace are gone by then, because their handler runs first.
func (s *Service) OnWorkspaceDeleting(ctx context.Context, event workspaces.WorkspaceDeleting) error {
	return s.repository.deleteByWorkspace(ctx, event.WorkspaceID)
}

// getAuthorized answers a notifier of a workspace the actor is not a member of exactly as a missing
// notifier, so its existence stays hidden.
func (s *Service) getAuthorized(
	ctx context.Context,
	actor users.Actor,
	notifierID uuid.UUID,
	action workspaces.Action,
) (Notifier, error) {
	foundNotifier, isFound, err := s.repository.get(ctx, notifierID)
	if err != nil {
		return Notifier{}, err
	}

	if !isFound {
		return Notifier{}, ErrNotifierNotFound
	}

	err = s.workspaces.Authorize(ctx, actor, foundNotifier.WorkspaceID, action)

	var problem *apierror.Problem
	if errors.As(err, &problem) && problem.Code == apierror.CodeNotFound {
		return Notifier{}, ErrNotifierNotFound
	}

	if err != nil {
		return Notifier{}, err
	}

	return foundNotifier, nil
}

// mergeAndValidate fills the secrets the incoming settings leave out from the stored notifier's
// settings, then validates the result.
func (s *Service) mergeAndValidate(
	ctx context.Context,
	storedNotifier Notifier,
	incomingSettings providers.Settings,
) (providers.Settings, error) {
	storedNotifierWithSettings, err := s.loadOneWithSettings(ctx, storedNotifier)
	if err != nil {
		return nil, err
	}

	provider, err := s.registry.GetProvider(storedNotifier.Type)
	if err != nil {
		return nil, err
	}

	mergedSettings, err := provider.MergeStoredSecrets(providers.SecretMerge{
		Incoming: incomingSettings,
		Stored:   storedNotifierWithSettings.Settings,
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

func (s *Service) loadOneWithSettings(ctx context.Context, foundNotifier Notifier) (WithSettings, error) {
	notifiersWithSettings, err := s.loadSettings(ctx, []Notifier{foundNotifier})
	if err != nil {
		return WithSettings{}, err
	}

	return notifiersWithSettings[0], nil
}

// loadSettings reads the settings of the notifiers with one query per provider type among them,
// keeping their order.
func (s *Service) loadSettings(ctx context.Context, notifiers []Notifier) ([]WithSettings, error) {
	notifierIDsByType := map[notification.NotifierType][]uuid.UUID{}
	for _, listedNotifier := range notifiers {
		notifierIDsByType[listedNotifier.Type] = append(notifierIDsByType[listedNotifier.Type], listedNotifier.ID)
	}

	settingsByNotifierID := map[uuid.UUID]providers.Settings{}

	for notifierType, notifierIDs := range notifierIDsByType {
		provider, err := s.registry.GetProvider(notifierType)
		if err != nil {
			return nil, err
		}

		settingsOfType, err := provider.GetRepository().ListByNotifierIDs(ctx, notifierIDs)
		if err != nil {
			return nil, err
		}

		maps.Copy(settingsByNotifierID, settingsOfType)
	}

	notifiersWithSettings := make([]WithSettings, 0, len(notifiers))
	for _, listedNotifier := range notifiers {
		settings, isFound := settingsByNotifierID[listedNotifier.ID]
		if !isFound {
			return nil, fmt.Errorf("%w: %s", errSettingsMissing, listedNotifier.ID)
		}

		notifiersWithSettings = append(
			notifiersWithSettings,
			WithSettings{Notifier: listedNotifier, Settings: settings},
		)
	}

	return notifiersWithSettings, nil
}

func validateName(name string) error {
	nameLength := utf8.RuneCountInString(name)
	if nameLength < 1 || nameLength > maxNotifierNameLength {
		return ErrInvalidName
	}

	return nil
}
