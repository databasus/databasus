package main

import (
	"maps"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const fixtureCascadeCutoff = 20261024122431

var fixtureSpec = checkSpec{entryPoints: []string{"./cmd/databasus"}, cascadeCutoff: fixtureCascadeCutoff}

// fakeTestifyFiles stand in for testify through a replace directive, so a fixture module builds offline.
var fakeTestifyFiles = map[string]string{
	"go.mod": "module fixture\n\ngo 1.26\n\nrequire github.com/stretchr/testify v1.0.0\n\n" +
		"replace github.com/stretchr/testify => ./testify\n",
	"testify/go.mod":           "module github.com/stretchr/testify\n\ngo 1.26\n",
	"testify/assert/assert.go": "package assert\n\nfunc True(value bool) bool { return value }\n",
}

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

	t.Run("passes for a clean module", func(t *testing.T) {
		t.Parallel()
		violations := checkFixture(t, cleanModuleFiles)

		assert.Empty(t, violations)
	})
}

func TestFileRoles(t *testing.T) {
	t.Parallel()

	t.Run("fails when a service method is declared in a handler file", func(t *testing.T) {
		t.Parallel()
		violations := checkFixture(t, map[string]string{
			"internal/modules/storages/module.go": "package storages\n",
			"internal/modules/storages/internal/httpapi/handler.go": "package httpapi\n\n" +
				"type StorageService struct{}\n\nfunc (s *StorageService) Delete() {}\n",
		})

		requireSingleViolation(t, violations, violationSpec{
			position:     "internal/modules/storages/internal/httpapi/handler.go:5",
			rule:         "file-role",
			messageParts: []string{"StorageService.Delete", "handler.go", "service.go or *_service.go"},
		})
	})

	t.Run("fails when an unexported service method is declared in a handler file", func(t *testing.T) {
		t.Parallel()
		violations := checkFixture(t, map[string]string{
			"internal/modules/storages/module.go": "package storages\n",
			"internal/modules/storages/internal/httpapi/handler.go": "package httpapi\n\n" +
				"type service struct{}\n\nfunc (s *service) Delete() {}\n",
		})

		requireSingleViolation(t, violations, violationSpec{
			position:     "internal/modules/storages/internal/httpapi/handler.go:5",
			rule:         "file-role",
			messageParts: []string{"service.Delete", "handler.go", "service.go or *_service.go"},
		})
	})

	t.Run("fails when an unexported repository method is declared in a service file", func(t *testing.T) {
		t.Parallel()
		violations := checkFixture(t, map[string]string{
			"internal/modules/storages/module.go": "package storages\n",
			"internal/modules/storages/internal/storage/service.go": "package storage\n\n" +
				"type repository struct{}\n\nfunc (r *repository) Find() {}\n",
		})

		requireSingleViolation(t, violations, violationSpec{
			position:     "internal/modules/storages/internal/storage/service.go:5",
			rule:         "file-role",
			messageParts: []string{"repository.Find", "service.go", "repository.go or *_repository.go"},
		})
	})

	t.Run("fails when a service method is declared in a file whose name only ends in service.go", func(t *testing.T) {
		t.Parallel()
		violations := checkFixture(t, map[string]string{
			"internal/modules/storages/module.go": "package storages\n",
			"internal/modules/storages/internal/storage/storageservice.go": "package storage\n\n" +
				"type StorageService struct{}\n\nfunc (s *StorageService) Delete() {}\n",
		})

		requireSingleViolation(t, violations, violationSpec{
			position:     "internal/modules/storages/internal/storage/storageservice.go:5",
			rule:         "file-role",
			messageParts: []string{"StorageService.Delete", "storageservice.go"},
		})
	})

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

func TestBuildTags(t *testing.T) {
	t.Parallel()

	t.Run("fails when a module has a file with the testhelpers build tag", func(t *testing.T) {
		t.Parallel()
		violations := checkFixture(t, map[string]string{
			"internal/modules/storages/module.go":                   "package storages\n",
			"internal/modules/storages/internal/storage/testing.go": "//go:build testhelpers\n\npackage storage\n",
		})

		requireSingleViolation(t, violations, violationSpec{
			position:     "internal/modules/storages/internal/storage/testing.go:1",
			rule:         "build-tag",
			messageParts: []string{"internal/modules/storages/internal/storage/testing.go", "testhelpers"},
		})
	})

	t.Run("passes when an old package has a file with the testhelpers build tag", func(t *testing.T) {
		t.Parallel()
		violations := checkFixture(t, map[string]string{
			"internal/features/storages/testing.go": "//go:build testhelpers\n\npackage storages\n",
		})

		assert.Empty(t, violations)
	})
}

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

func TestCascades(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		migrationName  string
		statement      string
		isViolation    bool
		messageSnippet string
	}{
		{
			name:          "fails when a new migration cascades between two modules",
			migrationName: "20261101000000_add_backups_logical.sql",
			statement: "ALTER TABLE backups_logical\n    ADD CONSTRAINT fk_backups_logical_storage\n" +
				"    FOREIGN KEY (storage_id)\n    REFERENCES storages (id)\n    ON DELETE CASCADE;\n",
			isViolation:    true,
			messageSnippet: "fk_backups_logical_storage cascades deletes from backups_logical (owner backups) to storages",
		},
		{
			name:          "fails when a new migration cascades from a backups table to a shared library table",
			migrationName: "20261101000000_add_backups_audit.sql",
			statement: "CREATE TABLE backups_audit (\n    event_id UUID NOT NULL REFERENCES lib_audit_events (id) " +
				"ON DELETE CASCADE\n);\n",
			isViolation:    true,
			messageSnippet: "to lib_audit_events (owner lib_audit)",
		},
		{
			name:          "passes when a new migration cascades between two tables of one lib owner",
			migrationName: "20261101000000_add_lib_audit_details.sql",
			statement: "CREATE TABLE lib_audit_details (\n    event_id UUID NOT NULL REFERENCES lib_audit_events (id) " +
				"ON DELETE CASCADE\n);\n",
		},
		{
			name:          "passes when a new migration cascades inside one module",
			migrationName: "20261101000000_add_storages_s3.sql",
			statement: "ALTER TABLE storages_s3\n    ADD CONSTRAINT fk_storages_s3_storage\n" +
				"    FOREIGN KEY (storage_id)\n    REFERENCES storages (id)\n    ON DELETE CASCADE;\n",
		},
		{
			name:          "passes when a new engine table without a module owner cascades to databases",
			migrationName: "20261101000000_add_oracle_databases.sql",
			statement: "CREATE TABLE oracle_databases (\n    database_id UUID REFERENCES databases(id) " +
				"ON DELETE CASCADE\n);\n",
		},
		{
			name:          "passes when a migration before the cutoff cascades between two modules",
			migrationName: "20250101000000_add_backups_logical.sql",
			statement: "ALTER TABLE backups_logical\n    ADD CONSTRAINT fk_backups_logical_storage\n" +
				"    FOREIGN KEY (storage_id)\n    REFERENCES storages (id)\n    ON DELETE CASCADE;\n",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			migration := "-- +goose Up\n-- +goose StatementBegin\n\n" + testCase.statement +
				"\n-- +goose StatementEnd\n\n-- +goose Down\nALTER TABLE storages_s3 ADD CONSTRAINT down_only " +
				"FOREIGN KEY (storage_id) REFERENCES backups (id) ON DELETE CASCADE;\n"
			violations := checkFixture(t, map[string]string{"migrations/" + testCase.migrationName: migration})

			if !testCase.isViolation {
				assert.Empty(t, violations)

				return
			}

			requireSingleViolation(t, violations, violationSpec{
				position:     "migrations/" + testCase.migrationName + ":5",
				rule:         "cascade",
				messageParts: []string{testCase.messageSnippet, "ON DELETE RESTRICT"},
			})
		})
	}
}

func TestMigrationTransactions(t *testing.T) {
	t.Parallel()

	t.Run("fails when a migration runs without a transaction", func(t *testing.T) {
		t.Parallel()
		violations := checkFixture(t, map[string]string{
			"migrations/20250101000000_add_index.sql": "-- +goose NO TRANSACTION\n-- +goose Up\n" +
				"CREATE INDEX CONCURRENTLY idx_backups_created_at ON backups (created_at);\n",
		})

		requireSingleViolation(t, violations, violationSpec{
			position:     "migrations/20250101000000_add_index.sql:1",
			rule:         "migration-transaction",
			messageParts: []string{"NO TRANSACTION"},
		})
	})

	t.Run("passes when a migration runs in its transaction", func(t *testing.T) {
		t.Parallel()
		violations := checkFixture(t, map[string]string{
			"migrations/20250101000000_add_index.sql": "-- +goose Up\n" +
				"CREATE INDEX idx_backups_created_at ON backups (created_at);\n",
		})

		assert.Empty(t, violations)
	})
}

var cleanModuleFiles = map[string]string{
	"internal/lib/database/database.go": "package database\n\ntype Pool struct{}\n",
	"internal/modules/users/module.go":  "package users\n\ntype Module interface{ GetActor() string }\n",
	"internal/modules/storages/module.go": "package storages\n\nimport (\n\t\"errors\"\n\n" +
		"\t_ \"fixture/internal/modules/users\"\n)\n\ntype Deps struct{}\n\n" +
		"var ErrStorageMissing = errors.New(\"storage missing\")\n",
	"internal/modules/storages/internal/storage/errors.go": "package storage\n\n" +
		"type QuotaError struct{}\n\nfunc (e QuotaError) Error() string { return \"quota exceeded\" }\n",
	"internal/modules/storages/internal/storage/service.go": "package storage\n\n" +
		"import _ \"fixture/internal/lib/database\"\n\ntype StorageService struct{}\n\n" +
		"func (s *StorageService) Delete() {}\n",
	"internal/modules/storages/internal/storage/storage_repository.go": "package storage\n\n" +
		"type storageRepository struct{}\n\nfunc (r *storageRepository) Delete() {}\n",
	"internal/modules/storages/internal/storage/service_test.go": "package storage\n\n" +
		"import \"testing\"\n\nfunc TestDelete(t *testing.T) {\n\tgo func() {}()\n}\n",
}

func withFakeTestify(files map[string]string) map[string]string {
	maps.Copy(files, fakeTestifyFiles)

	return files
}

// withFakeThirdPartyModule adds a module at modulePath, replaced by a local directory, whose
// production file imports testing.
func withFakeThirdPartyModule(modulePath string, files map[string]string) map[string]string {
	files["go.mod"] = "module fixture\n\ngo 1.26\n\nrequire " + modulePath + " v1.0.0\n\n" +
		"replace " + modulePath + " => ./" + modulePath + "\n"
	files[modulePath+"/go.mod"] = "module " + modulePath + "\n\ngo 1.26\n"
	files[modulePath+"/sdk.go"] = "package sdk\n\nimport _ \"testing\"\n"

	return files
}

func checkFixture(t *testing.T, files map[string]string) []violation {
	t.Helper()

	root := t.TempDir()

	if _, hasGoMod := files["go.mod"]; !hasGoMod {
		require.NoError(t, os.WriteFile(filepath.Join(root, "go.mod"), []byte("module fixture\n\ngo 1.26\n"), 0o600))
	}

	for filePath, content := range files {
		fullPath := filepath.Join(root, filepath.FromSlash(filePath))
		require.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0o750))
		require.NoError(t, os.WriteFile(fullPath, []byte(content), 0o600))
	}

	violations, err := check(root, fixtureSpec)
	require.NoError(t, err)

	return violations
}

type violationSpec struct {
	position     string
	rule         string
	messageParts []string
}

func requireSingleViolation(t *testing.T, violations []violation, spec violationSpec) {
	t.Helper()

	require.Len(t, violations, 1, "violations: %v", violations)
	reportedViolation := violations[0]
	assert.Equal(t, spec.position, reportedViolation.position)
	assert.Equal(t, spec.rule, reportedViolation.rule)

	for _, messagePart := range spec.messageParts {
		assert.Contains(t, reportedViolation.message, messagePart)
	}
}
