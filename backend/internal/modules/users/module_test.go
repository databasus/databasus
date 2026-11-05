package users_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/modules/users"
)

func TestGetActor(t *testing.T) {
	t.Parallel()

	t.Run("returns the actor when the context carries one", func(t *testing.T) {
		t.Parallel()
		storedActor := users.Actor{UserID: uuid.New(), Role: users.RoleMember}

		readActor, isFound := users.GetActor(users.WithActor(t.Context(), storedActor))

		require.True(t, isFound)
		require.Equal(t, storedActor, readActor)
	})

	t.Run("reports no actor when the context carries none", func(t *testing.T) {
		t.Parallel()

		_, isFound := users.GetActor(t.Context())

		require.False(t, isFound)
	})
}
