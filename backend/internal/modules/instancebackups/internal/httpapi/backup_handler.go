package httpapi

import (
	"context"
	"mime"

	"databasus-backend/internal/modules/instancebackups/internal/catalog"
	"databasus-backend/internal/modules/users"
)

const defaultHistoryLimit = 20

func (h *Handler) StartInstanceBackup(
	ctx context.Context,
	_ StartInstanceBackupRequestObject,
) (StartInstanceBackupResponseObject, error) {
	actor, err := users.RequireActor(ctx)
	if err != nil {
		return nil, errNoActor
	}

	backup, err := h.backups.StartBackup(ctx, actor)
	if err != nil {
		return nil, err
	}

	return StartInstanceBackup202JSONResponse(toInstanceBackup(backup)), nil
}

func (h *Handler) ListInstanceBackups(
	ctx context.Context,
	request ListInstanceBackupsRequestObject,
) (ListInstanceBackupsResponseObject, error) {
	actor, err := users.RequireActor(ctx)
	if err != nil {
		return nil, errNoActor
	}

	limit := defaultHistoryLimit
	if request.Params.Limit != nil {
		limit = *request.Params.Limit
	}

	page, err := h.backups.ListBackups(ctx, actor, catalog.HistoryQuery{Limit: limit, After: request.Params.After})
	if err != nil {
		return nil, err
	}

	items := make([]InstanceBackup, 0, len(page.Items))
	for _, backup := range page.Items {
		items = append(items, toInstanceBackup(backup))
	}

	return ListInstanceBackups200JSONResponse{Items: items, NextCursor: page.NextCursor}, nil
}

func (h *Handler) GetInstanceBackup(
	ctx context.Context,
	request GetInstanceBackupRequestObject,
) (GetInstanceBackupResponseObject, error) {
	actor, err := users.RequireActor(ctx)
	if err != nil {
		return nil, errNoActor
	}

	backup, err := h.backups.GetBackup(ctx, actor, request.InstanceBackupId)
	if err != nil {
		return nil, err
	}

	return GetInstanceBackup200JSONResponse(toInstanceBackup(backup)), nil
}

func (h *Handler) CreateInstanceBackupDownloadToken(
	ctx context.Context,
	request CreateInstanceBackupDownloadTokenRequestObject,
) (CreateInstanceBackupDownloadTokenResponseObject, error) {
	actor, err := users.RequireActor(ctx)
	if err != nil {
		return nil, errNoActor
	}

	issued, err := h.backups.CreateDownloadToken(ctx, actor, request.InstanceBackupId, request.Body.CopyId)
	if err != nil {
		return nil, err
	}

	return CreateInstanceBackupDownloadToken201JSONResponse{
		Token:     issued.Token,
		FileName:  issued.FileName,
		ExpiresAt: issued.ExpiresAt,
	}, nil
}

func (h *Handler) DownloadInstanceBackup(
	ctx context.Context,
	request DownloadInstanceBackupRequestObject,
) (DownloadInstanceBackupResponseObject, error) {
	download, err := h.backups.OpenDownload(ctx, request.Params.Token)
	if err != nil {
		return nil, err
	}

	contentDisposition := mime.FormatMediaType("attachment", map[string]string{"filename": download.FileName})
	cacheControl := noStore

	return DownloadInstanceBackup200ApplicationoctetStreamResponse{
		Body: download.File,
		Headers: DownloadInstanceBackup200ResponseHeaders{
			CacheControl:       &cacheControl,
			ContentDisposition: &contentDisposition,
		},
		ContentLength: download.SizeBytes,
	}, nil
}

func toInstanceBackup(backup catalog.BackupWithCopies) InstanceBackup {
	copies := make([]InstanceBackupCopy, 0, len(backup.Copies))
	for _, backupCopy := range backup.Copies {
		copies = append(copies, InstanceBackupCopy{
			Id:             backupCopy.ID,
			StorageId:      backupCopy.StorageID,
			StorageName:    backupCopy.StorageName,
			WorkspaceName:  backupCopy.WorkspaceName,
			Status:         InstanceBackupCopyStatus(backupCopy.Status),
			AttemptCount:   backupCopy.AttemptCount,
			StoredAt:       backupCopy.StoredAt,
			LastError:      backupCopy.LastError,
			IsDownloadable: backupCopy.IsDownloadable(),
		})
	}

	return InstanceBackup{
		Id:            backup.Backup.ID,
		Status:        InstanceBackupStatusValue(backup.Backup.Status),
		Trigger:       InstanceBackupTrigger(backup.Backup.Trigger),
		SourceRelease: backup.Backup.SourceRelease,
		StartedAt:     backup.Backup.StartedAt,
		SnapshotAt:    backup.Backup.SnapshotAt,
		CompletedAt:   backup.Backup.CompletedAt,
		SizeBytes:     backup.Backup.SizeBytes,
		FailureReason: backup.Backup.FailureReason,
		Copies:        copies,
	}
}
