package main

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type warningHandler struct{ records []slog.Record }

func (h *warningHandler) Enabled(context.Context, slog.Level) bool { return true }
func (h *warningHandler) Handle(_ context.Context, record slog.Record) error {
	if strings.HasPrefix(record.Message, "warning:") {
		h.records = append(h.records, record.Clone())
	}
	return nil
}
func (h *warningHandler) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h *warningHandler) WithGroup(string) slog.Handler      { return h }

func TestLoadConfig(t *testing.T) {
	t.Run("warns at every start and keeps loading when narrowing fails", func(t *testing.T) {
		directory := t.TempDir()
		t.Chdir(directory)
		path := filepath.Join(directory, "databasus-verification.json")
		require.NoError(t, os.WriteFile(path, []byte(`{"token":"private-agent-token"}`), 0o644))
		require.NoError(t, os.Chmod(path, 0o644))
		account, err := user.Current()
		require.NoError(t, err)

		for range 2 {
			var stderr bytes.Buffer
			handler := &warningHandler{}
			cfg, err := loadConfig(t.Context(), configLoadOptions{
				log: slog.New(handler), stderr: &stderr,
				chmod: func(string, os.FileMode) error { return os.ErrPermission },
			})
			require.NoError(t, err)
			require.Equal(t, "private-agent-token", cfg.Token)
			require.Len(t, handler.records, 1)

			record := handler.records[0]
			var logOutput bytes.Buffer
			logOutput.WriteString(record.Message)
			record.Attrs(
				func(attribute slog.Attr) bool { logOutput.WriteString(" " + attribute.Value.String()); return true },
			)

			assert.True(t, strings.HasPrefix(stderr.String(), "WARNING:"))
			assert.True(t, strings.HasPrefix(record.Message, "warning:"))

			for _, output := range []string{stderr.String(), logOutput.String()} {
				for _, detail := range []string{
					path, "0644", account.Username, os.ErrPermission.Error(),
					"other users can read the agent token", "chmod 600 " + path,
				} {
					assert.Contains(t, output, detail)
				}
				assert.NotContains(t, output, cfg.Token)
				assert.NotContains(t, output, "sudo chown")
			}
		}
	})
	t.Run("loads silently when an older file can be narrowed", func(t *testing.T) {
		t.Chdir(t.TempDir())
		require.NoError(
			t,
			os.WriteFile("databasus-verification.json", []byte(`{"token":"private-agent-token"}`), 0o644),
		)
		require.NoError(t, os.Chmod("databasus-verification.json", 0o644))

		var stderr bytes.Buffer
		handler := &warningHandler{}
		cfg, err := loadConfig(t.Context(), configLoadOptions{log: slog.New(handler), stderr: &stderr, chmod: os.Chmod})
		require.NoError(t, err)

		assert.Equal(t, "private-agent-token", cfg.Token)
		assert.Empty(t, stderr.String())
		assert.Empty(t, handler.records)

		fileInfo, err := os.Stat("databasus-verification.json")
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o600), fileInfo.Mode().Perm())
	})
}
