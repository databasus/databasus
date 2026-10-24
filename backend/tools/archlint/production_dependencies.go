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

var testLibraryPaths = []string{"testing", "github.com/stretchr/testify", "github.com/testcontainers/testcontainers-go"}

// acceptedThirdPartyTestImporters are dependencies outside our control whose production code imports
// a test library. Each entry names why the package is linked; any other importer fails the check.
var acceptedThirdPartyTestImporters = map[string]string{
	"github.com/Files-com/files-sdk-go/v3/lib": "rclone's Files.com backend links the SDK's lib package, " +
		"which keeps assertion helpers in production code",
	"github.com/rclone/Proton-API-Bridge": "rclone's Proton Drive backend links the bridge, " +
		"which imports testing in production code",
}

func isTestLibrary(importPath string) bool {
	return slices.ContainsFunc(testLibraryPaths, func(libraryPath string) bool {
		return hasPathPrefix(importPath, libraryPath)
	})
}

// checkProductionDependencies fails when a production entry point links a test library. Packages are
// loaded without build tags, exactly as the image builds them.
func checkProductionDependencies(root string, entryPoints []string) ([]violation, error) {
	var violations []violation

	for _, entryPoint := range entryPoints {
		if _, err := os.Stat(filepath.Join(root, entryPoint)); os.IsNotExist(err) {
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

func checkEntryPointDependencies(root, entryPoint string) ([]violation, error) {
	config := &packages.Config{
		Mode:       packages.NeedName | packages.NeedImports | packages.NeedDeps | packages.NeedFiles,
		Dir:        root,
		BuildFlags: []string{"-tags="},
	}

	loadedPackages, err := packages.Load(config, entryPoint)
	if err != nil {
		return nil, fmt.Errorf("load %s: %w", entryPoint, err)
	}

	if packages.PrintErrors(loadedPackages) > 0 {
		return nil, fmt.Errorf("load %s: %w", entryPoint, errPackagesHaveErrors)
	}

	var violations []violation

	packages.Visit(loadedPackages, nil, func(loadedPackage *packages.Package) {
		if isTestLibrary(loadedPackage.PkgPath) || acceptedThirdPartyTestImporters[loadedPackage.PkgPath] != "" {
			return
		}

		for importPath := range loadedPackage.Imports {
			if !isTestLibrary(importPath) {
				continue
			}

			violations = append(violations, violation{
				position: getImportPosition(root, loadedPackage.GoFiles, importPath),
				rule:     "production-dependencies",
				message: fmt.Sprintf(
					"%s imports %s, so the production binary of %s links a test library; "+
						"move the code into a test file or a package only tests import",
					loadedPackage.PkgPath, importPath, entryPoint,
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
