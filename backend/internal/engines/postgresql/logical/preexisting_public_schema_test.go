package logical

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const publicSchemaClashStderr = `pg_restore: error: could not execute query: ERROR:  schema "public" already exists
Command was: CREATE SCHEMA public;


pg_restore: warning: errors ignored on restore: 1
`

func TestIsPreexistingPublicSchemaOnly(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		name             string
		stderr           string
		isExpectedResult bool
	}{
		{
			name:             "returns true when the only ignored error is the existing public schema",
			stderr:           publicSchemaClashStderr,
			isExpectedResult: true,
		},
		{
			// A user-facing restore that silently dropped an extension is still a failed restore.
			name: "returns false when a missing extension is ignored too",
			stderr: `pg_restore: error: could not execute query: ERROR:  schema "public" already exists
Command was: CREATE SCHEMA public;
pg_restore: error: could not execute query: ERROR:  extension "postgis" is not available
Command was: CREATE EXTENSION IF NOT EXISTS postgis WITH SCHEMA public;
pg_restore: warning: errors ignored on restore: 2`,
			isExpectedResult: false,
		},
		{
			// Error output that cannot account for all N errors must not be tolerated.
			name: "returns false when fewer errors are visible than the marker counts",
			stderr: `pg_restore: error: could not execute query: ERROR:  schema "public" already exists
Command was: CREATE SCHEMA public;
pg_restore: warning: errors ignored on restore: 4`,
			isExpectedResult: false,
		},
		{
			name: "returns false without the ignored errors marker",
			stderr: `pg_restore: error: could not execute query: ERROR:  schema "public" already exists
Command was: CREATE SCHEMA public;
pg_restore: error: aborting because of errors`,
			isExpectedResult: false,
		},
		{
			name:             "returns false with empty error output",
			stderr:           "",
			isExpectedResult: false,
		},
		{
			name: "returns false when a relation already exists",
			stderr: `pg_restore: error: could not execute query: ERROR:  relation "public.t_a" already exists
Command was: CREATE TABLE public.t_a (id integer NOT NULL);
pg_restore: warning: errors ignored on restore: 1`,
			isExpectedResult: false,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, testCase.isExpectedResult, isPreexistingPublicSchemaOnly(testCase.stderr))
		})
	}
}
