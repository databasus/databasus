package users

import (
	"context"

	"github.com/google/uuid"
)

type Role string

const (
	RoleAdmin  Role = "ADMIN"
	RoleMember Role = "MEMBER"
)

func (r Role) IsValid() bool {
	switch r {
	case RoleAdmin, RoleMember:
		return true
	default:
		return false
	}
}

type Actor struct {
	UserID uuid.UUID
	Role   Role
}

type actorContextKey struct{}

func WithActor(ctx context.Context, actor Actor) context.Context {
	return context.WithValue(ctx, actorContextKey{}, actor)
}

func GetActor(ctx context.Context) (Actor, bool) {
	actor, isFound := ctx.Value(actorContextKey{}).(Actor)

	return actor, isFound
}
