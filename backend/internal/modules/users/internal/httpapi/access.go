package httpapi

import (
	"errors"
	"log/slog"
	"strings"

	"github.com/gin-gonic/gin"

	"databasus-backend/internal/lib/apierror"
	"databasus-backend/internal/lib/httpserver"
	"databasus-backend/internal/lib/logging"
	"databasus-backend/internal/modules/users/internal/identity"
	"databasus-backend/internal/modules/users/internal/signin"
)

const bearerPrefix = "Bearer "

type Access struct {
	signIn *signin.Service
	logger *slog.Logger
}

func NewAccess(signIn *signin.Service, logger *slog.Logger) *Access {
	return &Access{signIn: signIn, logger: logger}
}

// GetAccessMiddleware enforces the access mark that the contract guard put into the request
// context. A public operation runs without a session and ignores a token; any other needs a valid
// session, and an admin operation also the admin role.
func (a *Access) GetAccessMiddleware() gin.HandlerFunc {
	return func(ginContext *gin.Context) {
		mark := httpserver.GetAccessMark(ginContext.Request.Context())
		if mark == httpserver.AccessMarkPublic {
			ginContext.Next()

			return
		}

		actor, isSignedIn := a.signInRequest(ginContext)
		if !isSignedIn {
			return
		}

		if mark == httpserver.AccessMarkAdmin && !actor.IsAdmin() {
			a.logger.WarnContext(ginContext.Request.Context(), "rejected a request from a user who is not an admin",
				slog.String("client_ip", ginContext.ClientIP()))

			apierror.Write(ginContext, errNotAdmin)

			return
		}

		ginContext.Next()
	}
}

// signInRequest puts the actor of the request's session into its context, or answers the request
// and reports false.
func (a *Access) signInRequest(ginContext *gin.Context) (identity.Actor, bool) {
	ctx := ginContext.Request.Context()

	token := strings.TrimPrefix(ginContext.GetHeader("Authorization"), bearerPrefix)
	if token == "" {
		a.logger.WarnContext(ctx, "rejected a request without a session token",
			slog.String("client_ip", ginContext.ClientIP()))

		apierror.Write(ginContext, errUnauthenticated)

		return identity.Actor{}, false
	}

	actor, err := a.signIn.Authenticate(ctx, token)
	if errors.Is(err, signin.ErrInvalidSession) {
		a.logger.WarnContext(ctx, "rejected a request with an invalid session token",
			slog.String("client_ip", ginContext.ClientIP()), slog.Any("error", err))

		apierror.Write(ginContext, errUnauthenticated)

		return identity.Actor{}, false
	}

	if err != nil {
		a.logger.ErrorContext(ctx, "failed to check a session", slog.Any("error", err))

		apierror.Write(ginContext, apierror.NewInternalProblem())

		return identity.Actor{}, false
	}

	ginContext.Request = ginContext.Request.WithContext(
		logging.With(identity.WithActor(ctx, actor), slog.String("user_id", actor.UserID.String())),
	)

	return actor, true
}
