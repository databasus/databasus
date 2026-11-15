package httpapi

import (
	"bytes"
	"context"
	"mime"

	"databasus-backend/internal/modules/instancebackups/internal/keyexport"
	"databasus-backend/internal/modules/users"
)

const noStore = "no-store"

func (h *Handler) RequestMasterKeyConfirmation(
	ctx context.Context,
	_ RequestMasterKeyConfirmationRequestObject,
) (RequestMasterKeyConfirmationResponseObject, error) {
	actor, err := users.RequireActor(ctx)
	if err != nil {
		return nil, errNoActor
	}

	expiresAt, err := h.keyExport.RequestConfirmation(ctx, actor)
	if err != nil {
		return nil, err
	}

	return RequestMasterKeyConfirmation201JSONResponse{ExpiresAt: expiresAt}, nil
}

func (h *Handler) CreateMasterKeyExport(
	ctx context.Context,
	request CreateMasterKeyExportRequestObject,
) (CreateMasterKeyExportResponseObject, error) {
	actor, err := users.RequireActor(ctx)
	if err != nil {
		return nil, errNoActor
	}

	issuedExport, err := h.keyExport.CreateExport(ctx, actor, keyexport.ExportRequest{
		Proof:                          toIdentityProof(request.Body.Proof),
		IsInsecureTransferAcknowledged: isTrue(request.Body.IsInsecureTransferAcknowledged),
	})
	if err != nil {
		return nil, err
	}

	return CreateMasterKeyExport201JSONResponse{
		Token:     issuedExport.Token,
		FileName:  issuedExport.FileName,
		ExpiresAt: issuedExport.ExpiresAt,
	}, nil
}

func (h *Handler) DownloadMasterKey(
	ctx context.Context,
	request DownloadMasterKeyRequestObject,
) (DownloadMasterKeyResponseObject, error) {
	keyFile, err := h.keyExport.Download(ctx, request.Params.Token)
	if err != nil {
		return nil, err
	}

	contentDisposition := mime.FormatMediaType("attachment", map[string]string{"filename": keyFile.FileName})
	cacheControl := noStore

	return DownloadMasterKey200ApplicationoctetStreamResponse{
		Body: bytes.NewReader(keyFile.Content),
		Headers: DownloadMasterKey200ResponseHeaders{
			CacheControl:       &cacheControl,
			ContentDisposition: &contentDisposition,
		},
		ContentLength: int64(len(keyFile.Content)),
	}, nil
}

func (h *Handler) AcknowledgeMasterKey(
	ctx context.Context,
	request AcknowledgeMasterKeyRequestObject,
) (AcknowledgeMasterKeyResponseObject, error) {
	actor, err := users.RequireActor(ctx)
	if err != nil {
		return nil, errNoActor
	}

	if err := h.keyExport.Acknowledge(ctx, actor, request.Body.IsSaved); err != nil {
		return nil, err
	}

	return AcknowledgeMasterKey200JSONResponse{IsSaved: true}, nil
}

func toIdentityProof(proof IdentityProof) users.IdentityProof {
	return users.IdentityProof{Password: proof.Password, Code: proof.Code}
}

func isTrue(value *bool) bool {
	return value != nil && *value
}
