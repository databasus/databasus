package mailer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetDefaultSecurityForPort(t *testing.T) {
	t.Parallel()

	t.Run("returns tls only for port 465", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, SecurityTLS, GetDefaultSecurityForPort(465))
		assert.Equal(t, SecurityStartTLS, GetDefaultSecurityForPort(587))
		assert.Equal(t, SecurityStartTLS, GetDefaultSecurityForPort(25))
		assert.Equal(t, SecurityStartTLS, GetDefaultSecurityForPort(2525))
	})
}
