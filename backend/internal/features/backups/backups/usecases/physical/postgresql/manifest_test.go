package usecases_physical_postgresql

import (
	"archive/tar"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/engines/postgresql/physical/walmath"
	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
)

func Test_SerializeManifest_ProducesParseableVersion2Manifest(t *testing.T) {
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
	expectedBody := `{ "PostgreSQL-Backup-Manifest-Version": 2,` + "\n" +
		`"System-Identifier": 7361234567890123456,` + "\n" +
		`"Files": [` + "\n" +
		`{ "Path": "backup_label", "Size": 227, "Last-Modified": "2026-05-29 12:00:00 GMT", "Checksum-Algorithm": "SHA256", "Checksum": "` + labelChecksum + `" }` + ",\n" +
		`{ "Path": "base/1/1259", "Size": 8192, "Last-Modified": "2026-05-29 12:00:00 GMT", "Checksum-Algorithm": "SHA256", "Checksum": "` + relationChecksum + `" }` + "\n" +
		`],` + "\n" +
		`"WAL-Ranges": [` + "\n" +
		`{ "Timeline": 1, "Start-LSN": "0/3000060", "End-LSN": "0/3000220" }` + "\n" +
		`],` + "\n"

	bodyChecksum := sha256.Sum256([]byte(expectedBody))
	expected := expectedBody + `"Manifest-Checksum": "` + hex.EncodeToString(bodyChecksum[:]) + `"}` + "\n"

	assert.Equal(t, expected, string(serialized))
}

func Test_WalkTarForManifest_ExcludesRawWalSegments_IncludesHistoryAndDone(t *testing.T) {
	labelBody := []byte("backup-label-contents")

	var tarBuffer bytes.Buffer
	tarWriter := tar.NewWriter(&tarBuffer)

	writeRegular := func(name string, body []byte) {
		require.NoError(t, tarWriter.WriteHeader(&tar.Header{
			Typeflag: tar.TypeReg,
			Name:     name,
			Mode:     0o600,
			Size:     int64(len(body)),
			ModTime:  time.Unix(1_700_000_000, 0).UTC(),
		}))
		_, writeErr := tarWriter.Write(body)
		require.NoError(t, writeErr)
	}

	writeRegular("backup_label", labelBody)
	require.NoError(t, tarWriter.WriteHeader(&tar.Header{
		Typeflag: tar.TypeDir, Name: "base/", Mode: 0o700,
	}))
	writeRegular("base/1/1259", bytes.Repeat([]byte("x"), 8192))
	writeRegular("global/pg_control", []byte("control"))
	writeRegular("pg_wal/000000010000000000000001", bytes.Repeat([]byte("w"), 1024))
	writeRegular("pg_wal/archive_status/000000010000000000000001.done", nil)
	writeRegular("pg_wal/00000002.history", []byte("1\t0/3000000\tno recovery target\n"))
	require.NoError(t, tarWriter.Close())

	var compressed bytes.Buffer
	encoder, err := zstd.NewWriter(&compressed)
	require.NoError(t, err)
	_, err = encoder.Write(tarBuffer.Bytes())
	require.NoError(t, err)
	require.NoError(t, encoder.Close())

	entries, err := walkTarForManifest(&compressed, physical_enums.PhysicalBackupCompressionZstd)
	require.NoError(t, err)

	includedPaths := make([]string, 0, len(entries))
	byPath := make(map[string]manifestFileEntry, len(entries))
	for _, entry := range entries {
		includedPaths = append(includedPaths, entry.Path)
		byPath[entry.Path] = entry
	}

	// Raw WAL segment and the directory are excluded; status/history files are in.
	assert.Equal(t, []string{
		"backup_label",
		"base/1/1259",
		"global/pg_control",
		"pg_wal/archive_status/000000010000000000000001.done",
		"pg_wal/00000002.history",
	}, includedPaths, "Files[] must keep tar order, drop raw WAL segments + dirs, keep .done/.history")

	labelDigest := sha256.Sum256(labelBody)
	assert.Equal(t, hex.EncodeToString(labelDigest[:]), byPath["backup_label"].ChecksumHex)
	assert.Equal(t, int64(len(labelBody)), byPath["backup_label"].Size)
}

func Test_IsRawWalSegment_MatchesOnlySegments(t *testing.T) {
	cases := []struct {
		path     string
		expected bool
	}{
		{"pg_wal/000000010000000000000001", true},
		{"pg_wal/0000000A0000000B0000000C", true},
		{"pg_wal/00000002.history", false},
		{"pg_wal/archive_status/000000010000000000000001.done", false},
		{"pg_wal/00000001000000000000000G", false}, // G is not hex
		{"pg_wal/0000000100000000000000", false},   // too short
		{"base/1/1259", false},
		{"backup_label", false},
	}

	for _, testCase := range cases {
		assert.Equal(t, testCase.expected, isRawWalSegment(testCase.path), testCase.path)
	}
}

func Test_EscapeJSONString_MatchesPostgresEscapeJson(t *testing.T) {
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
}

func Test_EscapeJSONString_EscapesControlBytesAsUnicode(t *testing.T) {
	escaped, err := escapeJSONString(string([]byte{0x01, 0x1f}))
	require.NoError(t, err)

	// Expected literal: "". Built from raw bytes (0x5c is the
	// backslash byte) so the source carries no backslash escapes to misencode.
	expected := string([]byte{'"', 0x5c, 'u', '0', '0', '0', '1', 0x5c, 'u', '0', '0', '1', 'f', '"'})
	assert.Equal(t, expected, escaped)
}

func Test_EscapeJSONString_WhenNonUTF8_ReturnsError(t *testing.T) {
	_, err := escapeJSONString(string([]byte{0xff, 0xfe}))
	require.Error(t, err)
}
