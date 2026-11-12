package physical

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/engines/clienttools"
	pgshared "databasus-backend/internal/engines/postgresql/shared"
	"databasus-backend/internal/lib/background"
	containers "databasus-backend/internal/lib/testcontainers"
	"databasus-backend/internal/lib/testwait"
	"databasus-backend/internal/lib/toolstub"
)

//nolint:tparallel // Each engine-version scope releases its source pair before the next version starts.
func TestVerifiedWalReceiver(t *testing.T) {
	t.Parallel()
	for _, image := range getSupportedVersionImages() {
		t.Run("streams only from the verified cluster with one WAL sender on "+image, func(t *testing.T) {
			original := startTestServer(t, image, containers.WithMaxWalSenders(1))
			foreign := startTestServer(t, image)
			proxy := containers.StartPostgresProxy(t, original.endpoint)
			proxy.SetResponseFragmentSize(3)
			connection := original.connectAs(t, getAdminRole())
			var identity string
			require.NoError(
				t,
				connection.QueryRow(t.Context(), "SELECT system_identifier::text FROM pg_control_system()").
					Scan(&identity),
			)
			spec := newWalReceiverTestSpec(t)
			spec.Logger = newTestLogger(t)
			spec.Version = original.version
			stub := toolstub.Write(
				t,
				spec.ClientTools.GetExecutable(spec.Version, pgshared.ExecutablePgReceivewal),
				toolstub.Spec{
					Script: fmt.Sprintf(
						"exec %q \"$@\"",
						getNativeWalReceiver(t, spec.Version),
					),
				},
			)
			spec.ExpectedSystemIdentifier = identity
			spec.SlotName = "receiver_" + strings.ReplaceAll(uuid.NewString(), "-", "")
			settings := original.getSettings(getAdminRole())
			settings.Host = proxy.Endpoint.Host
			settings.Port = proxy.Endpoint.Port
			source := NewSource(settings)
			require.NoError(t, source.VerifySlot(t.Context(), spec.Logger, spec.SlotName))
			started := make(chan struct{}, 1)
			spec.OnStarted = func() {
				select {
				case started <- struct{}{}:
				default:
				}
			}
			ctx, cancel := context.WithCancel(t.Context())
			runs := background.New(background.Deps{Logger: spec.Logger})
			completed := make(chan WalReceiverExit, 1)
			runs.Go(
				ctx,
				"receiver_under_test",
				func(ctx context.Context) error { completed <- source.ReceiveWal(ctx, spec); return nil },
			)
			t.Cleanup(func() { cancel(); require.NoError(t, runs.Wait(context.WithoutCancel(t.Context()))) })
			testwait.Until(t, "the verified native receiver enters replication", func(context.Context) error {
				select {
				case <-started:
					return nil
				case exit := <-completed:
					t.Fatalf("receiver exited: %+v; cause: %v", exit, exit.FatalErr)
					return nil
				default:
					return fmt.Errorf("not streaming")
				}
			})
			arguments := stub.GetLastArgs(t)
			socketDirectory := ""
			for index, argument := range arguments {
				if argument == "-h" {
					socketDirectory = arguments[index+1]
				}
			}
			require.NotEmpty(t, socketDirectory)
			permissions, err := os.Stat(socketDirectory)
			require.NoError(t, err)
			require.Equal(t, os.FileMode(0o700), permissions.Mode().Perm())
			extraConnection, err := pgconn.Connect(
				t.Context(),
				fmt.Sprintf("host=%s port=5432 user=testuser replication=true sslmode=disable", socketDirectory),
			)
			if extraConnection != nil {
				_ = extraConnection.Close(t.Context())
			}
			require.Error(t, err, "a second local client must not attach to the verified source")
			_, err = connection.Exec(t.Context(), "SELECT pg_switch_wal()")
			require.NoError(t, err)
			testwait.Until(t, "native receivewal writes a completed WAL segment", func(context.Context) error {
				entries, err := os.ReadDir(spec.WatchDir)
				if err != nil {
					return err
				}
				for _, entry := range entries {
					if len(entry.Name()) == 24 {
						return nil
					}
				}
				return fmt.Errorf("no completed segment")
			})
			cancel()
			exit := <-completed
			require.Equal(t, WalReceiverStopped, exit.Disposition)
			require.NoDirExists(t, socketDirectory)
			testwait.Until(t, "PostgreSQL releases the closed receiver session", func(ctx context.Context) error {
				var isActive bool
				if err := connection.QueryRow(ctx, "SELECT active FROM pg_replication_slots WHERE slot_name=$1", spec.SlotName).
					Scan(&isActive); err != nil {
					return err
				}
				if isActive {
					return fmt.Errorf("the server has not released the slot")
				}
				return nil
			})
			proxy.Route(
				containers.PostgresProxyRoute{
					SQLDestination:         original.endpoint,
					ReplicationDestination: foreign.endpoint,
				},
			)
			exit = source.ReceiveWal(t.Context(), spec)
			require.Equal(t, WalReceiverIdentityMismatch, exit.Disposition)
			var identityError *SourceIdentityError
			require.ErrorAs(t, exit.FatalErr, &identityError)
			require.Equal(t, identity, identityError.ExpectedSystemIdentifier)
			require.NotEqual(t, identity, identityError.ActualSystemIdentifier)
			spec.ExpectedSystemIdentifier = ""
			exit = source.ReceiveWal(t.Context(), spec)
			require.ErrorIs(t, exit.FatalErr, ErrSourceIdentityMissing)
		})
	}
}

func TestVerifiedWalCancellation(t *testing.T) {
	t.Parallel()
	for _, command := range []string{"IDENTIFY_SYSTEM", "START_REPLICATION"} {
		t.Run("releases the source and private socket when cancellation interrupts "+command, func(t *testing.T) {
			t.Parallel()
			server := startTestServer(t, "postgres:17")
			proxy := containers.StartPostgresProxy(t, server.endpoint)
			connection := server.connectAs(t, getAdminRole())
			spec := newWalReceiverTestSpec(t)
			require.NoError(
				t,
				connection.QueryRow(t.Context(), "SELECT system_identifier::text FROM pg_control_system()").
					Scan(&spec.ExpectedSystemIdentifier),
			)
			spec.SlotName = "cancel_" + strings.ReplaceAll(uuid.NewString(), "-", "")
			settings := server.getSettings(getAdminRole())
			settings.Host = proxy.Endpoint.Host
			settings.Port = proxy.Endpoint.Port
			source := NewSource(settings)
			require.NoError(t, source.VerifySlot(t.Context(), spec.Logger, spec.SlotName))
			binaryPath := getNativeWalReceiver(t, spec.Version)
			stub := toolstub.Write(
				t,
				spec.ClientTools.GetExecutable(spec.Version, pgshared.ExecutablePgReceivewal),
				toolstub.Spec{Script: fmt.Sprintf("exec %q \"$@\"", binaryPath)},
			)
			barrier := proxy.HoldCommand(command)
			ctx, cancel := context.WithCancel(t.Context())
			runs := background.New(background.Deps{Logger: spec.Logger})
			completed := make(chan WalReceiverExit, 1)
			runs.Go(
				ctx,
				"cancelled_receiver",
				func(ctx context.Context) error { completed <- source.ReceiveWal(ctx, spec); return nil },
			)
			t.Cleanup(func() { cancel(); require.NoError(t, runs.Wait(context.WithoutCancel(t.Context()))) })
			testwait.Until(t, "the source reaches the held replication command", func(context.Context) error {
				select {
				case <-barrier.Reached:
					return nil
				default:
					return fmt.Errorf("command not reached")
				}
			})
			cancel()
			var exit WalReceiverExit
			testwait.Until(t, "the cancelled attempt joins its child and bridge", func(context.Context) error {
				select {
				case exit = <-completed:
					return nil
				default:
					return fmt.Errorf("attempt still running")
				}
			})
			require.Equal(t, WalReceiverStopped, exit.Disposition)
			close(barrier.Release)
			if command == "START_REPLICATION" {
				arguments := stub.GetLastArgs(t)
				for index, argument := range arguments {
					if argument == "-h" {
						require.NoDirExists(t, arguments[index+1])
					}
				}
			}
			testwait.Until(t, "the cancelled session releases its source slot", func(ctx context.Context) error {
				var isActive bool
				if err := connection.QueryRow(ctx, "SELECT active FROM pg_replication_slots WHERE slot_name=$1", spec.SlotName).
					Scan(&isActive); err != nil {
					return err
				}
				if isActive {
					return fmt.Errorf("slot remains active")
				}
				return nil
			})
		})
	}
}

func TestVerifiedWalRejection(t *testing.T) {
	t.Parallel()
	t.Run("never reports streaming when the verified server rejects the native receiver", func(t *testing.T) {
		t.Parallel()
		server := startTestServer(t, "postgres:17")
		connection := server.connectAs(t, getAdminRole())
		spec := newWalReceiverTestSpec(t)
		require.NoError(
			t,
			connection.QueryRow(t.Context(), "SELECT system_identifier::text FROM pg_control_system()").
				Scan(&spec.ExpectedSystemIdentifier),
		)
		spec.SlotName = "absent_" + strings.ReplaceAll(uuid.NewString(), "-", "")
		binaryPath := getNativeWalReceiver(t, spec.Version)
		stub := toolstub.Write(
			t,
			spec.ClientTools.GetExecutable(spec.Version, pgshared.ExecutablePgReceivewal),
			toolstub.Spec{Script: fmt.Sprintf("exec %q \"$@\"", binaryPath)},
		)
		hasStarted := false
		spec.OnStarted = func() { hasStarted = true }
		receiver := server.newSource(getAdminRole()).ReceiveWal(t.Context(), spec)
		require.Equal(t, WalReceiverRetryable, receiver.Disposition)
		require.False(t, hasStarted)
		arguments := stub.GetLastArgs(t)
		for index, argument := range arguments {
			if argument == "-h" {
				require.NoDirExists(t, arguments[index+1])
			}
		}
	})
}

func getNativeWalReceiver(t *testing.T, version pgshared.Version) string {
	t.Helper()
	workingDirectory, err := os.Getwd()
	require.NoError(t, err)
	bundleRoot, err := clienttools.FindBundleRoot(workingDirectory)
	require.NoError(t, err)
	return (pgshared.ClientTools{Root: bundleRoot}).GetExecutable(version, pgshared.ExecutablePgReceivewal)
}
