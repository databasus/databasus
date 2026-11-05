package agentapi

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"

	"github.com/gin-gonic/gin"

	"databasus-backend/internal/lib/apierror"
)

func GetHeartbeatValidationMiddleware(hooks *apierror.Hooks) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if !strings.HasSuffix(ctx.FullPath(), "/heartbeat") {
			ctx.Next()
			return
		}

		body, err := io.ReadAll(ctx.Request.Body)
		if err != nil {
			hooks.HandleRequestError(ctx, err)
			return
		}
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(body, &fields); err != nil {
			hooks.HandleRequestError(ctx, err)
			return
		}

		memory, hasMemory := fields["maxRamMb"]
		if !hasMemory || bytes.Equal(bytes.TrimSpace(memory), []byte("null")) {
			hooks.HandleHandlerError(
				ctx,
				apierror.NewInvalidFieldsProblem(
					[]apierror.FieldError{{Field: "maxRamMb", Detail: "is required and must not be null"}},
				),
			)
			return
		}

		ctx.Request.Body = io.NopCloser(bytes.NewReader(body))
		ctx.Next()
	}
}
