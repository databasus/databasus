package pgrestore

import (
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetFailureDiagnostic(t *testing.T) {
	t.Parallel()

	for _, section := range []string{"DETAIL:", "CONTEXT:", "Command was:"} {
		t.Run("omits record-looking body lines when an unstructured "+section+" section starts", func(t *testing.T) {
			t.Parallel()
			primaryError := "pg_restore: error: could not execute query: ERROR: permission denied for language plpython3u"
			output := primaryError + "\n" + section + " CREATE FUNCTION private_body() RETURNS text AS $private$\n" +
				"ERROR: error-shaped-private-value\nFATAL: fatal-shaped-private-value\n" +
				"pg_restore: error: restore-shaped-private-value\n" + diagnosticSectionMarker +
				"\nmarker-shaped-private-value\nstill-private-body\n$private$ LANGUAGE sql;\n" +
				"pg_restore: warning: errors ignored on restore: 1"

			diagnostic := GetFailureDiagnostic(FailureDiagnosticSpec{Output: output, Secrets: nil, IsTruncated: false})

			require.NotNil(t, diagnostic)
			assert.Equal(t, primaryError+"\n"+diagnosticSectionMarker, diagnostic.Output)

			repeatedDiagnostic := GetFailureDiagnostic(
				FailureDiagnosticSpec{Output: diagnostic.Output, Secrets: nil, IsTruncated: diagnostic.IsTruncated},
			)

			assert.Equal(t, diagnostic, repeatedDiagnostic)
		})
	}

	t.Run("preserves the primary errors when sensitive multiline sections are removed twice", func(t *testing.T) {
		t.Parallel()
		output := "pg_restore: error: type geometry does not exist\npg_restore: error: table locations does not exist\nDETAIL: row secret-row\ncontinued-row\nCONTEXT: COPY users\nmore-row\nCommand was: CREATE FUNCTION private_fn()\nSQL body\npg_restore: warning: errors ignored on restore: 1\n"

		diagnostic := GetFailureDiagnostic(FailureDiagnosticSpec{Output: output, Secrets: nil, IsTruncated: false})

		require.NotNil(t, diagnostic)
		assert.Contains(t, diagnostic.Output, "type geometry does not exist")
		assert.Contains(t, diagnostic.Output, "table locations does not exist")
		for _, privateValue := range []string{"secret-row", "continued-row", "more-row", "SQL body", "CREATE FUNCTION"} {
			assert.NotContains(t, diagnostic.Output, privateValue)
		}

		repeatedDiagnostic := GetFailureDiagnostic(
			FailureDiagnosticSpec{Output: diagnostic.Output, Secrets: nil, IsTruncated: diagnostic.IsTruncated},
		)

		assert.Equal(t, diagnostic, repeatedDiagnostic)
	})

	t.Run("masks credentials and removes control sequences before truncating", func(t *testing.T) {
		t.Parallel()
		credential := "exact-known-credential"
		output := "\x1b[31mpg_restore: error: postgres://name:password@db/test token=secret\x1b[0m\r\n" +
			strings.Repeat(
				"x",
				MaxFailureDiagnosticBytes/2-10,
			) + credential + strings.Repeat("z", MaxFailureDiagnosticBytes)

		diagnostic := GetFailureDiagnostic(
			FailureDiagnosticSpec{Output: output, Secrets: []string{credential, ""}, IsTruncated: false},
		)

		require.NotNil(t, diagnostic)
		assert.True(t, diagnostic.IsTruncated)
		assert.LessOrEqual(t, len(diagnostic.Output), MaxFailureDiagnosticBytes)
		for _, privateValue := range []string{credential, "password", "token=secret", "\x1b", "\r"} {
			assert.NotContains(t, diagnostic.Output, privateValue)
		}
		assert.Contains(t, diagnostic.Output, "pg_restore: error:")
	})

	for _, outputBytes := range []int{MaxFailureDiagnosticBytes - 1, MaxFailureDiagnosticBytes, MaxFailureDiagnosticBytes + 1} {
		t.Run(
			fmt.Sprintf("bounds output when it has %d bytes", outputBytes),
			func(t *testing.T) {
				t.Parallel()
				output := strings.Repeat("a", outputBytes)

				diagnostic := GetFailureDiagnostic(
					FailureDiagnosticSpec{Output: output, Secrets: nil, IsTruncated: false},
				)

				require.NotNil(t, diagnostic)
				assert.Equal(t, outputBytes > MaxFailureDiagnosticBytes, diagnostic.IsTruncated)
				assert.LessOrEqual(t, len(diagnostic.Output), MaxFailureDiagnosticBytes)
				if outputBytes <= MaxFailureDiagnosticBytes {
					assert.Equal(t, output, diagnostic.Output)
				}
			},
		)
	}

	for _, boundary := range []string{"beginning", "end"} {
		t.Run(
			"masks both credential fragments when the credential crosses the "+boundary+" truncation boundary",
			func(t *testing.T) {
				t.Parallel()
				credential := "private-credential-left-half-right-half"
				fragmentBytes := len(credential) / 2
				retainedBytes := (MaxFailureDiagnosticBytes - len(diagnosticOmissionMarker)) / 2
				outputBytes := MaxFailureDiagnosticBytes + 4096
				credentialStart := retainedBytes - fragmentBytes
				if boundary == "end" {
					credentialStart = outputBytes - retainedBytes - fragmentBytes
				}
				output := strings.Repeat(
					"a",
					credentialStart,
				) + credential + strings.Repeat(
					"z",
					outputBytes-credentialStart-len(credential),
				)
				boundaryFragment := credential[:fragmentBytes]
				if boundary == "end" {
					boundaryFragment = credential[fragmentBytes:]
				}
				unredactedDiagnostic := GetFailureDiagnostic(
					FailureDiagnosticSpec{Output: output, Secrets: nil, IsTruncated: false},
				)
				require.NotNil(t, unredactedDiagnostic)
				require.Contains(t, unredactedDiagnostic.Output, boundaryFragment)

				diagnostic := GetFailureDiagnostic(
					FailureDiagnosticSpec{Output: output, Secrets: []string{credential}, IsTruncated: false},
				)

				require.NotNil(t, diagnostic)
				assert.True(t, diagnostic.IsTruncated)
				assert.NotContains(t, diagnostic.Output, credential[:fragmentBytes])
				assert.NotContains(t, diagnostic.Output, credential[fragmentBytes:])
				assert.LessOrEqual(t, len(diagnostic.Output), MaxFailureDiagnosticBytes)
			},
		)
	}

	t.Run("keeps valid UTF-8 and both ends when multibyte output exceeds the limit", func(t *testing.T) {
		t.Parallel()
		output := "first error\n" + strings.Repeat("\u754c", MaxFailureDiagnosticBytes) + "\nlast error"

		diagnostic := GetFailureDiagnostic(FailureDiagnosticSpec{Output: output, Secrets: nil, IsTruncated: false})

		require.NotNil(t, diagnostic)
		assert.True(t, utf8.ValidString(diagnostic.Output))
		assert.LessOrEqual(t, len(diagnostic.Output), MaxFailureDiagnosticBytes)
		assert.True(t, strings.HasPrefix(diagnostic.Output, "first error"))
		assert.True(t, strings.HasSuffix(diagnostic.Output, "last error"))
		assert.Contains(t, diagnostic.Output, diagnosticOmissionMarker)
	})

	t.Run("omits the diagnostic when normalization leaves only whitespace", func(t *testing.T) {
		t.Parallel()
		assert.Nil(t, GetFailureDiagnostic(FailureDiagnosticSpec{Output: "\x00 \n", Secrets: nil, IsTruncated: false}))
	})

	t.Run("preserves truncation when received output contains invalid UTF-8", func(t *testing.T) {
		t.Parallel()

		diagnostic := GetFailureDiagnostic(FailureDiagnosticSpec{Output: "error\xff", Secrets: nil, IsTruncated: true})

		require.NotNil(t, diagnostic)
		assert.True(t, utf8.ValidString(diagnostic.Output))
		assert.True(t, diagnostic.IsTruncated)
	})
}
