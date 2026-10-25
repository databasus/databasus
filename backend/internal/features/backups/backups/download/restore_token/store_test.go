package restore_token

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/cache"
	"databasus-backend/internal/lib/clock"
)

type failingStore struct {
	err error
}

func (s failingStore) Get(context.Context, string) ([]byte, bool, error) {
	return nil, false, s.err
}

func (s failingStore) Set(context.Context, cache.Entry) error {
	return s.err
}

func (s failingStore) CreateIfAbsent(context.Context, cache.Entry) (bool, error) {
	return false, s.err
}

func (s failingStore) ReadAndDelete(context.Context, string) ([]byte, bool, error) {
	return nil, false, s.err
}

func (s failingStore) Delete(context.Context, string) error {
	return s.err
}

func (s failingStore) Clear(context.Context) error {
	return s.err
}

func Test_Store_IssueAndConsume_ConsumesTokenOnce(t *testing.T) {
	tokenStore := newStore(cache.NewMemoryStore(cache.Deps{PayloadBudgetBytes: 4096, Clock: clock.NewSystem()}))
	restoreToken := Token{DatabaseID: uuid.New(), UserID: uuid.New()}

	require.NoError(t, tokenStore.issue(t.Context(), "token", restoreToken))
	consumedToken, err := tokenStore.consume(t.Context(), "token")
	require.NoError(t, err)
	require.NotNil(t, consumedToken)
	assert.Equal(t, restoreToken, *consumedToken)

	consumedToken, err = tokenStore.consume(t.Context(), "token")
	require.NoError(t, err)
	assert.Nil(t, consumedToken)
}

func Test_Store_WhenProviderFails_ReturnsError(t *testing.T) {
	providerError := errors.New("provider failed")
	tokenStore := newStore(failingStore{err: providerError})

	err := tokenStore.issue(t.Context(), "token", Token{})
	assert.ErrorIs(t, err, providerError)

	_, err = tokenStore.consume(t.Context(), "token")
	assert.ErrorIs(t, err, providerError)
}
