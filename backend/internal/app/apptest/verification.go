package apptest

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

type VerificationAgent struct {
	ID         uuid.UUID  `json:"id"`
	MaxRAMGb   int        `json:"maxRamGb"`
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
	status, body := a.CallOldAPI(
		tb,
		admin,
		OldAPICall{Method: http.MethodPost, Path: "/verification/agents", Body: map[string]string{"name": tb.Name()}},
	)
	var createdAgent CreatedVerificationAgent
	decodeVerificationResponse(tb, status, http.StatusCreated, body, &createdAgent)
	return createdAgent
}

func (a *App) ListVerificationAgents(tb testing.TB, admin Session) []VerificationAgent {
	tb.Helper()
	status, body := a.CallOldAPI(tb, admin, OldAPICall{Method: http.MethodGet, Path: "/verification/agents", Body: nil})
	var agents []VerificationAgent
	decodeVerificationResponse(tb, status, http.StatusOK, body, &agents)
	return agents
}

func (a *App) EnqueueVerification(tb testing.TB, member Session, backupID uuid.UUID) uuid.UUID {
	tb.Helper()
	status, body := a.CallOldAPI(
		tb,
		member,
		OldAPICall{
			Method: http.MethodPost,
			Path:   "/verifications/enqueue",
			Body:   map[string]uuid.UUID{"backupId": backupID},
		},
	)
	var verification Verification
	decodeVerificationResponse(tb, status, http.StatusOK, body, &verification)
	return verification.ID
}

func (a *App) GetVerification(tb testing.TB, member Session, verificationID uuid.UUID) Verification {
	tb.Helper()
	status, body := a.CallOldAPI(
		tb,
		member,
		OldAPICall{Method: http.MethodGet, Path: "/verifications/" + verificationID.String(), Body: nil},
	)
	var verification Verification
	decodeVerificationResponse(tb, status, http.StatusOK, body, &verification)
	return verification
}

func decodeVerificationResponse(tb testing.TB, status, requiredStatus int, body []byte, target any) {
	tb.Helper()
	if status != requiredStatus {
		tb.Fatalf("verification request answered %d: %s", status, body)
	}
	if err := json.Unmarshal(body, target); err != nil {
		tb.Fatalf("decode verification response: %v", err)
	}
}
