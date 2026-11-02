package clienttools

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/toolstub"
)

func TestCheck(t *testing.T) {
	t.Parallel()

	t.Run("passes when the client reports a version on its release line", func(t *testing.T) {
		t.Parallel()
		binDir := t.TempDir()
		writeFakeClient(t, binDir, "mariadb-dump",
			"mariadb-dump  Ver 10.19 Distrib 10.6.28-MariaDB, for debian-linux-gnu (x86_64)")
		checker := NewChecker([]Bundle{fakeBundle(binDir, "10.6", "mariadb-dump")})

		results := checker.Check(t.Context())

		require.Len(t, results, 1)
		assert.Empty(t, results[0].Errors)
	})

	// The state assets/tools/arm/mariadb/mariadb-10.6/ was in: a 10.11 binary under a directory
	// named for the 10.6 line.
	t.Run("fails naming both versions when the bundle holds a client of another line", func(t *testing.T) {
		t.Parallel()
		binDir := t.TempDir()
		writeFakeClient(t, binDir, "mariadb-dump",
			"mariadb-dump  Ver 10.19 Distrib 10.11.14-MariaDB, for debian-linux-gnu (aarch64)")
		checker := NewChecker([]Bundle{fakeBundle(binDir, "10.6", "mariadb-dump")})

		results := checker.Check(t.Context())

		require.Len(t, results[0].Errors, 1)
		require.ErrorIs(t, results[0].Errors[0], ErrForeignClientVersion)
		assert.Contains(t, results[0].Errors[0].Error(), "10.11.14")
		assert.Contains(t, results[0].Errors[0].Error(), "10.6")
	})

	// An ELF header this host cannot load stands in for a missing shared library: the file is there
	// and executable, and still cannot run.
	t.Run("fails when the client cannot start", func(t *testing.T) {
		t.Parallel()
		binDir := t.TempDir()
		require.NoError(
			t,
			toolstub.WriteExecutable(filepath.Join(binDir, "mysqldump"), "\x7fELF not a loadable image"),
		)
		checker := NewChecker([]Bundle{fakeBundle(binDir, "8.0", "mysqldump")})

		results := checker.Check(t.Context())

		require.Len(t, results[0].Errors, 1)
		assert.Contains(t, results[0].Errors[0].Error(), "cannot run")
	})

	t.Run("only requires the client to start without a release line", func(t *testing.T) {
		t.Parallel()
		binDir := t.TempDir()
		writeFakeClient(t, binDir, "mongodump", "mongodump version: 100.19.1")
		bundle := fakeBundle(binDir, "", "mongodump")
		bundle.ParseVersion = nil
		checker := NewChecker([]Bundle{bundle})

		results := checker.Check(t.Context())

		assert.Empty(t, results[0].Errors)
	})
}

func TestGetHealthError(t *testing.T) {
	t.Parallel()

	t.Run("reuses the startup verdict instead of starting the clients again", func(t *testing.T) {
		t.Parallel()
		binDir := t.TempDir()
		startCountFile := filepath.Join(t.TempDir(), "starts")
		writeCountingFakeClient(t, binDir, "pg_dump", "pg_dump (PostgreSQL) 17.10", startCountFile)
		checker := NewChecker([]Bundle{fakeBundle(binDir, "17", "pg_dump")})
		checker.Check(t.Context())

		healthErr := checker.GetHealthError(t.Context())

		require.NoError(t, healthErr)
		startCount, err := os.ReadFile(startCountFile)
		require.NoError(t, err)
		assert.Equal(t, "x", string(startCount))
	})

	t.Run("reports a fatal bundle whose client was removed after startup", func(t *testing.T) {
		t.Parallel()
		binDir := t.TempDir()
		writeFakeClient(t, binDir, "pg_dump", "pg_dump (PostgreSQL) 17.10")
		checker := NewChecker([]Bundle{fakeBundle(binDir, "17", "pg_dump")})
		checker.Check(t.Context())
		require.NoError(t, os.Remove(filepath.Join(binDir, "pg_dump")))

		healthErr := checker.GetHealthError(t.Context())

		require.ErrorIs(t, healthErr, ErrClientToolsBroken)
		assert.Contains(t, healthErr.Error(), "client command not found")
	})

	t.Run("ignores a broken bundle that is not fatal", func(t *testing.T) {
		t.Parallel()
		bundle := fakeBundle(filepath.Join(t.TempDir(), "missing"), "8.0", "mysqldump")
		bundle.IsFatal = false
		checker := NewChecker([]Bundle{bundle})

		healthErr := checker.GetHealthError(t.Context())

		require.NoError(t, healthErr)
	})
}

var fakeVersionPattern = regexp.MustCompile(`(\d+\.\d+(\.\d+)?)`)

// The fake bundles report versions in several engine shapes, so the parser takes the last dotted
// number, which is the server line in each of them.
func parseLastDottedNumber(output string) (string, error) {
	matches := fakeVersionPattern.FindAllString(output, -1)
	if len(matches) == 0 {
		return "", fmt.Errorf("no version in %q", output)
	}

	return matches[len(matches)-1], nil
}

func fakeBundle(binDir, releaseLine, command string) Bundle {
	return Bundle{
		Engine:           "fake",
		Version:          releaseLine,
		BinDir:           binDir,
		RequiredCommands: []string{command},
		ReleaseLine:      releaseLine,
		ParseVersion:     parseLastDottedNumber,
		IsFatal:          true,
	}
}

func writeFakeClient(t *testing.T, dir, name, output string) {
	t.Helper()

	toolstub.Write(t, filepath.Join(dir, name), toolstub.Spec{Script: "echo " + strconv.Quote(output)})
}

func writeCountingFakeClient(t *testing.T, dir, name, output, startCountFile string) {
	t.Helper()

	toolstub.Write(t, filepath.Join(dir, name), toolstub.Spec{
		Script: "printf x >> " + strconv.Quote(startCountFile) + "\necho " + strconv.Quote(output),
	})
}
