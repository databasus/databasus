package systemhttpapi

import "context"

var _ StrictServerInterface = (*Handler)(nil)

type Deps struct {
	Version string
}

type Handler struct {
	version string
}

func NewHandler(deps Deps) *Handler {
	return &Handler{version: deps.Version}
}

func (h *Handler) GetVersion(context.Context, GetVersionRequestObject) (GetVersionResponseObject, error) {
	return GetVersion200JSONResponse{Version: h.version}, nil
}
