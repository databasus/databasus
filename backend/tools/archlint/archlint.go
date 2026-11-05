package main

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"
)

type checkSpec struct {
	entryPoints []entryPointSpec
	// Migrations up to this version predate the module table prefixes, so the cascade rule skips them.
	cascadeCutoff int64
}

func getBackendSpec() checkSpec {
	return checkSpec{
		entryPoints: []entryPointSpec{
			{path: "./cmd/databasus", forbiddenPaths: getTestLibraryPaths()},
			{path: "./cmd/verification-agent", forbiddenPaths: slices.Concat(getTestLibraryPaths(),
				[]string{"gorm.io", "github.com/gin-gonic/gin", "github.com/pressly/goose"})},
		},
		cascadeCutoff: 20261024122431,
	}
}

type violation struct {
	position string
	rule     string
	message  string
}

func (v violation) String() string {
	return fmt.Sprintf("%s: %s: %s", v.position, v.rule, v.message)
}

func check(root string, spec checkSpec) ([]violation, error) {
	sourceModule, err := loadModule(root)
	if err != nil {
		return nil, err
	}

	violations := slices.Concat(
		checkImports(sourceModule),
		checkModuleRoots(sourceModule),
		checkFileRoles(sourceModule),
		checkErrorsFile(sourceModule),
		checkBareGo(sourceModule),
	)

	cascadeViolations, err := checkCascades(root, filepath.Join(root, "migrations"), spec.cascadeCutoff)
	if err != nil {
		return nil, err
	}

	transactionViolations, err := checkMigrationTransactions(root, filepath.Join(root, "migrations"))
	if err != nil {
		return nil, err
	}

	dependencyViolations, err := checkProductionDependencies(root, spec.entryPoints)
	if err != nil {
		return nil, err
	}

	violations = slices.Concat(violations, cascadeViolations, transactionViolations, dependencyViolations)
	slices.SortFunc(violations, func(first, second violation) int {
		return strings.Compare(first.String(), second.String())
	})

	return slices.Compact(violations), nil
}
