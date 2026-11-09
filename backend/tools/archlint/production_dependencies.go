package main

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"golang.org/x/tools/go/packages"
)

func getTestLibraryPaths() []string {
	return []string{"testing", "github.com/stretchr/testify", "github.com/testcontainers/testcontainers-go"}
}

type entryPointSpec struct {
	path           string
	forbiddenPaths []string
}

func (spec entryPointSpec) isForbiddenDependency(importPath string) bool {
	return slices.ContainsFunc(spec.forbiddenPaths, func(forbiddenPath string) bool {
		return hasPathPrefix(importPath, forbiddenPath)
	})
}

// Packages are loaded without build tags, exactly as the image builds them.
func checkProductionDependencies(root string, entryPoints []entryPointSpec) ([]violation, error) {
	var violations []violation

	for _, entryPoint := range entryPoints {
		if _, err := os.Stat(filepath.Join(root, entryPoint.path)); os.IsNotExist(err) {
			continue
		}

		entryViolations, err := checkEntryPointDependencies(root, entryPoint)
		if err != nil {
			return nil, err
		}

		violations = append(violations, entryViolations...)
	}

	return violations, nil
}

func checkEntryPointDependencies(root string, entryPoint entryPointSpec) ([]violation, error) {
	config := &packages.Config{
		Mode:       packages.NeedName | packages.NeedImports | packages.NeedDeps | packages.NeedFiles,
		Dir:        root,
		BuildFlags: []string{"-tags="},
	}

	loadedPackages, err := packages.Load(config, entryPoint.path)
	if err != nil {
		return nil, fmt.Errorf("load %s: %w", entryPoint.path, err)
	}

	if packages.PrintErrors(loadedPackages) > 0 {
		return nil, fmt.Errorf("load %s: %w", entryPoint.path, errPackagesHaveErrors)
	}

	var violations []violation

	packages.Visit(loadedPackages, nil, func(loadedPackage *packages.Package) {
		if entryPoint.isForbiddenDependency(loadedPackage.PkgPath) {
			return
		}

		for importPath := range loadedPackage.Imports {
			if !entryPoint.isForbiddenDependency(importPath) {
				continue
			}

			violations = append(violations, violation{
				position: getImportPosition(root, loadedPackage.GoFiles, importPath),
				rule:     "production-dependencies",
				message: fmt.Sprintf(
					"%s imports %s, which is forbidden in the production binary of %s",
					loadedPackage.PkgPath, importPath, entryPoint.path,
				),
			})
		}
	})

	return violations, nil
}

func getImportPosition(root string, goFiles []string, importPath string) string {
	fileSet := token.NewFileSet()

	for _, goFile := range goFiles {
		syntax, err := parser.ParseFile(fileSet, goFile, nil, parser.ImportsOnly)
		if err != nil {
			continue
		}

		for _, importSpec := range syntax.Imports {
			if unquoted, err := strconv.Unquote(importSpec.Path.Value); err != nil || unquoted != importPath {
				continue
			}

			return fmt.Sprintf("%s:%d", getDisplayPath(root, goFile), fileSet.Position(importSpec.Pos()).Line)
		}
	}

	return importPath
}

func getDisplayPath(root, filePath string) string {
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return filePath
	}

	relativePath, err := filepath.Rel(absoluteRoot, filePath)
	if err != nil || strings.HasPrefix(relativePath, "..") {
		return filePath
	}

	return filepath.ToSlash(relativePath)
}
