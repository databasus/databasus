package clienttools_test

import (
	"os"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/engines/clienttools"
	"databasus-backend/internal/engines/mariadb"
	"databasus-backend/internal/engines/mongodb"
	"databasus-backend/internal/engines/mysql"
	pgshared "databasus-backend/internal/engines/postgresql/shared"
)

func TestShippedBundles(t *testing.T) {
	t.Parallel()

	// Only gaps of the host's architecture can appear here, because the test runs on that host.
	t.Run("pass the check for every engine", func(t *testing.T) {
		t.Parallel()
		workingDir, err := os.Getwd()
		require.NoError(t, err)
		bundleRoot, err := clienttools.FindBundleRoot(workingDir)
		require.NoError(t, err)
		checker := clienttools.NewChecker(slices.Concat(
			pgshared.GetClientToolBundles(bundleRoot),
			mysql.GetClientToolBundles(bundleRoot),
			mariadb.GetClientToolBundles(bundleRoot),
			mongodb.GetClientToolBundles(bundleRoot),
		))

		for _, result := range checker.Check(t.Context()) {
			assert.Emptyf(
				t,
				result.Errors,
				"%s %s (%s)",
				result.Bundle.Engine,
				result.Bundle.Version,
				result.Bundle.BinDir,
			)
		}
	})
}
