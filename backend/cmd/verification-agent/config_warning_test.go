package main

import (
	"bytes"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/agent/config"
)

func TestWarnConfigPermissions(t *testing.T) {
	t.Parallel()
	for _, scenario := range []struct {
		name        string
		ownerUID    int
		shouldChown bool
	}{
		{"includes chown when the file belongs to another user", 0, true},
		{"omits chown when the file belongs to the agent user", 1000, false},
		{"omits chown when the owner cannot be determined", -1, false},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()
			var stderr, logOutput bytes.Buffer
			warnConfigPermissions(t.Context(), config.PermissionWarning{
				Path: "/agent/databasus-verification.json", Mode: 0o644,
				Owner: "file-owner", OwnerUID: scenario.ownerUID,
				AgentUser: "agent-user", AgentUID: 1000, Err: os.ErrPermission,
			}, configWarningOutput{
				log: slog.New(slog.NewTextHandler(&logOutput, nil)), stderr: &stderr,
			})

			for _, output := range []string{stderr.String(), logOutput.String()} {
				for _, detail := range []string{
					"/agent/databasus-verification.json", "0644", "file-owner", os.ErrPermission.Error(),
					"other users can read the agent token", "chmod 600 /agent/databasus-verification.json",
				} {
					assert.Contains(t, output, detail)
				}
				if scenario.shouldChown {
					assert.Contains(t, output, "sudo chown agent-user /agent/databasus-verification.json")
				} else {
					assert.NotContains(t, output, "chown")
				}
			}

			require.Contains(t, logOutput.String(), "warning: agent config file is readable by other users")
			assert.Contains(t, stderr.String(), "WARNING:")
		})
	}
}
