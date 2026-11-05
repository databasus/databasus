package apierror

import (
	"errors"
	"log/slog"

	"github.com/gin-gonic/gin"
)

type Deps struct {
	Logger *slog.Logger
}

// Hooks are the error handlers of a generated Gin server: the request, handler and response error
// handlers of the strict server and the parameter error handler of its wrapper. Without them the
// generated code answers its own {"msg": ...} body with the error's text.
type Hooks struct {
	logger *slog.Logger
}

func NewHooks(deps Deps) *Hooks {
	return &Hooks{logger: deps.Logger}
}

// HandleRequestError answers a request body the operation cannot decode.
func (h *Hooks) HandleRequestError(ctx *gin.Context, err error) {
	h.writeInvalidRequest(ctx, err)
}

// HandleParameterError answers a malformed path, query or header parameter. The status the
// wrapper proposes is ignored, because the contract answers every such request with 400.
func (h *Hooks) HandleParameterError(ctx *gin.Context, err error, _ int) {
	h.writeInvalidRequest(ctx, err)
}

// HandleHandlerError answers a Problem the service returned as it is, and anything else as 500.
func (h *Hooks) HandleHandlerError(ctx *gin.Context, err error) {
	var problem *Problem
	if errors.As(err, &problem) {
		Write(ctx, problem)

		return
	}

	h.writeInternal(ctx, err)
}

// HandleResponseError answers a response the strict server could not encode.
func (h *Hooks) HandleResponseError(ctx *gin.Context, err error) {
	h.writeInternal(ctx, err)
}

func (h *Hooks) writeInvalidRequest(ctx *gin.Context, err error) {
	h.logger.InfoContext(ctx.Request.Context(), "request breaks the contract",
		slog.String("method", ctx.Request.Method),
		slog.String("path", ctx.Request.URL.Path),
		slog.Any("error", err),
	)

	Write(ctx, &Problem{Code: CodeInvalidRequest, Detail: err.Error(), Errors: nil})
}

func (h *Hooks) writeInternal(ctx *gin.Context, err error) {
	h.logger.ErrorContext(ctx.Request.Context(), "contract operation failed",
		slog.String("method", ctx.Request.Method),
		slog.String("path", ctx.Request.URL.Path),
		slog.Any("error", err),
	)

	Write(ctx, NewInternalProblem())
}
