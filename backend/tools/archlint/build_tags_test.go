package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
