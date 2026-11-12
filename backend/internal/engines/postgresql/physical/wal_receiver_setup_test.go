package physical

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"

	pgshared "databasus-backend/internal/engines/postgresql/shared"
	"databasus-backend/internal/lib/background"
	containers "databasus-backend/internal/lib/testcontainers"
	"databasus-backend/internal/lib/testwait"
	"databasus-backend/internal/lib/toolstub"
)

func TestWalReceiverSetup(t *testing.T) {
	t.Parallel()
	t.Run("retries with released resources when native setup stalls after identity verification", func(t *testing.T) {
		t.Parallel()
		server := startTestServer(t, "postgres:17")
		proxy := containers.StartPostgresProxy(t, server.endpoint)
		spec := newWalReceiverTestSpec(t)
		spec.ExpectedSystemIdentifier = server.readSystemIdentifier(t)
		spec.SlotName = newSlotName()
		stub := toolstub.Write(t,
			spec.ClientTools.GetExecutable(spec.Version, pgshared.ExecutablePgReceivewal),
			toolstub.Spec{Script: fmt.Sprintf("exec %q \"$@\"", getNativeWalReceiver(t, spec.Version))},
		)
		settings := server.getSettings(getAdminRole())
		settings.Host = proxy.Endpoint.Host
		settings.Port = proxy.Endpoint.Port
		source := NewSource(settings)
		require.NoError(t, source.VerifySlot(t.Context(), spec.Logger, spec.SlotName))
		inspection := server.openInspection(t, getAdminRole())
		barrier := proxy.HoldCommand("START_REPLICATION")
		var hasStarted atomic.Bool
		spec.OnStarted = func() { hasStarted.Store(true) }
		ctx, cancel := context.WithCancel(t.Context())
		runs := background.New(background.Deps{Logger: spec.Logger})
		completed := make(chan WalReceiverExit, 1)
		startReceiver := func(ctx context.Context) {
			runs.Go(ctx, "setup_receiver", func(ctx context.Context) error {
				completed <- source.ReceiveWal(ctx, spec)
				return nil
			})
		}
		t.Cleanup(func() { cancel(); require.NoError(t, runs.Wait(context.WithoutCancel(t.Context()))) })

		startReceiver(ctx)
		var heldCommand containers.PostgresProxyEvent
		testwait.Until(t, "the native receiver reaches setup on the verified session", func(context.Context) error {
			select {
			case heldCommand = <-barrier.Reached:
				return nil
			default:
				return fmt.Errorf("native setup has not reached the source")
			}
		})
		hasVerifiedIdentity := false
		for _, event := range proxy.GetEvents() {
			if event.ConnectionID == heldCommand.ConnectionID && event.Kind == "identified" {
				require.Equal(t, spec.ExpectedSystemIdentifier, event.SystemIdentifier)
				hasVerifiedIdentity = true
			}
		}

		require.True(t, hasVerifiedIdentity)

		var exit WalReceiverExit
		testwait.Until(t, "the setup deadline stops the child and joins the bridge", func(context.Context) error {
			select {
			case exit = <-completed:
				return nil
			default:
				return fmt.Errorf("verified native setup remains blocked")
			}
		})
		require.NoError(t, ctx.Err(), "setup must expire independently of caller cancellation")
		require.Equal(t, WalReceiverRetryable, exit.Disposition)
		require.False(t, hasStarted.Load())
		arguments := stub.GetLastArgs(t)
		for index, argument := range arguments {
			if argument == "-h" {
				require.NoDirExists(t, arguments[index+1])
			}
		}

		close(barrier.Release)
		testwait.Until(t, "the proxy finishes the expired source connection", func(context.Context) error {
			for _, event := range proxy.GetEvents() {
				if event.ConnectionID == heldCommand.ConnectionID && event.Kind == "closed" {
					return nil
				}
			}

			return fmt.Errorf("expired source connection remains open")
		})
		testwait.Until(t, "the expired setup releases its source slot", func(ctx context.Context) error {
			slot, err := inspection.GetSlot(ctx, spec.SlotName)
			if err != nil {
				return err
			}

			if slot == nil || slot.IsActive {
				return fmt.Errorf("source slot is missing or remains active")
			}

			return nil
		})

		startReceiver(ctx)
		testwait.Until(t, "a new receiver streams after the source responds again", func(context.Context) error {
			select {
			case retryExit := <-completed:
				return fmt.Errorf("retry exited before streaming: %+v", retryExit)
			default:
			}

			if !hasStarted.Load() {
				return fmt.Errorf("retry has not entered replication")
			}

			return nil
		})
		cancel()
		require.Equal(t, WalReceiverStopped, (<-completed).Disposition)
	})
}
