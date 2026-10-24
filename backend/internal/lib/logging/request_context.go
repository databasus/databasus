package logging

import "context"

const (
	requestIDKey = "request_id"
	userIDKey    = "user_id"
)

type requestIDContextKey struct{}

type userIDContextKey struct{}

func ContextWithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDContextKey{}, requestID)
}

func GetRequestID(ctx context.Context) string {
	requestID, _ := ctx.Value(requestIDContextKey{}).(string)

	return requestID
}

// The principal rides the context for the same reason the request ID does: services hold a
// process-wide logger, so there is no scoped logger to attach it to.
func ContextWithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDContextKey{}, userID)
}

func GetUserID(ctx context.Context) string {
	userID, _ := ctx.Value(userIDContextKey{}).(string)

	return userID
}
