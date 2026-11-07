package systemhttpapi

import (
	"context"

	"databasus-backend/internal/lib/httpserver"
)

var _ StrictServerInterface = (*Handler)(nil)

type Handler struct{ service *Service }

func NewHandler(deps Deps) *Handler { return &Handler{service: NewService(deps)} }

func (h *Handler) GetVersion(ctx context.Context, _ GetVersionRequestObject) (GetVersionResponseObject, error) {
	ctx = httpserver.GetRequestContext(ctx)

	return h.service.GetVersion(ctx)
}

func (h *Handler) GetInstanceHealth(
	ctx context.Context,
	_ GetInstanceHealthRequestObject,
) (GetInstanceHealthResponseObject, error) {
	ctx = httpserver.GetRequestContext(ctx)

	return h.service.GetHealth(ctx)
}

func (h *Handler) GetDiskUsage(ctx context.Context, _ GetDiskUsageRequestObject) (GetDiskUsageResponseObject, error) {
	ctx = httpserver.GetRequestContext(ctx)

	return h.service.GetDiskUsage(ctx)
}

func (h *Handler) DownloadVerificationAgent(
	ctx context.Context,
	request DownloadVerificationAgentRequestObject,
) (DownloadVerificationAgentResponseObject, error) {
	ctx = httpserver.GetRequestContext(ctx)

	return h.service.DownloadAgent(ctx, request.Params.Arch)
}
