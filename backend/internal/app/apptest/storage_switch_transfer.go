package apptest

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
)

type StorageSwitchTransfer struct {
	DatabaseID  uuid.UUID
	WorkspaceID uuid.UUID
	StorageID   uuid.UUID
}

func TransferToStorage(t *testing.T, owner Session, target StorageSwitchTransfer) apiclient.BackupConfig {
	t.Helper()

	var storage apiclient.DatabaseTransferStorage
	require.NoError(t, storage.FromStorageSwitchedOnTransfer(apiclient.StorageSwitchedOnTransfer{
		Type:      apiclient.StorageSwitchedOnTransferTypeSWITCHTOSTORAGE,
		StorageId: target.StorageID,
	}))

	response, err := owner.Client.TransferDatabaseWithResponse(
		t.Context(),
		target.DatabaseID,
		apiclient.DatabaseTransfer{
			TargetWorkspaceId: target.WorkspaceID,
			Storage:           storage,
		},
	)
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, response.StatusCode(), "%s", response.Body)
	return GetBackupConfig(t, owner, target.DatabaseID)
}
