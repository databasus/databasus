package filename

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_SanitizeFilename_ReplacesSpecialCharacters(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "replaces spaces with underscores",
			input:    "my database name",
			expected: "my_database_name",
		},
		{
			name:     "replaces forward slashes",
			input:    "db/prod/main",
			expected: "db-prod-main",
		},
		{
			name:     "replaces backslashes",
			input:    "db\\prod\\main",
			expected: "db-prod-main",
		},
		{
			name:     "replaces colons",
			input:    "db:production:main",
			expected: "db-production-main",
		},
		{
			name:     "replaces asterisks",
			input:    "db*wildcard",
			expected: "db-wildcard",
		},
		{
			name:     "replaces question marks",
			input:    "db?query",
			expected: "db-query",
		},
		{
			name:     "replaces double quotes",
			input:    "db\"quoted\"name",
			expected: "db-quoted-name",
		},
		{
			name:     "replaces less than signs",
			input:    "db<redirect",
			expected: "db-redirect",
		},
		{
			name:     "replaces greater than signs",
			input:    "db>output",
			expected: "db-output",
		},
		{
			name:     "replaces pipes",
			input:    "db|pipe",
			expected: "db-pipe",
		},
		{
			name:     "replaces multiple different special characters",
			input:    "my db:/backup\\file*2024?",
			expected: "my_db--backup-file-2024-",
		},
		{
			name:     "handles all special characters at once",
			input:    " /\\:*?\"<>|",
			expected: "_---------",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeFilename(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func Test_SanitizeFilename_HandlesEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string returns empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "string with no special characters remains unchanged",
			input:    "simple_database_name",
			expected: "simple_database_name",
		},
		{
			name:     "string with hyphens and underscores remains unchanged",
			input:    "my-database_name-123",
			expected: "my-database_name-123",
		},
		{
			name:     "preserves alphanumeric characters",
			input:    "Database123ABC",
			expected: "Database123ABC",
		},
		{
			name:     "preserves dots and parentheses",
			input:    "db.production.(v2)",
			expected: "db.production.(v2)",
		},
		{
			name:     "handles unicode characters",
			input:    "\u0431\u0430\u0437\u0430_\u0434\u0430\u043d\u043d\u044b\u0445_\u30c6\u30b9\u30c8",
			expected: "\u0431\u0430\u0437\u0430_\u0434\u0430\u043d\u043d\u044b\u0445_\u30c6\u30b9\u30c8",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeFilename(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func Test_SanitizeFilename_WindowsReservedNames(t *testing.T) {
	// Windows reserved names are case-insensitive: CON, PRN, AUX, NUL, COM1-COM9, LPT1-LPT9
	// Our function doesn't handle these specifically because:
	// 1. Database names in our system are typically lowercase
	// 2. These are combined with timestamps and UUIDs in filenames (e.g., "CON-20240102-150405-uuid")
	// 3. The timestamp and UUID suffix make the final filename safe on Windows

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "CON remains as CON (will be safe with timestamp suffix)",
			input:    "CON",
			expected: "CON",
		},
		{
			name:     "PRN remains as PRN (will be safe with timestamp suffix)",
			input:    "PRN",
			expected: "PRN",
		},
		{
			name:     "COM1 remains as COM1 (will be safe with timestamp suffix)",
			input:    "COM1",
			expected: "COM1",
		},
		{
			name:     "handles database name with reserved name as part",
			input:    "my:CON/database",
			expected: "my-CON-database",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeFilename(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func Test_SanitizeFilename_RealWorldExamples(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "production database with environment",
			input:    "prod:main/db",
			expected: "prod-main-db",
		},
		{
			name:     "database with spaces and version",
			input:    "My App Database v2.0",
			expected: "My_App_Database_v2.0",
		},
		{
			name:     "database with special query chars",
			input:    "analytics?region=us*",
			expected: "analytics-region=us-",
		},
		{
			name:     "windows-style path in database name",
			input:    "C:\\databases\\prod",
			expected: "C--databases-prod",
		},
		{
			name:     "unix-style path in database name",
			input:    "/var/lib/postgres/main",
			expected: "-var-lib-postgres-main",
		},
		{
			name:     "database name with quotes",
			input:    "\"production\" database",
			expected: "-production-_database",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeFilename(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
