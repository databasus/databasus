package identity_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/modules/users/internal/identity"
)

func TestRequireActor(t *testing.T) {
	t.Parallel()

	t.Run("returns the authenticated actor when the context carries one", func(t *testing.T) {
		t.Parallel()
		authenticatedActor := identity.Actor{UserID: uuid.New(), Role: identity.RoleMember}
		ctx := identity.WithActor(t.Context(), authenticatedActor)

		actor, err := identity.RequireActor(ctx)

		require.NoError(t, err)
		assert.Equal(t, authenticatedActor, actor)
	})

	t.Run("refuses lookup when the context has no actor", func(t *testing.T) {
		t.Parallel()

		actor, err := identity.RequireActor(t.Context())

		require.Error(t, err)
		assert.Equal(t, identity.Actor{}, actor)
	})
}
