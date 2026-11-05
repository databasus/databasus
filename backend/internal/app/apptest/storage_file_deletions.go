package apptest

import (
	"testing"

	"github.com/stretchr/testify/require"

	"databasus-backend/internal/modules/storages"
)

func (a *App) RequestStorageFileDeletions(t *testing.T, request storages.FileDeletionRequest) {
	t.Helper()
	require.NoError(t, a.app.RequestStorageFileDeletions(t.Context(), request))
}

func (a *App) RefuseStorageFileDeletions(t *testing.T) {
	t.Helper()

	a.ExecuteSQL(
		t,
		`CREATE FUNCTION refuse_file_deletion() RETURNS trigger AS $$ BEGIN RAISE EXCEPTION 'cleanup refused'; END $$ LANGUAGE plpgsql`,
	)
	a.ExecuteSQL(
		t,
		`CREATE TRIGGER refuse_file_deletion BEFORE INSERT ON storages_pending_file_deletions FOR EACH ROW EXECUTE FUNCTION refuse_file_deletion()`,
	)
}
