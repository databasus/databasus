package physical

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	containers "databasus-backend/internal/lib/testcontainers"
)

func TestSourceIdentity(t *testing.T) {
	t.Parallel()
	t.Run("matches an unsigned replication identifier with the signed SQL representation", func(t *testing.T) {
		t.Parallel()
		require.NoError(
			t,
			verifySourceIdentity(sourceIdentityComparison{Expected: "18446744073709551615", Actual: "-1"}),
		)
		require.NoError(
			t,
			verifySourceIdentity(sourceIdentityComparison{Expected: "-1", Actual: "18446744073709551615"}),
		)
	})
	t.Run("refuses invalid or zero identifiers without adopting a source", func(t *testing.T) {
		t.Parallel()
		for _, identity := range []sourceIdentityComparison{{Expected: "0", Actual: "0"}, {Expected: "invalid", Actual: "10"}, {Expected: "10", Actual: "invalid"}} {
			var refusal *SourceIdentityError
			require.ErrorAs(t, verifySourceIdentity(identity), &refusal)
		}
	})
}

func TestVerifiedSourceInspection(t *testing.T) {
	t.Parallel()
	t.Run("keeps slot mutations on the verified session after routing changes", func(t *testing.T) {
		t.Parallel()
		original := startTestServer(t, "postgres:17")
		foreign := startTestServer(t, "postgres:17")
		proxy := containers.StartPostgresProxy(t, original.endpoint)
		connection := original.connectAs(t, getAdminRole())
		var identity string
		require.NoError(
			t,
			connection.QueryRow(t.Context(), "SELECT system_identifier::text FROM pg_control_system()").Scan(&identity),
		)
		settings := original.getSettings(getAdminRole())
		settings.Host = proxy.Endpoint.Host
		settings.Port = proxy.Endpoint.Port
		source := NewSource(settings)
		proxy.RouteNext(foreign.endpoint, original.endpoint)
		inspection, err := source.OpenVerifiedInspection(t.Context(), identity)
		require.Nil(t, inspection)
		var refusal *SourceIdentityError
		require.ErrorAs(t, err, &refusal)
		inspection, err = source.OpenVerifiedInspection(t.Context(), identity)
		require.NoError(t, err)
		t.Cleanup(func() { _ = inspection.Close(context.WithoutCancel(t.Context())) })
		proxy.Route(
			containers.PostgresProxyRoute{SQLDestination: foreign.endpoint, ReplicationDestination: foreign.endpoint},
		)
		slotName := "inspection_" + strings.ReplaceAll(uuid.NewString(), "-", "")
		require.NoError(t, foreign.newSource(getAdminRole()).VerifySlot(t.Context(), newTestLogger(t), slotName))
		require.NoError(t, inspection.EnsureSlot(t.Context(), newTestLogger(t), slotName))
		var originalSlots int
		require.NoError(
			t,
			connection.QueryRow(t.Context(), "SELECT count(*) FROM pg_replication_slots WHERE slot_name=$1", slotName).
				Scan(&originalSlots),
		)
		require.Equal(t, 1, originalSlots)
		require.NoError(t, inspection.DropSlot(t.Context(), newTestLogger(t), slotName))
		foreignConnection := foreign.connectAs(t, getAdminRole())
		var foreignSlots int
		require.NoError(
			t,
			foreignConnection.QueryRow(t.Context(), "SELECT count(*) FROM pg_replication_slots WHERE slot_name=$1", slotName).
				Scan(&foreignSlots),
		)
		require.Equal(t, 1, foreignSlots)
	})
}
