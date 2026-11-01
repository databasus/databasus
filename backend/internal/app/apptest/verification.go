package apptest

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/agentclient"
	"databasus-backend/internal/lib/agentclienttest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

type VerificationAgent struct {
	ID         uuid.UUID  `json:"id"`
	MaxRAMMb   int        `json:"maxRamMb"`
	LastSeenAt *time.Time `json:"lastSeenAt"`
}

type CreatedVerificationAgent struct {
	Agent VerificationAgent `json:"agent"`
	Token string            `json:"token"`
}

type Verification struct {
	ID                      uuid.UUID               `json:"id"`
	Status                  string                  `json:"status"`
	AttemptCount            int                     `json:"attemptCount"`
	PgRestoreExitCode       *int                    `json:"pgRestoreExitCode"`
	DBSizeBytesAfterRestore *int64                  `json:"dbSizeBytesAfterRestore"`
	TableCount              *int                    `json:"tableCount"`
	SchemaCount             *int                    `json:"schemaCount"`
	FailMessage             *string                 `json:"failMessage"`
	TableStats              []VerificationTableStat `json:"tableStats"`
}

type VerificationTableStat struct {
	SchemaName string `json:"schemaName"`
	Name       string `json:"name"`
	RowCount   int64  `json:"rowCount"`
}

func (a *App) CreateVerificationAgent(tb testing.TB, admin Session) CreatedVerificationAgent {
	tb.Helper()
	response, err := admin.Client.CreateVerificationAgentWithResponse(
		tb.Context(),
		apiclient.VerificationAgentInput{Name: tb.Name()},
	)
	if err != nil {
		tb.Fatal(err)
	}
	if response.JSON201 == nil {
		tb.Fatalf("create verification agent: %d %s", response.StatusCode(), response.Body)
	}
	agent := response.JSON201.Agent
	return CreatedVerificationAgent{
		Agent: VerificationAgent{ID: agent.Id, MaxRAMMb: agent.MaxRamMb, LastSeenAt: agent.LastSeenAt},
		Token: response.JSON201.Token,
	}
}

func (a *App) ListVerificationAgents(tb testing.TB, admin Session) []VerificationAgent {
	tb.Helper()
	agents := []VerificationAgent{}
	var after *string
	for {
		response, err := admin.Client.ListVerificationAgentsWithResponse(
			tb.Context(),
			&apiclient.ListVerificationAgentsParams{Limit: nil, After: after},
		)
		if err != nil {
			tb.Fatal(err)
		}
		if response.JSON200 == nil {
			tb.Fatalf("list verification agents: %d %s", response.StatusCode(), response.Body)
		}
		for _, agent := range response.JSON200.Items {
			agents = append(
				agents,
				VerificationAgent{ID: agent.Id, MaxRAMMb: agent.MaxRamMb, LastSeenAt: agent.LastSeenAt},
			)
		}
		after = response.JSON200.NextCursor
		if after == nil {
			break
		}
	}
	return agents
}

func (a *App) EnqueueVerification(tb testing.TB, member Session, backupID uuid.UUID) uuid.UUID {
	tb.Helper()
	response, err := member.Client.StartVerificationWithResponse(
		tb.Context(),
		apiclient.StartVerificationInput{BackupId: backupID},
	)
	if err != nil {
		tb.Fatal(err)
	}
	if response.JSON202 == nil {
		tb.Fatalf("start verification: %d %s", response.StatusCode(), response.Body)
	}
	return response.JSON202.Id
}

func (a *App) GetVerification(tb testing.TB, member Session, id uuid.UUID) Verification {
	tb.Helper()
	response, err := member.Client.GetVerificationWithResponse(tb.Context(), id)
	if err != nil {
		tb.Fatal(err)
	}
	if response.JSON200 == nil {
		tb.Fatalf("get verification: %d %s", response.StatusCode(), response.Body)
	}

	verification := response.JSON200
	stats := make([]VerificationTableStat, len(verification.TableStats))
	for index, stat := range verification.TableStats {
		stats[index] = VerificationTableStat{SchemaName: stat.SchemaName, Name: stat.Name, RowCount: stat.RowCount}
	}
	return Verification{
		ID:                      verification.Id,
		Status:                  string(verification.Status),
		AttemptCount:            verification.AttemptCount,
		PgRestoreExitCode:       verification.PgRestoreExitCode,
		DBSizeBytesAfterRestore: verification.RestoredDatabaseSizeBytes,
		TableCount:              verification.TableCount,
		SchemaCount:             verification.SchemaCount,
		FailMessage:             verification.FailMessage,
		TableStats:              stats,
	}
}

func (a *App) WaitForBackupRuns(tb testing.TB) {
	tb.Helper()
	waitCtx, cancelWait := context.WithTimeout(tb.Context(), backupRunsWaitTimeout)
	defer cancelWait()
	if err := a.app.WaitForBackupRuns(waitCtx); err != nil {
		tb.Fatalf("wait for backup runs: %v", err)
	}
}

func (a *App) NewAgentClient(tb testing.TB, credentials CreatedVerificationAgent) *agentclient.ClientWithResponses {
	tb.Helper()
	return agentclienttest.New(tb, a.URL, credentials.Token)
}
