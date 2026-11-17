package httpapi

import (
	"context"

	"databasus-backend/internal/modules/instancebackups/internal/health"
	"databasus-backend/internal/modules/users"
)

func (h *Handler) GetInstanceBackupStatus(
	ctx context.Context,
	_ GetInstanceBackupStatusRequestObject,
) (GetInstanceBackupStatusResponseObject, error) {
	actor, err := users.RequireActor(ctx)
	if err != nil {
		return nil, errNoActor
	}

	status, err := h.status.GetStatus(ctx, actor)
	if err != nil {
		return nil, err
	}

	return GetInstanceBackupStatus200JSONResponse(fromStatus(status)), nil
}

func fromStatus(status health.Status) InstanceBackupStatus {
	destinations := make([]InstanceBackupDestinationHealth, 0, len(status.Destinations))
	for _, destination := range status.Destinations {
		storageType := StorageType(destination.StorageType)
		destinations = append(destinations, InstanceBackupDestinationHealth{
			StorageId:                destination.StorageID,
			StorageName:              destination.StorageName,
			StorageType:              &storageType,
			WorkspaceId:              destination.WorkspaceID,
			WorkspaceName:            destination.WorkspaceName,
			Health:                   InstanceBackupHealth(destination.Health),
			LastSuccessfulSnapshotAt: destination.LastSuccessfulSnapshotAt,
			OverdueAt:                destination.OverdueAt,
			LastError:                destination.LastError,
		})
	}

	methods := make([]IdentityProofMethod, 0, len(status.IdentityProofMethods))
	for _, method := range status.IdentityProofMethods {
		methods = append(methods, IdentityProofMethod(method))
	}

	var incident *InstanceBackupIncident
	if status.Incident != nil {
		incident = &InstanceBackupIncident{
			Id:            status.Incident.Incident.ID,
			Kind:          InstanceBackupIncidentKind(status.Incident.Incident.Kind),
			StartedAt:     status.Incident.Incident.StartedAt,
			MailStatus:    InstanceBackupMailStatus(status.Incident.MailStatus),
			LastMailedAt:  status.Incident.Incident.LastMailedAt,
			LastMailError: status.Incident.Incident.LastMailError,
		}
	}

	var latestRecovery *InstanceRecovery
	if status.Recovery != nil {
		converted := fromRecovery(*status.Recovery)
		latestRecovery = &converted
	}

	return InstanceBackupStatus{
		Health:                        InstanceBackupHealth(status.Health),
		Destinations:                  destinations,
		Incident:                      incident,
		IsEmailConfigured:             status.IsEmailConfigured,
		IsRootEmailUsable:             status.IsRootEmailUsable,
		IsMasterKeyAcknowledged:       status.IsMasterKeyAcknowledged,
		IsMasterKeyExportCodeRequired: status.IsMasterKeyExportCodeRequired,
		IdentityProofMethods:          methods,
		MasterKeyPath:                 status.MasterKeyPath,
		Reminder: InstanceBackupReminder{
			IsEligible: status.Reminder.IsEligible,
		},
		IsRecoveryPaused: status.IsRecoveryPaused,
		Recovery:         latestRecovery,
	}
}
