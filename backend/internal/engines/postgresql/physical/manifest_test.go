package physical

import (
	"archive/tar"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWalkTarForManifest(t *testing.T) {
	t.Parallel()
	t.Run("excludes raw wal segments while keeping history and done files", func(t *testing.T) {
		t.Parallel()
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

		entries, err := walkTarForManifest(&compressed, CodecZstd)
		require.NoError(t, err)

		includedPaths := make([]string, 0, len(entries))
		byPath := make(map[string]manifestFileEntry, len(entries))
		for _, entry := range entries {
			includedPaths = append(includedPaths, entry.Path)
			byPath[entry.Path] = entry
		}

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
	})
}

func TestIsRawWalSegment(t *testing.T) {
	t.Parallel()
	t.Run("matches only raw segments with valid filenames", func(t *testing.T) {
		t.Parallel()
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
	})
}
