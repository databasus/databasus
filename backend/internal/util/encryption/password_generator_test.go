package encryption

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateComplexPassword(t *testing.T) {
	t.Parallel()

	t.Run("returns 24 characters with every required character class", func(t *testing.T) {
		t.Parallel()

		password := GenerateComplexPassword()

		assert.Len(t, password, 24)
		assert.True(t, strings.ContainsAny(password, "abcdefghijklmnopqrstuvwxyz"))
		assert.True(t, strings.ContainsAny(password, "ABCDEFGHIJKLMNOPQRSTUVWXYZ"))
		assert.True(t, strings.ContainsAny(password, "0123456789"))
		assert.True(t, strings.ContainsAny(password, "!@#$%^&*()-_=+"))
	})
}
