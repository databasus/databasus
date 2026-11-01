package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSaveToJSON(t *testing.T) {
	t.Run("creates a private file on first save", func(t *testing.T) {
		t.Chdir(t.TempDir())
		cfg := &Config{Token: "private-token"}
		require.NoError(t, cfg.SaveToJSON())

		fileInfo, err := os.Stat(configFileName)
		require.NoError(t, err)
		require.Equal(t, os.FileMode(0o600), fileInfo.Mode().Perm())
	})
	t.Run("keeps the file private when rewriting", func(t *testing.T) {
		t.Chdir(t.TempDir())
		require.NoError(t, os.WriteFile(configFileName, []byte("{}"), 0o600))
		cfg := &Config{Token: "private-token"}
		require.NoError(t, cfg.SaveToJSON())

		fileInfo, err := os.Stat(configFileName)
		require.NoError(t, err)
		require.Equal(t, os.FileMode(0o600), fileInfo.Mode().Perm())
	})
}

func TestEnsurePrivateFile(t *testing.T) {
	for _, scenario := range []struct {
		name        string
		mode        os.FileMode
		shouldChmod bool
	}{
		{"narrows a file from an earlier release", 0o644, true},
		{"narrows a group-readable file", 0o640, true},
		{"leaves a file without group or other bits alone", 0o400, false},
		{"leaves a private writable file alone", 0o600, false},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			require.NoError(t, os.WriteFile(configFileName, []byte("{}"), 0o600))
			require.NoError(t, os.Chmod(configFileName, scenario.mode))

			hasChmod := false
			warning, err := EnsurePrivateFile(configFileName, func(path string, mode os.FileMode) error {
				hasChmod = true
				return os.Chmod(path, mode)
			})
			require.NoError(t, err)

			require.Nil(t, warning)
			require.Equal(t, scenario.shouldChmod, hasChmod)

			fileInfo, err := os.Stat(configFileName)
			require.NoError(t, err)

			privateMode := scenario.mode
			if scenario.shouldChmod {
				privateMode = 0o600
			}
			require.Equal(t, privateMode, fileInfo.Mode().Perm())
		})
	}
	t.Run("returns warning details when narrowing fails", func(t *testing.T) {
		t.Chdir(t.TempDir())
		require.NoError(t, os.WriteFile(configFileName, []byte("{}"), 0o644))
		require.NoError(t, os.Chmod(configFileName, 0o644))

		warning, err := EnsurePrivateFile(configFileName, func(string, os.FileMode) error { return os.ErrPermission })
		require.NoError(t, err)

		require.NotNil(t, warning)
		require.ErrorIs(t, warning.Err, os.ErrPermission)
		require.Equal(t, configFileName, warning.Path)
		require.Equal(t, os.FileMode(0o644), warning.Mode)
		require.NotEmpty(t, warning.Owner)
		require.NotEmpty(t, warning.AgentUser)
		require.Equal(t, os.Getuid(), warning.OwnerUID)
		require.Equal(t, os.Getuid(), warning.AgentUID)
	})
}
