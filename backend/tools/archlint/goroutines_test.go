package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBareGo(t *testing.T) {
	t.Parallel()

	t.Run("fails when a module service starts a goroutine with a bare go statement", func(t *testing.T) {
		t.Parallel()
		violations := checkFixture(t, map[string]string{
			"internal/modules/backups/module.go": "package backups\n",
			"internal/modules/backups/internal/backuping/service.go": "package backuping\n\n" +
				"type BackupService struct{}\n\nfunc (s *BackupService) Start() {\n\tgo func() {}()\n}\n",
		})

		requireSingleViolation(t, violations, violationSpec{
			position:     "internal/modules/backups/internal/backuping/service.go:6",
			rule:         "bare-go",
			messageParts: []string{"belongs only in internal/lib"},
		})
	})

	t.Run("passes when the shared library starts a goroutine", func(t *testing.T) {
		t.Parallel()
		violations := checkFixture(t, map[string]string{
			"internal/lib/background/runner.go": "package background\n\nfunc Go(work func()) {\n\tgo work()\n}\n",
		})

		assert.Empty(t, violations)
	})
}
