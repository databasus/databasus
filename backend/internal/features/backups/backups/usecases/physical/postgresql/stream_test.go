package usecases_physical_postgresql

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestManifestSaveTimeout(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name             string
		payloadSizeBytes int
		timeout          time.Duration
	}{
		{"uses the minimum timeout with an empty manifest", 0, manifestSaveMinTimeout},
		{"uses the minimum timeout with a 1 mib manifest", 1024 * 1024, manifestSaveMinTimeout},
		{"scales the timeout with a 100 mib manifest", 100 * 1024 * 1024, 400 * time.Second},
		{"caps the timeout with a 1 gib manifest", 1024 * 1024 * 1024, manifestSaveMaxTimeout},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.timeout, manifestSaveTimeout(testCase.payloadSizeBytes))
		})
	}
}
