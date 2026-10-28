package httpapi

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"databasus-backend/internal/lib/httpserver"
	"databasus-backend/internal/modules/backups/internal/physical"
)

func (h *Handler) CreatePhysicalRestoreToken(
	ctx context.Context,
	request CreatePhysicalRestoreTokenRequestObject,
) (CreatePhysicalRestoreTokenResponseObject, error) {
	ctx = httpserver.GetRequestContext(ctx)

	actor, err := getActor(ctx)
	if err != nil {
		return nil, err
	}

	subject, err := toRestoreTokenSubject(*request.Body)
	if err != nil {
		return nil, err
	}

	issuedToken, err := h.physicalBackups.IssueRestoreToken(ctx, actor, subject)
	if err != nil {
		return nil, err
	}

	return CreatePhysicalRestoreToken201JSONResponse{Token: issuedToken.Token, ExpiresAt: issuedToken.ExpiresAt}, nil
}

func toRestoreTokenSubject(body PhysicalRestoreTokenCreate) (physical.RestoreTokenSubject, error) {
	variant, err := body.ValueByDiscriminator()
	if err != nil {
		return physical.RestoreTokenSubject{}, errUnknownRestoreTokenSubject
	}

	switch subject := variant.(type) {
	case PhysicalPointInTimeRestoreTokenCreate:
		return physical.RestoreTokenSubject{DatabaseID: subject.DatabaseId, TargetTime: subject.TargetTime}, nil
	case PhysicalBackupRestoreTokenCreate:
		return physical.RestoreTokenSubject{BackupID: &subject.BackupId}, nil
	default:
		return physical.RestoreTokenSubject{}, errUnknownRestoreTokenSubject
	}
}

func (h *Handler) DownloadPhysicalRestoreSet(
	ctx context.Context,
	request DownloadPhysicalRestoreSetRequestObject,
) (DownloadPhysicalRestoreSetResponseObject, error) {
	ctx = httpserver.GetRequestContext(ctx)

	stream, err := h.physicalBackups.OpenRestoreStream(ctx, request.Params.Token)
	if err != nil {
		return nil, err
	}

	return physicalRestoreSetResponse{stream: stream}, nil
}

type physicalRestoreSetResponse struct{ stream physical.RestoreStream }

func (response physicalRestoreSetResponse) VisitDownloadPhysicalRestoreSetResponse(writer http.ResponseWriter) error {
	defer response.stream.Close()

	writer.Header().Set("Content-Type", "application/x-tar")
	writer.Header().
		Set("Content-Disposition", fmt.Sprintf(`attachment; filename="restore-%s.tar"`, response.stream.DatabaseID))
	writer.WriteHeader(http.StatusOK)

	if err := response.stream.Write(writer); err != nil {
		panic(http.ErrAbortHandler)
	}

	return nil
}

func (h *Handler) GetRecoveryScript(
	_ context.Context,
	_ GetRecoveryScriptRequestObject,
) (GetRecoveryScriptResponseObject, error) {
	script := h.physicalBackups.GetRecoveryScript()
	disposition := `inline; filename="databasus-recovery.sh"`

	return GetRecoveryScript200TextxShellscriptResponse{
		Body:          io.NopCloser(strings.NewReader(script)),
		Headers:       GetRecoveryScript200ResponseHeaders{ContentDisposition: &disposition},
		ContentLength: int64(len(script)),
	}, nil
}
