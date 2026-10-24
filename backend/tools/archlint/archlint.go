package main

import (
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

type checkSpec struct {
	entryPoints []string
	// Migrations up to this version predate the module table prefixes, so the cascade rule skips them.
	cascadeCutoff int64
}

var backendSpec = checkSpec{
	entryPoints:   []string{"./cmd/databasus", "./cmd/verification-agent"},
	cascadeCutoff: 20261024122431,
}

// targetLayoutPattern is the Go part of "Backend layout" in docs/architecture.md; backend/.golangci.yml
// holds the same expression, and TestTargetLayoutPattern keeps the two equal.
var targetLayoutPattern = regexp.MustCompile(
	`^(api|cmd/databasus|cmd/verification-agent|e2e|internal/(app|modules|engines|lib|agent)|tools)/`,
)

type violation struct {
	position string
	rule     string
	message  string
}

func (v violation) String() string {
	return fmt.Sprintf("%s: %s: %s", v.position, v.rule, v.message)
}

func isInTargetLayout(relativePath string) bool {
	return targetLayoutPattern.MatchString(relativePath)
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
		checkBuildTags(sourceModule),
		checkProviderWrites(sourceModule),
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
