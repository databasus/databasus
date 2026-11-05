package apptest

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
)

type PhysicalRequest func(context.Context, *apiclient.ClientWithResponses) (*http.Response, error)

func (a *App) OpenPhysicalAPI(tb testing.TB, session Session, request PhysicalRequest) *http.Response {
	tb.Helper()
	response, err := request(tb.Context(), a.NewSessionClient(tb, session.Token))
	require.NoError(tb, err)
	tb.Cleanup(func() { _ = response.Body.Close() })
	return response
}

func (a *App) CallPhysicalAPI(tb testing.TB, session Session, request PhysicalRequest) (int, []byte) {
	tb.Helper()
	response := a.OpenPhysicalAPI(tb, session, request)
	defer func() { _ = response.Body.Close() }()
	body, err := io.ReadAll(response.Body)
	require.NoError(tb, err)
	return response.StatusCode, body
}

func PhysicalListRequest(databaseID uuid.UUID, query string) PhysicalRequest {
	return func(ctx context.Context, client *apiclient.ClientWithResponses) (*http.Response, error) {
		values, err := url.ParseQuery(strings.TrimPrefix(query, "?"))
		if err != nil {
			return nil, err
		}

		params := apiclient.ListPhysicalBackupsParams{DatabaseId: databaseID}
		if values.Has("limit") {
			limit, err := strconv.Atoi(values.Get("limit"))
			if err != nil {
				return nil, err
			}
			params.Limit = &limit
		}
		if values.Has("after") {
			params.After = new(values.Get("after"))
		}
		if values.Has("createdBefore") {
			before, err := time.Parse(time.RFC3339, values.Get("createdBefore"))
			if err != nil {
				return nil, err
			}
			params.CreatedBefore = &before
		}
		if values.Has("type") {
			types := []apiclient.PhysicalBackupListRowType{}
			for _, backupType := range values["type"] {
				types = append(types, apiclient.PhysicalBackupListRowType(backupType))
			}
			params.Type = &types
		}
		if values.Has("status") {
			statuses := []apiclient.PhysicalBackupStatus{}
			for _, status := range values["status"] {
				statuses = append(statuses, apiclient.PhysicalBackupStatus(status))
			}
			params.Status = &statuses
		}

		return client.ListPhysicalBackups(ctx, &params)
	}
}

func PhysicalStartRequest(databaseID uuid.UUID, kind string) PhysicalRequest {
	return func(ctx context.Context, client *apiclient.ClientWithResponses) (*http.Response, error) {
		return client.StartPhysicalBackup(
			ctx,
			apiclient.PhysicalBackupStart{DatabaseId: databaseID, Kind: apiclient.PhysicalBackupStartKind(kind)},
		)
	}
}

func PhysicalCancelRequest(backupID uuid.UUID) PhysicalRequest {
	return func(ctx context.Context, client *apiclient.ClientWithResponses) (*http.Response, error) {
		return client.CancelPhysicalBackup(ctx, backupID)
	}
}

func PhysicalDeleteRequest(backupID uuid.UUID) PhysicalRequest {
	return func(ctx context.Context, client *apiclient.ClientWithResponses) (*http.Response, error) {
		return client.DeletePhysicalBackup(ctx, backupID)
	}
}

func PhysicalPitrTokenRequest(databaseID uuid.UUID, targetTime *time.Time) PhysicalRequest {
	return func(ctx context.Context, client *apiclient.ClientWithResponses) (*http.Response, error) {
		var body apiclient.PhysicalRestoreTokenCreate
		if err := body.FromPhysicalPointInTimeRestoreTokenCreate(
			apiclient.PhysicalPointInTimeRestoreTokenCreate{
				Type:       apiclient.PhysicalPointInTimeRestoreTokenCreateTypePOINTINTIME,
				DatabaseId: databaseID,
				TargetTime: targetTime,
			},
		); err != nil {
			return nil, err
		}
		return client.CreatePhysicalRestoreToken(ctx, body)
	}
}

func PhysicalBackupTokenRequest(backupID uuid.UUID) PhysicalRequest {
	return func(ctx context.Context, client *apiclient.ClientWithResponses) (*http.Response, error) {
		var body apiclient.PhysicalRestoreTokenCreate
		if err := body.FromPhysicalBackupRestoreTokenCreate(
			apiclient.PhysicalBackupRestoreTokenCreate{
				Type:     apiclient.PhysicalBackupRestoreTokenCreateTypeBACKUP,
				BackupId: backupID,
			},
		); err != nil {
			return nil, err
		}
		return client.CreatePhysicalRestoreToken(ctx, body)
	}
}

func PhysicalDownloadRequest(token string) PhysicalRequest {
	return func(ctx context.Context, client *apiclient.ClientWithResponses) (*http.Response, error) {
		return client.DownloadPhysicalRestoreSet(ctx, &apiclient.DownloadPhysicalRestoreSetParams{Token: token})
	}
}

func PhysicalScriptRequest() PhysicalRequest {
	return func(ctx context.Context, client *apiclient.ClientWithResponses) (*http.Response, error) {
		return client.GetRecoveryScript(ctx)
	}
}

func PhysicalStatisticsRequest(databaseID uuid.UUID, period string) PhysicalRequest {
	return func(ctx context.Context, client *apiclient.ClientWithResponses) (*http.Response, error) {
		return client.GetWalStreamStatistics(
			ctx,
			databaseID,
			&apiclient.GetWalStreamStatisticsParams{Period: apiclient.WalStreamStatisticsPeriod(period)},
		)
	}
}

func PhysicalLiveRequest(databaseID uuid.UUID) PhysicalRequest {
	return func(ctx context.Context, client *apiclient.ClientWithResponses) (*http.Response, error) {
		return client.GetWalStream(ctx, databaseID)
	}
}
