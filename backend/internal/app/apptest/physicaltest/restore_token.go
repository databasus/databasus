package physicaltest

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	"databasus-backend/internal/lib/testwait"
)

func (team Team) restoreTokenCall(targetTime *time.Time) apptest.PhysicalRequest {
	return apptest.PhysicalPitrTokenRequest(team.Database.Id, targetTime)
}

func (team Team) restoreToken(t *testing.T, targetTime *time.Time) string {
	t.Helper()

	var response struct {
		Token string `json:"token"`
	}
	team.call(
		t,
		physicalResponseSpec{
			Call:        team.restoreTokenCall(targetTime),
			Status:      http.StatusCreated,
			Destination: &response,
		},
	)
	require.NotEmpty(t, response.Token)
	return response.Token
}

func (team Team) downloadRestoreBundle(t *testing.T, targetTime *time.Time) string {
	t.Helper()

	token := team.restoreToken(t, targetTime)
	hostPath := filepath.Join(t.TempDir(), "restore.tar")
	response := team.App.OpenPhysicalAPI(t, apptest.Session{}, apptest.PhysicalDownloadRequest(token))
	defer func() { _ = response.Body.Close() }()
	require.Equal(t, http.StatusOK, response.StatusCode)

	destination, err := os.Create(hostPath)
	require.NoError(t, err)
	defer func() { _ = destination.Close() }()
	_, err = io.Copy(destination, response.Body)
	require.NoError(t, err)
	require.NoError(t, destination.Close())
	return hostPath
}

func (team Team) waitForTargetTimeReplayable(t *testing.T, targetTime time.Time) {
	t.Helper()

	testwait.Until(t, "the target time becomes restorable", func(context.Context) error {
		status, body := team.App.CallPhysicalAPI(t, team.Owner, team.restoreTokenCall(&targetTime))
		if status != http.StatusCreated {
			return fmt.Errorf("restore token answered %d: %s", status, body)
		}
		return nil
	})
}
