package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProductionDependencies(t *testing.T) {
	t.Parallel()

	t.Run("fails when production code reachable from the entry point imports an assertion library", func(t *testing.T) {
		t.Parallel()
		violations := checkFixture(t, withFakeTestify(map[string]string{
			"cmd/databasus/main.go": "package main\n\nimport _ \"fixture/internal/features/helper\"\n\nfunc main() {}\n",
			"internal/features/helper/helper.go": "package helper\n\n" +
				"import \"github.com/stretchr/testify/assert\"\n\nvar IsReady = assert.True(true)\n",
		}))

		requireSingleViolation(t, violations, violationSpec{
			position:     "internal/features/helper/helper.go:3",
			rule:         "production-dependencies",
			messageParts: []string{"fixture/internal/features/helper imports github.com/stretchr/testify/assert"},
		})
	})

	t.Run("fails when a third-party package outside the accepted list imports testing", func(t *testing.T) {
		t.Parallel()
		violations := checkFixture(t, withFakeThirdPartyModule("example.com/sdk", map[string]string{
			"cmd/databasus/main.go": "package main\n\nimport _ \"example.com/sdk\"\n\nfunc main() {}\n",
		}))

		requireSingleViolation(t, violations, violationSpec{
			position:     "example.com/sdk/sdk.go:3",
			rule:         "production-dependencies",
			messageParts: []string{"example.com/sdk imports testing"},
		})
	})

	t.Run("passes when an accepted third-party importer imports testing", func(t *testing.T) {
		t.Parallel()
		violations := checkFixture(t, withFakeThirdPartyModule("github.com/rclone/Proton-API-Bridge", map[string]string{
			"cmd/databasus/main.go": "package main\n\nimport _ \"github.com/rclone/Proton-API-Bridge\"\n\nfunc main() {}\n",
		}))

		assert.Empty(t, violations)
	})

	t.Run("passes when the entry point imports nothing from tests", func(t *testing.T) {
		t.Parallel()
		violations := checkFixture(t, withFakeTestify(map[string]string{
			"cmd/databasus/main.go":            "package main\n\nimport _ \"fixture/internal/features/clock\"\n\nfunc main() {}\n",
			"internal/features/clock/clock.go": "package clock\n\nimport \"time\"\n\nvar Started = time.Now()\n",
		}))

		assert.Empty(t, violations)
	})

	t.Run("passes when only a test file of the entry point imports a helper that imports testify", func(t *testing.T) {
		t.Parallel()
		violations := checkFixture(t, withFakeTestify(map[string]string{
			"cmd/databasus/main.go": "package main\n\nfunc main() {}\n",
			"cmd/databasus/main_test.go": "package main\n\nimport (\n\t\"testing\"\n\n\t\"fixture/internal/lib/testhelper\"\n)\n\n" +
				"func TestMain(m *testing.M) { testhelper.Prepare() }\n",
			"internal/lib/testhelper/testhelper.go": "package testhelper\n\n" +
				"import \"github.com/stretchr/testify/assert\"\n\nfunc Prepare() { assert.True(true) }\n",
		}))

		assert.Empty(t, violations)
	})
}

func TestAgentProductionDependencies(t *testing.T) {
	t.Parallel()
	t.Run("fails with the importer when an agent dependency imports GORM", func(t *testing.T) {
		t.Parallel()
		violations := checkFixture(t, withFakeServerFrameworks(map[string]string{
			"cmd/verification-agent/main.go":                "package main\n\nimport _ \"fixture/internal/agent/restore\"\n\nfunc main() {}\n",
			"internal/agent/restore/restorer.go":            "package restore\n\nimport _ \"fixture/internal/engines/postgresql/logical\"\n",
			"internal/engines/postgresql/logical/source.go": "package logical\n\nimport _ \"fixture/internal/lib/database\"\n",
			"internal/lib/database/database.go":             "package database\n\nimport _ \"gorm.io/gorm\"\n",
		}))
		requireSingleViolation(t, violations, violationSpec{
			position:     "internal/lib/database/database.go:3",
			rule:         "production-dependencies",
			messageParts: []string{"fixture/internal/lib/database imports gorm.io/gorm", "./cmd/verification-agent"},
		})
	})
	t.Run("passes when an agent dependency imports only the standard library", func(t *testing.T) {
		t.Parallel()
		violations := checkFixture(t, map[string]string{
			"cmd/verification-agent/main.go":                        "package main\n\nimport _ \"fixture/internal/agent/restore\"\n\nfunc main() {}\n",
			"internal/agent/restore/restorer.go":                    "package restore\n\nimport _ \"fixture/internal/engines/postgresql/logical/pgrestore\"\n",
			"internal/engines/postgresql/logical/pgrestore/args.go": "package pgrestore\n\nimport _ \"strconv\"\n",
		})
		assert.Empty(t, violations)
	})
	t.Run("passes when the server imports GORM Gin and goose", func(t *testing.T) {
		t.Parallel()
		violations := checkFixture(t, withFakeServerFrameworks(map[string]string{
			"cmd/databasus/main.go": "package main\n\nimport (\n_ \"gorm.io/gorm\"\n_ \"github.com/gin-gonic/gin\"\n_ \"github.com/pressly/goose\"\n)\n\nfunc main() {}\n",
		}))
		assert.Empty(t, violations)
	})
}

func withFakeServerFrameworks(files map[string]string) map[string]string {
	files["go.mod"] = "module fixture\n\ngo 1.26\n"
	for _, modulePath := range []string{"gorm.io/gorm", "github.com/gin-gonic/gin", "github.com/pressly/goose"} {
		files["go.mod"] += "\nrequire " + modulePath + " v1.0.0\nreplace " + modulePath + " => ./" + modulePath + "\n"
		files[modulePath+"/go.mod"] = "module " + modulePath + "\n\ngo 1.26\n"
		files[modulePath+"/framework.go"] = "package framework\n"
	}
	files["gorm.io/gorm/framework.go"] = "package framework\n\nimport _ \"gorm.io/gorm/clause\"\n"
	files["gorm.io/gorm/clause/clause.go"] = "package clause\n"

	return files
}
