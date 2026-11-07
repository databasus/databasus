package identity

import (
	"context"

	"github.com/google/uuid"
)

// Actor is the user a request runs for: the user ID and the global role.
type Actor struct {
	UserID uuid.UUID
	Role   Role
}

func (a Actor) IsAdmin() bool {
	return a.Role == RoleAdmin
}

type actorContextKey struct{}

func WithActor(ctx context.Context, actor Actor) context.Context {
	return context.WithValue(ctx, actorContextKey{}, actor)
}

func GetActor(ctx context.Context) (Actor, bool) {
	actor, isFound := ctx.Value(actorContextKey{}).(Actor)

	return actor, isFound
}

func RequireActor(ctx context.Context) (Actor, error) {
	actor, hasActor := GetActor(ctx)
	if !hasActor {
		return Actor{}, errNoActor
	}

	return actor, nil
}
