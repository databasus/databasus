package identity

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
)

type sessionContextKey struct{}

// WithSession records which session a request runs in, by the SHA-256 of its token, so a proof can
// be bound to that session without keeping the token.
func WithSession(ctx context.Context, token string) context.Context {
	digest := sha256.Sum256([]byte(token))

	return context.WithValue(ctx, sessionContextKey{}, hex.EncodeToString(digest[:]))
}

// GetSessionSha256 reports false for a request without a session.
func GetSessionSha256(ctx context.Context) (string, bool) {
	sessionSha256, isFound := ctx.Value(sessionContextKey{}).(string)

	return sessionSha256, isFound && sessionSha256 != ""
}
