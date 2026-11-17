package mysqlfamily

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateConnectionValues(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		name            string
		values          ConnectionValues
		expectedSetting string
	}{
		{"accepts plain values", newValidConnectionValues(), ""},
		{"accepts a dash inside the database name", withDatabase("orders-archive"), ""},
		{"accepts URL separators in the database name", withDatabase("app?x=1&y=/z"), ""},
		{"accepts URL separators in the password", withPassword("p@ss/w?rd:1"), ""},
		{"refuses a database name that starts with a dash", withDatabase("--result-file=/tmp/out.sql"), "database"},
		{"refuses a database name of a single dash", withDatabase("-"), "database"},
		{"refuses a line break in the host", withHost("db.example.com\nresult-file=/tmp/out.sql"), "host"},
		{"refuses a carriage return in the user name", withUsername("backup\ruser"), "username"},
		{"refuses a line break in the password", withPassword("secret\nresult-file=/tmp/out.sql"), "password"},
		{"refuses a NUL in the database name", withDatabase("orders\x00"), "database"},
		{"refuses a DEL in the user name", withUsername("backup\x7f"), "username"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			err := testCase.values.Validate()

			if testCase.expectedSetting == "" {
				require.NoError(t, err)

				return
			}

			invalidSettingError, isInvalidSetting := errors.AsType[*InvalidSettingError](err)
			require.True(t, isInvalidSetting, err)
			assert.Equal(t, testCase.expectedSetting, invalidSettingError.Setting)
		})
	}
}

func newValidConnectionValues() ConnectionValues {
	return ConnectionValues{
		Host:     "db.example.com",
		Username: "backup_user",
		Password: "secret",
		Database: "orders",
	}
}

func withHost(host string) ConnectionValues {
	values := newValidConnectionValues()
	values.Host = host

	return values
}

func withUsername(username string) ConnectionValues {
	values := newValidConnectionValues()
	values.Username = username

	return values
}

func withPassword(password string) ConnectionValues {
	values := newValidConnectionValues()
	values.Password = password

	return values
}

func withDatabase(database string) ConnectionValues {
	values := newValidConnectionValues()
	values.Database = database

	return values
}
