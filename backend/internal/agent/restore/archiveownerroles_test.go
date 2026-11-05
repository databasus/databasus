package restore

import (
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Verbatim `pg_restore -l` output shapes: header comments, a multi-word desc, a tag holding
// a space, an ownerless entry (trailing space after the tag), reserved and bootstrap roles.
const tocListing = ";\n" +
	"; Archive created at 2026-01-01 00:00:00 UTC\n" +
	";     dbname: postgres\n" +
	";\n" +
	"4; 3079 16388 EXTENSION - timescaledb \n" +
	"215; 1255 16400 FUNCTION public add(integer, integer) ts_app\n" +
	"251; 1259 16404 TABLE public sensor_data ts_app\n" +
	"3057; 0 16404 TABLE DATA public sensor_data ts_app\n" +
	"260; 1259 16410 TABLE public audit_log reporting\n" +
	"2615; 2200 2200 SCHEMA - public postgres\n" +
	"270; 1259 16420 TABLE public masked pg_monitor\n"

func TestParseTocOwnerRoleNames(t *testing.T) {
	t.Run("returns unique non reserved names when the archive lists owner roles", func(t *testing.T) {
		roleNames := parseTocOwnerRoleNames(tocListing)

		assert.Equal(t, []string{"ts_app", "reporting"}, roleNames)
	})
	t.Run("returns nothing when the archive listing is empty", func(t *testing.T) {
		assert.Empty(t, parseTocOwnerRoleNames(""))
	})
	t.Run("skips the tag when an archive entry has no owner", func(t *testing.T) {
		roleNames := parseTocOwnerRoleNames("4; 3079 16388 EXTENSION - timescaledb \n")

		assert.Empty(t, roleNames, "an ownerless entry must not turn its tag into a role name")
	})
}

func TestEnsureArchiveOwnerRoles(t *testing.T) {
	t.Run("skips the database when the archive has no owners", func(t *testing.T) {
		exec := &fakeExecRunner{result: ExecResult{ExitCode: 0, Stdout: ";\n"}}
		r := NewRestorer(slog.New(slog.DiscardHandler))

		ensuredRoleNames, err := r.EnsureArchiveOwnerRoles(t.Context(), exec, "/tmp/x.dump", testConn())

		require.NoError(t, err)
		assert.Empty(t, ensuredRoleNames)
		require.Len(t, exec.recorded, 1)
		assert.Equal(t, []string{"pg_restore", "-l", "/tmp/x.dump"}, exec.recorded[0].cmd)
	})
	t.Run("returns error when listing fails", func(t *testing.T) {
		exec := &fakeExecRunner{err: errors.New("exec attach failed")}
		r := NewRestorer(slog.New(slog.DiscardHandler))

		_, err := r.EnsureArchiveOwnerRoles(t.Context(), exec, "/tmp/x.dump", testConn())

		require.ErrorContains(t, err, "pg_restore -l exec")
	})
	t.Run("yields to pg restore verdict when the archive is unreadable", func(t *testing.T) {
		exec := &fakeExecRunner{result: ExecResult{ExitCode: 1, Stderr: "not a valid archive"}}
		r := NewRestorer(slog.New(slog.DiscardHandler))

		ensuredRoleNames, err := r.EnsureArchiveOwnerRoles(t.Context(), exec, "/tmp/x.dump", testConn())

		require.NoError(t, err,
			"a corrupt archive must reach pg_restore, whose exit code lets the backend reject the backup")
		assert.Empty(t, ensuredRoleNames)
	})
}
