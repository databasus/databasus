package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestImports(t *testing.T) {
	t.Parallel()

	t.Run("fails when a module imports another module's internals", func(t *testing.T) {
		t.Parallel()
		violations := checkFixture(t, map[string]string{
			"internal/modules/storages/module.go":               "package storages\n",
			"internal/modules/storages/internal/files/store.go": "package files\n",
			"internal/modules/backups/module.go":                "package backups\n",
			"internal/modules/backups/internal/backuping/backuper.go": "package backuping\n\n" +
				"import _ \"fixture/internal/modules/storages/internal/files\"\n",
		})

		requireSingleViolation(t, violations, violationSpec{
			position: "internal/modules/backups/internal/backuping/backuper.go:3",
			rule:     "module-boundary",
			messageParts: []string{
				"internal/modules/backups/internal/backuping imports internal/modules/storages/internal/files",
			},
		})
	})

	t.Run("fails when the shared library imports a module", func(t *testing.T) {
		t.Parallel()
		violations := checkFixture(t, map[string]string{
			"internal/modules/users/module.go": "package users\n",
			"internal/lib/audit/audit.go":      "package audit\n\nimport _ \"fixture/internal/modules/users\"\n",
		})

		requireSingleViolation(t, violations, violationSpec{
			position: "internal/lib/audit/audit.go:3",
			rule:     "layer",
			messageParts: []string{
				"internal/lib/audit imports internal/modules/users",
				"cmd → app → modules → engines → lib",
			},
		})
	})

	t.Run("fails when the agent imports a module", func(t *testing.T) {
		t.Parallel()
		violations := checkFixture(t, map[string]string{
			"internal/modules/backups/module.go": "package backups\n",
			"internal/agent/verifier/verifier.go": "package verifier\n\n" +
				"import _ \"fixture/internal/modules/backups\"\n",
		})

		requireSingleViolation(t, violations, violationSpec{
			position: "internal/agent/verifier/verifier.go:3",
			rule:     "agent-imports",
			messageParts: []string{
				"internal/agent/verifier imports internal/modules/backups",
				"imports only internal/lib, internal/engines and internal/agent",
			},
		})
	})

	t.Run("passes for a clean module", func(t *testing.T) {
		t.Parallel()
		violations := checkFixture(t, getCleanModuleFiles())

		assert.Empty(t, violations)
	})
}

func TestCommandImports(t *testing.T) {
	t.Parallel()
	for _, commandName := range []string{"seed_dev", "other"} {
		t.Run("rejects agent implementation imports from "+commandName, func(t *testing.T) {
			t.Parallel()
			violations := checkFixture(t, map[string]string{
				"internal/agent/worker/worker.go": "package worker\n",
				"cmd/" + commandName + "/main.go": "package main\nimport _ \"fixture/internal/agent/worker\"\n",
			})
			requireSingleViolation(
				t,
				violations,
				violationSpec{
					position:     "cmd/" + commandName + "/main.go:2",
					rule:         "layer",
					messageParts: []string{"only the agent imports the agent"},
				},
			)
		})
	}
}

func TestUnclassifiedImports(t *testing.T) {
	t.Parallel()

	t.Run("rejects agent implementation when the importer has no classified layer", func(t *testing.T) {
		t.Parallel()
		violations := checkFixture(t, map[string]string{
			"internal/agent/config/config.go": "package config\n",
			"internal/other/service.go":       "package other\n\nimport _ \"fixture/internal/agent/config\"\n",
		})

		requireSingleViolation(t, violations, violationSpec{
			position:     "internal/other/service.go:3",
			rule:         "layer",
			messageParts: []string{"only the agent imports the agent"},
		})
	})

	t.Run("rejects module internals when the importer has no classified layer", func(t *testing.T) {
		t.Parallel()
		violations := checkFixture(t, map[string]string{
			"internal/modules/users/module.go":                  "package users\n",
			"internal/modules/users/internal/access/service.go": "package access\n",
			"internal/other/service.go": "package other\n\n" +
				"import _ \"fixture/internal/modules/users/internal/access\"\n",
		})

		requireSingleViolation(t, violations, violationSpec{
			position:     "internal/other/service.go:3",
			rule:         "module-boundary",
			messageParts: []string{"only through internal/modules/users"},
		})
	})

	t.Run("accepts a module contract when the importer has no classified layer", func(t *testing.T) {
		t.Parallel()
		violations := checkFixture(t, map[string]string{
			"internal/modules/users/module.go": "package users\n",
			"internal/other/service.go":        "package other\n\nimport _ \"fixture/internal/modules/users\"\n",
		})

		assert.Empty(t, violations)
	})
}
