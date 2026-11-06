package agentapi

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"

	"github.com/gin-gonic/gin"

	"databasus-backend/internal/lib/apierror"
)

func GetReportValidationMiddleware(hooks *apierror.Hooks) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if !strings.HasSuffix(ctx.FullPath(), "/report") {
			ctx.Next()
			return
		}

		body, err := io.ReadAll(ctx.Request.Body)
		if err != nil {
			hooks.HandleRequestError(ctx, err)
			ctx.Abort()
			return
		}

		var fields map[string]json.RawMessage
		if err := json.Unmarshal(body, &fields); err != nil {
			hooks.HandleRequestError(ctx, err)
			ctx.Abort()
			return
		}

		if diagnosticJSON, hasDiagnostic := fields["failureDiagnostic"]; hasDiagnostic {
			var diagnostic struct {
				Output      *string `json:"output"`
				IsTruncated *bool   `json:"isTruncated"`
			}
			if err := json.Unmarshal(diagnosticJSON, &diagnostic); err != nil {
				hooks.HandleRequestError(ctx, err)
				ctx.Abort()
				return
			}

			if diagnostic.Output == nil || diagnostic.IsTruncated == nil {
				hooks.HandleHandlerError(ctx, apierror.NewInvalidFieldsProblem([]apierror.FieldError{
					{
						Field:  "failureDiagnostic",
						Detail: "must be an object with non-null output and isTruncated fields",
					},
				}))
				ctx.Abort()
				return
			}
		}

		ctx.Request.Body = io.NopCloser(bytes.NewReader(body))
		ctx.Next()
	}
}
