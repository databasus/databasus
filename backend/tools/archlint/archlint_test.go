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

func getFixtureSpec() checkSpec {
	return checkSpec{entryPoints: getBackendSpec().entryPoints, cascadeCutoff: fixtureCascadeCutoff}
}

func getFakeTestifyFiles() map[string]string {
	return map[string]string{
		"go.mod": "module fixture\n\ngo 1.26\n\nrequire github.com/stretchr/testify v1.0.0\n\n" +
			"replace github.com/stretchr/testify => ./testify\n",
		"testify/go.mod":           "module github.com/stretchr/testify\n\ngo 1.26\n",
		"testify/assert/assert.go": "package assert\n\nfunc True(value bool) bool { return value }\n",
	}
}

func getCleanModuleFiles() map[string]string {
	return map[string]string{
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
}

func withFakeTestify(files map[string]string) map[string]string {
	maps.Copy(files, getFakeTestifyFiles())

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

	violations, err := check(writeFixture(t, files), getFixtureSpec())
	require.NoError(t, err)

	return violations
}

func writeFixture(t *testing.T, files map[string]string) string {
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

	return root
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
