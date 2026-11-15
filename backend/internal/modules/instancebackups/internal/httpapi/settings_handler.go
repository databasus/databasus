package httpapi

import (
	"context"

	"databasus-backend/internal/lib/schedule"
	"databasus-backend/internal/modules/instancebackups/internal/configs"
	"databasus-backend/internal/modules/users"
)

func (h *Handler) GetInstanceBackupSettings(
	ctx context.Context,
	_ GetInstanceBackupSettingsRequestObject,
) (GetInstanceBackupSettingsResponseObject, error) {
	actor, err := users.RequireActor(ctx)
	if err != nil {
		return nil, errNoActor
	}

	settings, err := h.configs.GetSettings(ctx, actor)
	if err != nil {
		return nil, err
	}

	return GetInstanceBackupSettings200JSONResponse(fromSettings(settings)), nil
}

func (h *Handler) UpdateInstanceBackupSettings(
	ctx context.Context,
	request UpdateInstanceBackupSettingsRequestObject,
) (UpdateInstanceBackupSettingsResponseObject, error) {
	actor, err := users.RequireActor(ctx)
	if err != nil {
		return nil, errNoActor
	}

	settings, err := h.configs.UpdateSettings(ctx, actor, toSettings(*request.Body))
	if err != nil {
		return nil, err
	}

	return UpdateInstanceBackupSettings200JSONResponse(fromSettings(settings)), nil
}

func (h *Handler) ListInstanceBackupDestinations(
	ctx context.Context,
	request ListInstanceBackupDestinationsRequestObject,
) (ListInstanceBackupDestinationsResponseObject, error) {
	actor, err := users.RequireActor(ctx)
	if err != nil {
		return nil, errNoActor
	}

	limit := 20
	if request.Params.Limit != nil {
		limit = *request.Params.Limit
	}

	page, err := h.configs.ListDestinationChoices(ctx, actor, configs.DestinationChoiceQuery{
		Limit: limit,
		After: request.Params.After,
		Query: request.Params.Query,
	})
	if err != nil {
		return nil, err
	}

	items := make([]InstanceBackupDestinationChoice, 0, len(page.Items))
	for _, choice := range page.Items {
		items = append(items, InstanceBackupDestinationChoice{
			StorageId:     choice.StorageID,
			StorageName:   choice.StorageName,
			StorageType:   StorageType(choice.StorageType),
			WorkspaceId:   choice.WorkspaceID,
			WorkspaceName: choice.WorkspaceName,
			IsSelected:    choice.IsSelected,
		})
	}

	return ListInstanceBackupDestinations200JSONResponse{Items: items, NextCursor: page.NextCursor}, nil
}

func (h *Handler) UpdateInstanceBackupReminder(
	ctx context.Context,
	request UpdateInstanceBackupReminderRequestObject,
) (UpdateInstanceBackupReminderResponseObject, error) {
	actor, err := users.RequireActor(ctx)
	if err != nil {
		return nil, errNoActor
	}

	reminder, err := h.configs.UpdateSetupReminder(ctx, actor, request.Body.IsDeferred)
	if err != nil {
		return nil, err
	}

	return UpdateInstanceBackupReminder200JSONResponse{
		IsEligible:    reminder.IsEligible,
		DeferredUntil: reminder.DeferredUntil,
	}, nil
}

func fromSettings(settings configs.Settings) InstanceBackupSettings {
	return InstanceBackupSettings{
		Schedule: Schedule{
			Type:           ScheduleType(settings.Schedule.Type),
			TimeOfDay:      settings.Schedule.TimeOfDay,
			Weekday:        settings.Schedule.Weekday,
			DayOfMonth:     settings.Schedule.DayOfMonth,
			CronExpression: settings.Schedule.CronExpression,
		},
		RetentionDays: settings.RetentionDays,
		StorageIds:    settings.StorageIDs,
	}
}

func toSettings(body InstanceBackupSettings) configs.Settings {
	return configs.Settings{
		Schedule: schedule.Schedule{
			Type:           schedule.Type(body.Schedule.Type),
			TimeOfDay:      body.Schedule.TimeOfDay,
			Weekday:        body.Schedule.Weekday,
			DayOfMonth:     body.Schedule.DayOfMonth,
			CronExpression: body.Schedule.CronExpression,
		},
		RetentionDays: body.RetentionDays,
		StorageIDs:    body.StorageIds,
	}
}
