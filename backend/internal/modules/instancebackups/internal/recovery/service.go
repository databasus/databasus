package recovery

import (
	"context"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/apierror"
	"databasus-backend/internal/lib/audit"
	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/lib/lifecycle"
	"databasus-backend/internal/lib/recoverypause"
	"databasus-backend/internal/modules/instancebackups/internal/configs"
	"databasus-backend/internal/modules/users"
)

type ServiceDeps struct {
	Journal        *JournalStore
	Clock          clock.Source
	Transactor     *database.Transactor
	AuditLog       *audit.Log
	Users          users.Module
	Configs        *configs.Service
	Pause          *recoverypause.Gate
	RunningRelease string
	// RebuildRequest asks the server process to publish an accepted apply in place.
	RebuildRequest *lifecycle.RebuildRequest
}

// ApplyRequest is root admin's confirmation that the installation is replaced, with the identity
// proof bound to the recovery.
type ApplyRequest struct {
	IsReplacementConfirmed bool
	Proof                  users.IdentityProof
}

// Service shows root admin the recoveries of the local journal, accepts the apply of a prepared one
// and ends the recovery pause.
type Service struct {
	journal        *JournalStore
	clock          clock.Source
	transactor     *database.Transactor
	auditLog       *audit.Log
	users          users.Module
	configs        *configs.Service
	pause          *recoverypause.Gate
	runningRelease string
	rebuildRequest *lifecycle.RebuildRequest
}

func NewService(deps ServiceDeps) *Service {
	return &Service{
		journal:        deps.Journal,
		clock:          deps.Clock,
		transactor:     deps.Transactor,
		auditLog:       deps.AuditLog,
		users:          deps.Users,
		configs:        deps.Configs,
		pause:          deps.Pause,
		runningRelease: deps.RunningRelease,
		rebuildRequest: deps.RebuildRequest,
	}
}

func (s *Service) GetRecovery(ctx context.Context, actor users.Actor, recoveryID uuid.UUID) (Recovery, error) {
	if _, err := s.users.RequireRootAdmin(ctx, actor); err != nil {
		return Recovery{}, err
	}

	journal, err := s.journal.Read()
	if err != nil {
		return Recovery{}, err
	}

	pausedBy, err := s.configs.GetRecoveryPause(ctx)
	if err != nil {
		return Recovery{}, err
	}

	foundRecovery, isFound := findRecovery(journal, recoveryID, pausedBy, s.runningRelease)
	if !isFound {
		return Recovery{}, ErrRecoveryNotFound
	}

	return foundRecovery, nil
}

// GetLatestRecovery returns the operation in progress, else the latest completed one, else nil.
// It does not check access, for the status of a caller that already did.
func (s *Service) GetLatestRecovery(ctx context.Context) (*Recovery, error) {
	journal, err := s.journal.Read()
	if err != nil {
		return nil, err
	}

	if journal.Operation != nil {
		recovery := newOperationRecovery(*journal.Operation, s.runningRelease)

		return &recovery, nil
	}

	if journal.LastCompleted == nil {
		return nil, nil
	}

	pausedBy, err := s.configs.GetRecoveryPause(ctx)
	if err != nil {
		return nil, err
	}

	recovery := newCompletedRecovery(*journal.LastCompleted, pausedBy, s.runningRelease)

	return &recovery, nil
}

// Activate ends the pause after root admin confirmed that the old instance is stopped. Schedules
// and outbound work resume; recovered local storages stay unavailable until validated.
func (s *Service) Activate(
	ctx context.Context,
	actor users.Actor,
	recoveryID uuid.UUID,
	isOldInstanceStopped bool,
) (Recovery, error) {
	rootAdmin, err := s.users.RequireRootAdmin(ctx, actor)
	if err != nil {
		return Recovery{}, err
	}

	if !isOldInstanceStopped {
		return Recovery{}, apierror.NewInvalidFieldsProblem([]apierror.FieldError{
			{Field: "isOldInstanceStopped", Detail: "confirm that the old instance is stopped"},
		})
	}

	if _, err := s.GetRecovery(ctx, actor, recoveryID); err != nil {
		return Recovery{}, err
	}

	err = s.transactor.Transaction(ctx, func(txCtx context.Context) error {
		isEnded, err := s.configs.EndRecoveryPause(txCtx, recoveryID)
		if err != nil {
			return err
		}

		if !isEnded {
			return ErrRecoveryNotPaused
		}

		return s.auditLog.Record(txCtx, audit.Event{
			Author:      rootAdmin.GetAuditAuthor(),
			WorkspaceID: nil,
			Message:     "Root admin confirmed the old instance is stopped and activated the recovered instance",
		})
	})
	if err != nil {
		return Recovery{}, err
	}

	s.pause.Resume()

	return s.GetRecovery(ctx, actor, recoveryID)
}

// RequestConfirmation emails root admin a code that confirms applying this prepared recovery in the
// current session.
func (s *Service) RequestConfirmation(ctx context.Context, actor users.Actor, recoveryID uuid.UUID) (time.Time, error) {
	if _, err := s.users.RequireRootAdmin(ctx, actor); err != nil {
		return time.Time{}, err
	}

	journal, err := s.journal.Read()
	if err != nil {
		return time.Time{}, err
	}

	operation, err := s.getApplicable(journal, recoveryID)
	if err != nil {
		return time.Time{}, err
	}

	return s.users.SendIdentityCode(ctx, actor, getApplyBinding(operation))
}

// Apply records root admin's accepted apply durably before answering, so publication proceeds even
// if the browser disconnects or the process stops. A repeated apply of an accepted recovery
// returns its state without checking another proof.
func (s *Service) Apply(
	ctx context.Context,
	actor users.Actor,
	recoveryID uuid.UUID,
	request ApplyRequest,
) (Recovery, error) {
	rootAdmin, err := s.users.RequireRootAdmin(ctx, actor)
	if err != nil {
		return Recovery{}, err
	}

	if !request.IsReplacementConfirmed {
		return Recovery{}, apierror.NewInvalidFieldsProblem([]apierror.FieldError{{
			Field:  "isReplacementConfirmed",
			Detail: "confirm that current users, settings and the backup catalog are replaced",
		}})
	}

	journal, err := s.journal.Read()
	if err != nil {
		return Recovery{}, err
	}

	if isApplyAccepted(journal, recoveryID) {
		return s.GetRecovery(ctx, actor, recoveryID)
	}

	operation, err := s.getApplicable(journal, recoveryID)
	if err != nil {
		return Recovery{}, err
	}

	if err := s.users.VerifyIdentityProof(ctx, actor, getApplyBinding(operation), request.Proof); err != nil {
		return Recovery{}, err
	}

	acceptedOperation, err := s.acceptApply(ctx, rootAdmin, operation)
	if err != nil {
		return Recovery{}, err
	}

	s.rebuildRequest.Request()

	return newOperationRecovery(acceptedOperation, s.runningRelease), nil
}

// acceptApply audits the apply and marks it accepted in the journal in one step, rechecking under
// the journal lock that the recovery is still prepared and is the one the proof confirmed.
func (s *Service) acceptApply(ctx context.Context, rootAdmin users.User, confirmed Operation) (Operation, error) {
	var acceptedOperation Operation

	err := s.transactor.Transaction(ctx, func(txCtx context.Context) error {
		err := s.auditLog.Record(txCtx, audit.Event{
			Author:      rootAdmin.GetAuditAuthor(),
			WorkspaceID: nil,
			Message:     "Root admin confirmed replacing this installation with a recovered Databasus backup",
		})
		if err != nil {
			return err
		}

		unlock, err := s.journal.Lock()
		if err != nil {
			return err
		}
		defer unlock()

		journal, err := s.journal.Read()
		if err != nil {
			return err
		}

		operation, err := s.getApplicable(journal, confirmed.ID)
		if err != nil {
			return err
		}

		if operation.ArchiveSha256 != confirmed.ArchiveSha256 {
			return ErrRecoveryNotReady
		}

		journal.Operation.IsApplyAccepted = true
		acceptedOperation = *journal.Operation

		return s.journal.Write(journal)
	})

	return acceptedOperation, err
}

// getApplicable returns the operation while it is prepared, not yet accepted and not expired.
func (s *Service) getApplicable(journal Journal, recoveryID uuid.UUID) (Operation, error) {
	operation := journal.Operation
	isCurrent := operation != nil && operation.ID == recoveryID

	if isCurrent && operation.Phase == PhaseImported && !operation.IsApplyAccepted &&
		operation.ExpiresAt != nil && s.clock.Now().Before(*operation.ExpiresAt) {
		return *operation, nil
	}

	if _, isFound := findRecovery(journal, recoveryID, nil, s.runningRelease); isFound {
		return Operation{}, ErrRecoveryNotReady
	}

	return Operation{}, ErrRecoveryNotFound
}

// isApplyAccepted is true for the operation whose apply was accepted and for a recovery that
// already completed.
func isApplyAccepted(journal Journal, recoveryID uuid.UUID) bool {
	if journal.Operation != nil && journal.Operation.ID == recoveryID {
		return journal.IsPublicationUnfinished()
	}

	return journal.LastCompleted != nil && journal.LastCompleted.OperationID == recoveryID
}

// getApplyBinding binds a proof to the operation and its archive's bytes, so a proof for one
// prepared recovery cannot apply another.
func getApplyBinding(operation Operation) users.IdentityProofBinding {
	return users.IdentityProofBinding{
		Purpose: users.IdentityProofPurposeInstanceRecovery,
		Subject: operation.ID.String() + ":" + operation.ArchiveSha256,
	}
}
