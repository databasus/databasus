package main

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
	"databasus-backend/internal/lib/testwait"
)

func TestSeedBootstrapAdminConcurrency(t *testing.T) {
	t.Parallel()

	for _, scenario := range []struct {
		name           string
		isDeactivation bool
		isThirdAccount bool
		role           apiclient.UserRole
	}{
		{name: "refuses demotion when administration overlaps the transfer", role: apiclient.UserRoleMEMBER},
		{name: "refuses deactivation when administration overlaps the transfer", isDeactivation: true},
		{name: "refuses administrator grants when the previous holder overlaps the transfer", isThirdAccount: true, role: apiclient.UserRoleADMIN},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			fixture, params := setUpSeedTest(t)
			router := fixture.router
			devSession := signUpForTest(t, router, params.Email, seedPassword)
			targetID := devSession.UserId

			if scenario.isThirdAccount {
				member := router.SignUp(t, apptest.NewTestEmail("third"))
				targetID = member.UserID
			}

			installBootstrapTrigger(t, router, bootstrapTriggerSpec{
				table:     "users",
				condition: fmt.Sprintf("NEW.is_root_admin AND NEW.id = '%s'::uuid", devSession.UserId),
				action:    "PERFORM pg_advisory_xact_lock(173, 29);",
			})
			barrier := router.BeginSQLTransaction(t)
			barrier.ExecuteSQL(t, "SELECT pg_advisory_xact_lock(173, 29)")
			var barrierPID int
			barrier.QuerySQL(t, &barrierPID, "SELECT pg_backend_pid()")
			seedFinished := make(chan seedRunOutcome, 1)

			go func() { seedFinished <- runSeedWithContext(t.Context(), params) }()

			seedPID := waitForBootstrapBlocker(t, router, barrierPID)
			administrationFinished := make(chan administrationOutcome, 1)
			go func() {
				if scenario.isDeactivation {
					response, err := fixture.admin.Client.DeactivateUserWithResponse(t.Context(), targetID)
					if err != nil {
						administrationFinished <- administrationOutcome{statusCode: 0, err: err}
						return
					}

					administrationFinished <- administrationOutcome{statusCode: response.StatusCode(), err: nil}
					return
				}

				response, err := fixture.admin.Client.ChangeUserRoleWithResponse(t.Context(), targetID,
					apiclient.UserRoleChange{Role: scenario.role})
				if err != nil {
					administrationFinished <- administrationOutcome{statusCode: 0, err: err}
					return
				}

				administrationFinished <- administrationOutcome{statusCode: response.StatusCode(), err: nil}
			}()

			waitForBootstrapBlocker(t, router, seedPID)
			barrier.Commit(t)

			var seedOutcome seedRunOutcome
			testwait.Until(t, "seed completes after the bootstrap grant", func(_ context.Context) error {
				select {
				case seedOutcome = <-seedFinished:
					return nil
				default:
					return fmt.Errorf("seed is still running")
				}
			})
			require.Equal(t, 0, seedOutcome.exitCode, seedOutcome.output)

			var administration administrationOutcome
			testwait.Until(t, "administration completes after the transfer", func(_ context.Context) error {
				select {
				case administration = <-administrationFinished:
					return nil
				default:
					return fmt.Errorf("administration is still running")
				}
			})

			require.NoError(t, administration.err)
			assert.Equal(t, http.StatusForbidden, administration.statusCode)
			assertBootstrapAssignment(t, router, devSession.UserId)

			if scenario.isThirdAccount {
				assert.Equal(t, apiclient.UserRoleMEMBER, getSeedAccount(t, router, targetID).Role)
			}

			var administrationEvents int
			router.QuerySQL(t, &administrationEvents, `SELECT COUNT(*) FROM lib_audit_events
				WHERE message LIKE 'User role changed:%' OR message LIKE 'User deactivated:%'`)

			assert.Zero(t, administrationEvents)
		})
	}
}

type administrationOutcome struct {
	statusCode int
	err        error
}

func waitForBootstrapBlocker(t *testing.T, router *apptest.App, blockerPID int) int {
	t.Helper()

	var blockedPID int

	testwait.Until(t, "a bootstrap operation waits on its blocker", func(_ context.Context) error {
		router.QuerySQL(t, &blockedPID, `SELECT pid FROM pg_stat_activity
			WHERE datname = current_database() AND ? = ANY(pg_blocking_pids(pid)) AND state = 'active'
			ORDER BY pid LIMIT 1`, blockerPID)

		if blockedPID == 0 {
			return fmt.Errorf("no active operation is blocked by backend %d", blockerPID)
		}

		return nil
	})

	return blockedPID
}
