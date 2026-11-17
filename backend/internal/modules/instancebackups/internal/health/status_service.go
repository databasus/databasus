package health

import (
	"context"

	"databasus-backend/internal/modules/users"
)

func (s *Service) GetStatus(ctx context.Context, actor users.Actor) (Status, error) {
	rootAdmin, err := s.users.RequireRootAdmin(ctx, actor)
	if err != nil {
		return Status{}, err
	}

	protection, err := s.assess(ctx, s.clock.Now())
	if err != nil {
		return Status{}, err
	}

	incident, err := s.getOpenIncident(ctx, rootAdmin)
	if err != nil {
		return Status{}, err
	}

	reminder, err := s.configs.GetReminder(ctx)
	if err != nil {
		return Status{}, err
	}

	isAcknowledged, err := s.keyExport.IsAcknowledged(ctx)
	if err != nil {
		return Status{}, err
	}

	isExportCodeRequired, err := s.keyExport.IsExportCodeRequired(ctx)
	if err != nil {
		return Status{}, err
	}

	methods, err := s.users.GetIdentityProofMethods(ctx, actor)
	if err != nil {
		return Status{}, err
	}

	latestRecovery, err := s.recoveries.GetLatestRecovery(ctx)
	if err != nil {
		return Status{}, err
	}

	return Status{
		Health:                        protection.Health,
		Destinations:                  protection.Destinations,
		Incident:                      incident,
		IsEmailConfigured:             s.mailer.IsConfigured(),
		IsRootEmailUsable:             isEmailAddress(rootAdmin.Email),
		IsMasterKeyAcknowledged:       isAcknowledged,
		IsMasterKeyExportCodeRequired: isExportCodeRequired,
		IdentityProofMethods:          methods,
		MasterKeyPath:                 s.masterKeyPath,
		Reminder:                      reminder,
		IsRecoveryPaused:              protection.Config.IsRecoveryPaused(),
		Recovery:                      latestRecovery,
	}, nil
}

func (s *Service) getOpenIncident(ctx context.Context, rootAdmin users.User) (*IncidentView, error) {
	incident, isOpen, err := s.incidents.getOpen(ctx)
	if err != nil || !isOpen {
		return nil, err
	}

	return &IncidentView{Incident: incident, MailStatus: incident.getMailStatus(s.isMailEligible(rootAdmin))}, nil
}
