package main

import (
	"testing"
)

func TestModuleRoots(t *testing.T) {
	t.Parallel()

	t.Run("fails when a module's root package holds a file other than module.go", func(t *testing.T) {
		t.Parallel()
		violations := checkFixture(t, map[string]string{
			"internal/modules/storages/module.go":  "package storages\n",
			"internal/modules/storages/service.go": "package storages\n",
		})

		requireSingleViolation(t, violations, violationSpec{
			position:     "internal/modules/storages/service.go:1",
			rule:         "module-root",
			messageParts: []string{"internal/modules/storages/service.go", "holds only module.go"},
		})
	})
}
