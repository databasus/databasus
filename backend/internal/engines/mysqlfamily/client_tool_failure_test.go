package mysqlfamily

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsCompressionRejection(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		name                string
		stderr              string
		isExpectedRejection bool
	}{
		{
			name: "returns true on error 2066",
			stderr: "mysqldump: Got error: 2066: Connection failed due to wrongly " +
				"configured compression algorithm when trying to connect",
			isExpectedRejection: true,
		},
		{
			name:                "returns false on denied access",
			stderr:              "mysqldump: Got error: 1045: Access denied for user 'app'@'%' when trying to connect",
			isExpectedRejection: false,
		},
		{
			name:                "returns false on the missing probe table",
			stderr:              `mysqldump: Couldn't find table: "__databasus_compression_probe__"`,
			isExpectedRejection: false,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, testCase.isExpectedRejection, isCompressionRejection(testCase.stderr))
		})
	}
}
