package physical

import (
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBaseBackupCodecRejection(t *testing.T) {
	t.Parallel()
	for _, scenario := range []struct {
		name       string
		stderr     string
		isRejected bool
	}{
		{"falls back when the source lacks zstd", "invalid compression specification: this build does not support compression with ZSTD", true},
		{"falls back when the source lacks gzip", "invalid compression specification: this build does not support compression with gzip", true},
		{"keeps a tool failure when the compression level is rejected", `invalid compression specification: compression algorithm "zstd" expects a compression level between 1 and 22 (default at 0)`, false},
		{"keeps a tool failure when the connection is rejected", "pg_basebackup: error: could not connect to server", false},
		{"keeps a tool failure when error output is empty", "", false},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()
			spec := newBaseBackupTestSpec(t)
			writeBaseBackupStub(t, spec, fmt.Sprintf("echo '%s' >&2; exit 1", scenario.stderr))
			_, err := NewSource(Settings{}).StreamBaseBackup(t.Context(), spec, io.Discard)
			var failure *BaseBackupError
			require.ErrorAs(t, err, &failure)
			assert.Equal(t, scenario.isRejected, failure.Kind == BaseBackupCodecRejected)
		})
	}
}

func TestCodecPreference(t *testing.T) {
	t.Parallel()
	t.Run("tries zstd then gzip then none when selecting compression", func(t *testing.T) {
		t.Parallel()
		codecs := GetCodecFallbackOrder()
		assert.Equal(t, []Codec{CodecZstd, CodecGzip, CodecNone}, codecs)
		arguments := make([]string, 0, len(codecs))
		for _, codec := range codecs {
			arguments = append(arguments, codec.GetCompressArgument())
		}
		assert.Equal(t, []string{"server-zstd:5", "server-gzip:6", "none"}, arguments)
	})
}
