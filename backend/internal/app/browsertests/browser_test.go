package browsertests

import (
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
)

func TestBrowser(t *testing.T) {
	t.Run("signs in and out when the production frontend uses the real API", func(t *testing.T) {
		frontendRoot, err := filepath.Abs("../../../../frontend")
		require.NoError(t, err)
		//nolint:forbidigo // The browser task supplies test-run ownership outside the application configuration.
		runRoot := os.Getenv("DATABASUS_BROWSER_RUN_ROOT")
		if runRoot == "" {
			runRoot = t.TempDir()
		}
		//nolint:forbidigo // Independent browser runs must share their identity with child processes.
		runID := os.Getenv("DATABASUS_BROWSER_RUN_ID")
		if runID == "" {
			runID = uuid.NewString()
		}

		ctx, stop := signal.NotifyContext(t.Context(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()

		runBrowserChild(
			t,
			browserChild{
				context:      ctx,
				frontendRoot: frontendRoot,
				runRoot:      runRoot,
				phase:        "build",
				runID:        runID,
				address:      "",
				email:        "",
				password:     "",
			},
		)

		application := apptest.New(t)
		admin := application.CreateAdmin(t)
		gateway := newGateway(
			t,
			gatewaySpec{backendURL: application.URL, outputDirectory: filepath.Join(runRoot, "frontend", "dist")},
		)
		t.Logf("Browser owner PID: %d", os.Getpid())
		t.Logf("Browser gateway ready: %s", gateway.URL)

		runBrowserChild(
			t,
			browserChild{
				context:      ctx,
				frontendRoot: frontendRoot,
				runRoot:      runRoot,
				phase:        "run",
				runID:        runID,
				address:      gateway.URL,
				email:        admin.Email,
				password:     apptest.TestPassword,
			},
		)
	})
}
