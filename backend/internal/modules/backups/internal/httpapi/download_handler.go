package httpapi

import (
	"context"
	"mime"

	"databasus-backend/internal/modules/users"
)

func (h *Handler) CreateDownloadToken(
	ctx context.Context,
	request CreateDownloadTokenRequestObject,
) (CreateDownloadTokenResponseObject, error) {
	actor, err := users.RequireActor(ctx)
	if err != nil {
		return nil, err
	}

	issuedToken, err := h.downloads.CreateToken(ctx, actor, request.Body.BackupId)
	if err != nil {
		return nil, err
	}

	return CreateDownloadToken201JSONResponse{
		Token:     issuedToken.Token,
		FileName:  issuedToken.FileName,
		ExpiresAt: issuedToken.ExpiresAt,
	}, nil
}

// DownloadLogicalBackup hands the opened file to the generated response, which closes it after the
// copy and so releases the user's stream.
func (h *Handler) DownloadLogicalBackup(
	ctx context.Context,
	request DownloadLogicalBackupRequestObject,
) (DownloadLogicalBackupResponseObject, error) {
	openedDownload, err := h.downloads.OpenDownload(ctx, request.BackupId, request.Params.Token)
	if err != nil {
		return nil, err
	}

	contentDisposition := mime.FormatMediaType("attachment", map[string]string{"filename": openedDownload.FileName})

	return DownloadLogicalBackup200ApplicationoctetStreamResponse{
		Body:          openedDownload.File,
		Headers:       DownloadLogicalBackup200ResponseHeaders{ContentDisposition: &contentDisposition},
		ContentLength: openedDownload.SizeBytes,
	}, nil
}
