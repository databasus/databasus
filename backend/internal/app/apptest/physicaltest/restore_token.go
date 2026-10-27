package physicaltest

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	"databasus-backend/internal/lib/testwait"
)

func (team Team) restoreTokenCall(targetTime *time.Time) apptest.OldAPICall {
	return apptest.OldAPICall{Method: http.MethodPost, Path: team.getDatabasePath() + "/restore-token", Body: struct {
		TargetTime *time.Time `json:"targetTime"`
	}{TargetTime: targetTime}}
}

func (team Team) restoreToken(t *testing.T, targetTime *time.Time) string {
	t.Helper()

	var response struct {
		Token string `json:"token"`
	}
	team.call(
		t,
		oldAPIResponseSpec{Call: team.restoreTokenCall(targetTime), Status: http.StatusOK, Destination: &response},
	)
	require.NotEmpty(t, response.Token)
	return response.Token
}

func (team Team) downloadRestoreBundle(t *testing.T, targetTime *time.Time) string {
	t.Helper()

	token := team.restoreToken(t, targetTime)
	hostPath := filepath.Join(t.TempDir(), "restore.tar")
	status := team.App.DownloadOldPublicAPI(t, "/backups/physical/restore-stream?token="+token, hostPath)
	require.Equal(t, http.StatusOK, status)
	return hostPath
}

func (team Team) waitForTargetTimeReplayable(t *testing.T, targetTime time.Time) {
	t.Helper()

	testwait.Until(t, "the target time becomes restorable", func(context.Context) error {
		status, body := team.App.CallOldAPI(t, team.Owner, team.restoreTokenCall(&targetTime))
		if status != http.StatusOK {
			return fmt.Errorf("restore token answered %d: %s", status, body)
		}
		return nil
	})
}
