package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProviderWrites(t *testing.T) {
	t.Parallel()

	t.Run("fails when backup code calls a provider's delete method directly", func(t *testing.T) {
		t.Parallel()
		violations := checkFixture(t, map[string]string{
			"internal/features/backups/cleaner.go": "package backups\n\n" +
				"type provider interface{ DeleteFile(name string) error }\n\n" +
				"func clean(storage provider) error {\n\treturn storage.DeleteFile(\"backup\")\n}\n",
		})

		requireSingleViolation(t, violations, violationSpec{
			position:     "internal/features/backups/cleaner.go:6",
			rule:         "provider-writes",
			messageParts: []string{"DeleteFile", "file store in internal/features/storages/files"},
		})
	})

	t.Run("passes when a backup test calls a provider's save method", func(t *testing.T) {
		t.Parallel()
		violations := checkFixture(t, map[string]string{
			"internal/features/backups/cleaner_test.go": "package backups\n\n" +
				"type provider interface{ SaveFile(name string) error }\n\n" +
				"func prepare(storage provider) error {\n\treturn storage.SaveFile(\"backup\")\n}\n",
		})

		assert.Empty(t, violations)
	})
}
