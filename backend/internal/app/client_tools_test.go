package app

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/engines/clienttools"
	"databasus-backend/internal/lib/toolstub"
)

func TestCheckClientTools(t *testing.T) {
	t.Run("passes the check when only a mysql client is broken", func(t *testing.T) {
		environment := newClientToolsEnvironment(t, newBrokenBundle(t, "mysql", false))

		assert.NoError(t, environment.CheckClientTools(t.Context()))
	})

	t.Run("refuses when a postgresql client is broken", func(t *testing.T) {
		environment := newClientToolsEnvironment(t, newBrokenBundle(t, "postgresql", true))

		assert.ErrorIs(t, environment.CheckClientTools(t.Context()), ErrFatalClientTools)
	})
}

func newClientToolsEnvironment(t *testing.T, bundle clienttools.Bundle) *Environment {
	t.Helper()

	t.Setenv("ENV_MODE", "development")

	environment, err := LoadEnvironment(t.Context(), newIsolatedLoadSpec(t, false))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, environment.Logging.Shutdown(context.WithoutCancel(t.Context()))) })

	environment.clientToolsDirErr = nil
	environment.clientToolsChecker = clienttools.NewChecker([]clienttools.Bundle{bundle})

	return environment
}

func newBrokenBundle(t *testing.T, engine string, isFatal bool) clienttools.Bundle {
	t.Helper()

	binDir := t.TempDir()
	toolstub.Write(t, filepath.Join(binDir, "tool"), toolstub.Spec{IsReadingStdin: false, Script: "exit 1"})

	return clienttools.Bundle{
		Engine:           engine,
		Version:          "1",
		BinDir:           binDir,
		RequiredCommands: []string{"tool"},
		ReleaseLine:      "1",
		ParseVersion:     nil,
		IsFatal:          isFatal,
	}
}
