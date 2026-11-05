package main

import (
	"testing"
)

func TestErrorsFile(t *testing.T) {
	t.Parallel()

	t.Run("fails when a module declares a sentinel error in its service file", func(t *testing.T) {
		t.Parallel()
		violations := checkFixture(t, map[string]string{
			"internal/modules/storages/module.go": "package storages\n",
			"internal/modules/storages/internal/storage/service.go": "package storage\n\n" +
				"import \"errors\"\n\nvar ErrStorageInUse = errors.New(\"storage is in use\")\n",
		})

		requireSingleViolation(t, violations, violationSpec{
			position: "internal/modules/storages/internal/storage/service.go:5",
			rule:     "errors-file",
			messageParts: []string{
				"ErrStorageInUse",
				"internal/modules/storages/internal/storage/service.go",
				"errors.go",
			},
		})
	})

	t.Run("fails when a module declares an error type outside errors.go", func(t *testing.T) {
		t.Parallel()
		violations := checkFixture(t, map[string]string{
			"internal/modules/storages/module.go": "package storages\n",
			"internal/modules/storages/internal/storage/storage.go": "package storage\n\n" +
				"type QuotaError struct{}\n\nfunc (e QuotaError) Error() string { return \"quota exceeded\" }\n",
		})

		requireSingleViolation(t, violations, violationSpec{
			position:     "internal/modules/storages/internal/storage/storage.go:3",
			rule:         "errors-file",
			messageParts: []string{"QuotaError"},
		})
	})
}

func TestErrorsOutsideFormerLayout(t *testing.T) {
	t.Parallel()
	t.Run("rejects an error declared outside errors.go in another backend package", func(t *testing.T) {
		t.Parallel()
		violations := checkFixture(
			t,
			map[string]string{
				"internal/other/service.go": "package other\nimport \"errors\"\nvar ErrFailure = errors.New(\"failure\")\n",
			},
		)
		requireSingleViolation(
			t,
			violations,
			violationSpec{
				position:     "internal/other/service.go:3",
				rule:         "errors-file",
				messageParts: []string{"ErrFailure"},
			},
		)
	})
}
