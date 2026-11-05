package physical

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/engines/postgresql/physical/walmath"
)

func TestSerializeManifest(t *testing.T) {
	t.Parallel()
	t.Run("preserves the postgres format and checksum with a wal range", func(t *testing.T) {
		t.Parallel()
		modTime := time.Date(2026, 5, 29, 12, 0, 0, 0, time.UTC)
		labelChecksum := strings.Repeat("a", 64)
		relationChecksum := strings.Repeat("b", 64)

		serialized, err := serializeManifest(serializeManifestInput{
			Files: []manifestFileEntry{
				{Path: "backup_label", Size: 227, ModTime: modTime, ChecksumHex: labelChecksum},
				{Path: "base/1/1259", Size: 8192, ModTime: modTime, ChecksumHex: relationChecksum},
			},
			SystemID:   7361234567890123456,
			TimelineID: 1,
			StartLSN:   walmath.LSN(0x3000060),
			StopLSN:    walmath.LSN(0x3000220),
		})
		require.NoError(t, err)

		// The body is hand-written here (independent of the implementation) so a
		// one-byte layout drift fails the test; the checksum is then computed over
		// exactly that body, mirroring PG's still_checksumming boundary.
		manifestBody := `{ "PostgreSQL-Backup-Manifest-Version": 2,` + "\n" +
			`"System-Identifier": 7361234567890123456,` + "\n" +
			`"Files": [` + "\n" +
			`{ "Path": "backup_label", "Size": 227, "Last-Modified": "2026-05-29 12:00:00 GMT", "Checksum-Algorithm": "SHA256", "Checksum": "` + labelChecksum + `" }` + ",\n" +
			`{ "Path": "base/1/1259", "Size": 8192, "Last-Modified": "2026-05-29 12:00:00 GMT", "Checksum-Algorithm": "SHA256", "Checksum": "` + relationChecksum + `" }` + "\n" +
			`],` + "\n" +
			`"WAL-Ranges": [` + "\n" +
			`{ "Timeline": 1, "Start-LSN": "0/3000060", "End-LSN": "0/3000220" }` + "\n" +
			`],` + "\n"

		bodyChecksum := sha256.Sum256([]byte(manifestBody))
		manifestText := manifestBody + `"Manifest-Checksum": "` + hex.EncodeToString(bodyChecksum[:]) + `"}` + "\n"

		assert.Equal(t, manifestText, string(serialized))
	})
}

func TestEscapeJSONString(t *testing.T) {
	t.Parallel()
	t.Run("matches postgres escaping with special characters", func(t *testing.T) {
		t.Parallel()
		cases := []struct {
			raw      string
			expected string
		}{
			{"plain", `"plain"`},
			{`a"b`, `"a\"b"`},
			{`a\b`, `"a\\b"`},
			{"line1\nline2", `"line1\nline2"`},
			{"tab\there", `"tab\there"`},
			{"\b\f\r", `"\b\f\r"`},
			{"<tag>&amp;", `"<tag>&amp;"`}, // PG does NOT HTML-escape (encoding/json would)
		}

		for _, testCase := range cases {
			escaped, err := escapeJSONString(testCase.raw)
			require.NoError(t, err, testCase.raw)
			assert.Equal(t, testCase.expected, escaped, testCase.raw)
		}
	})
}

func TestEscapeJSONControlBytes(t *testing.T) {
	t.Parallel()
	t.Run("escapes control bytes with unicode escapes", func(t *testing.T) {
		t.Parallel()
		escaped, err := escapeJSONString(string([]byte{0x01, 0x1f}))
		require.NoError(t, err)

		// Expected literal: "". Built from raw bytes (0x5c is the
		// backslash byte) so the source carries no backslash escapes to misencode.
		encodedControlBytes := string([]byte{'"', 0x5c, 'u', '0', '0', '0', '1', 0x5c, 'u', '0', '0', '1', 'f', '"'})
		assert.Equal(t, encodedControlBytes, escaped)
	})
}

func TestEscapeJSONInvalidUTF8(t *testing.T) {
	t.Parallel()
	t.Run("returns an error with invalid utf8", func(t *testing.T) {
		t.Parallel()
		_, err := escapeJSONString(string([]byte{0xff, 0xfe}))
		require.Error(t, err)
	})
}
