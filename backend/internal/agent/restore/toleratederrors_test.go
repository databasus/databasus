package restore

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const missingAuxExtensionsStderr = `pg_restore: error: could not execute query: ERROR:  extension "pg_stat_kcache" is not available
DETAIL:  Could not open extension control file "/usr/share/postgresql/16/extension/pg_stat_kcache.control": No such file or directory.
HINT:  The extension must first be installed on the system where PostgreSQL is running.
Command was: CREATE EXTENSION IF NOT EXISTS pg_stat_kcache WITH SCHEMA public;


pg_restore: error: could not execute query: ERROR:  extension "pg_stat_kcache" does not exist
Command was: COMMENT ON EXTENSION pg_stat_kcache IS 'Kernel statistics gathering';


pg_restore: error: could not execute query: ERROR:  extension "set_user" is not available
DETAIL:  Could not open extension control file "/usr/share/postgresql/16/extension/set_user.control": No such file or directory.
HINT:  The extension must first be installed on the system where PostgreSQL is running.
Command was: CREATE EXTENSION IF NOT EXISTS set_user WITH SCHEMA public;


pg_restore: error: could not execute query: ERROR:  extension "set_user" does not exist
Command was: COMMENT ON EXTENSION set_user IS 'similar to SET ROLE but with added logging';


pg_restore: warning: errors ignored on restore: 4
`

// A missing data-bearing extension (PostGIS) cascades into type/relation errors
// that are NOT extension-related — the data did not restore, so this must not be
// tolerated.
const postgisCascadeStderr = `pg_restore: error: could not execute query: ERROR:  extension "postgis" is not available
DETAIL:  Could not open extension control file "/usr/share/postgresql/13/extension/postgis.control": No such file or directory.
Command was: CREATE EXTENSION IF NOT EXISTS postgis WITH SCHEMA public;


pg_restore: error: could not execute query: ERROR:  type "public.geometry" does not exist
Command was: CREATE TABLE public.places (geom public.geometry);


pg_restore: error: could not execute query: ERROR:  relation "public.places" does not exist
Command was: COPY public.places (geom) FROM stdin;


pg_restore: warning: errors ignored on restore: 3
`

// An archive dumped with -n public carries the schema definition, and the restore target already
// owns public (issue #726).
const preexistingPublicSchemaStderr = `pg_restore: error: could not execute query: ERROR:  schema "public" already exists
Command was: CREATE SCHEMA public;


pg_restore: warning: errors ignored on restore: 1
`

func TestIsToleratedErrorsOnly(t *testing.T) {
	t.Run("returns true when all ignored errors concern missing extensions", func(t *testing.T) {
		assert.True(t, IsToleratedErrorsOnly(missingAuxExtensionsStderr))
	})
	t.Run("returns false when cascade data errors are present", func(t *testing.T) {
		assert.False(t, IsToleratedErrorsOnly(postgisCascadeStderr))
	})
	t.Run("returns false when there is no ignored error count", func(t *testing.T) {
		fatal := `pg_restore: error: could not execute query: ERROR:  extension "set_user" is not available
Command was: CREATE EXTENSION IF NOT EXISTS set_user WITH SCHEMA public;
pg_restore: error: aborting because of errors`

		assert.False(t, IsToleratedErrorsOnly(fatal))
	})
	t.Run("returns false when the visible error count is below the ignored count", func(t *testing.T) {
		truncated := `pg_restore: error: could not execute query: ERROR:  extension "set_user" is not available
Command was: CREATE EXTENSION IF NOT EXISTS set_user WITH SCHEMA public;
pg_restore: error: could not execute query: ERROR:  extension "set_user" does not exist
Command was: COMMENT ON EXTENSION set_user IS 'x';
pg_restore: warning: errors ignored on restore: 5`

		assert.False(t, IsToleratedErrorsOnly(truncated),
			"a truncated tail that cannot account for all N errors must not be tolerated")
	})
	t.Run("returns false when an unrelated error is present", func(t *testing.T) {
		mixed := `pg_restore: error: could not execute query: ERROR:  extension "set_user" is not available
Command was: CREATE EXTENSION IF NOT EXISTS set_user WITH SCHEMA public;
pg_restore: error: could not execute query: ERROR:  duplicate key value violates unique constraint "places_pkey"
Command was: COPY public.places (id) FROM stdin;
pg_restore: warning: errors ignored on restore: 2`

		assert.False(t, IsToleratedErrorsOnly(mixed))
	})
	t.Run("returns false when the input is empty", func(t *testing.T) {
		assert.False(t, IsToleratedErrorsOnly(""))
	})
	t.Run("returns true when the only error concerns the existing public schema", func(t *testing.T) {
		assert.True(t, IsToleratedErrorsOnly(preexistingPublicSchemaStderr))
	})
	t.Run("returns true when errors concern missing extensions and the existing public schema", func(t *testing.T) {
		combined := `pg_restore: error: could not execute query: ERROR:  schema "public" already exists
Command was: CREATE SCHEMA public;
pg_restore: error: could not execute query: ERROR:  extension "set_user" is not available
Command was: CREATE EXTENSION IF NOT EXISTS set_user WITH SCHEMA public;
pg_restore: warning: errors ignored on restore: 2`

		assert.True(t, IsToleratedErrorsOnly(combined))
	})
	t.Run("returns false when a relation already exists", func(t *testing.T) {
		relationClash := `pg_restore: error: could not execute query: ERROR:  relation "public.t_a" already exists
Command was: CREATE TABLE public.t_a (id integer NOT NULL);
pg_restore: warning: errors ignored on restore: 1`

		assert.False(t, IsToleratedErrorsOnly(relationClash),
			"tolerating a pre-existing public schema must not become a blanket already-exists rule")
	})
}

func TestExtractUnavailableExtensions(t *testing.T) {
	t.Run("returns sorted deduped names", func(t *testing.T) {
		assert.Equal(t,
			[]string{"pg_stat_kcache", "set_user"},
			ExtractUnavailableExtensions(missingAuxExtensionsStderr))
	})
}
