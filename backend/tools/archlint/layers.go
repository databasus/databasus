package main

import (
	"fmt"
	"path"
	"strconv"
	"strings"
)

type layer int

const (
	layerNone layer = iota
	layerLib
	layerEngines
	layerModules
	layerApp
	layerCmd
	layerAgent
)

func getLayer(relativePath string) layer {
	switch {
	case hasPathPrefix(relativePath, "internal/agent"), hasPathPrefix(relativePath, "cmd/verification-agent"):
		return layerAgent
	case hasPathPrefix(relativePath, "cmd/databasus"):
		return layerCmd
	case hasPathPrefix(relativePath, "internal/app"):
		return layerApp
	case hasPathPrefix(relativePath, "internal/modules"):
		return layerModules
	case hasPathPrefix(relativePath, "internal/engines"):
		return layerEngines
	case hasPathPrefix(relativePath, "internal/lib"):
		return layerLib
	default:
		return layerNone
	}
}

// getModuleName returns the domain module a path belongs to, or false outside internal/modules.
func getModuleName(relativePath string) (string, bool) {
	rest, isModulePath := strings.CutPrefix(relativePath, "internal/modules/")
	if !isModulePath || rest == "" {
		return "", false
	}

	name, _, _ := strings.Cut(rest, "/")

	return name, true
}

func hasPathPrefix(relativePath, prefix string) bool {
	return relativePath == prefix || strings.HasPrefix(relativePath, prefix+"/")
}

// checkImports enforces the layer direction, the module boundary and the agent's allowed imports on
// production files. Tests may reach further: a module's API tests build the whole app through apptest.
func checkImports(sourceModule *sourceModule) []violation {
	var violations []violation

	for _, file := range sourceModule.files {
		if file.isTest || !isInTargetLayout(file.relativePath) {
			continue
		}

		importerPackage := file.getDirectory()
		importerLayer := getLayer(importerPackage)

		for _, importSpec := range file.syntax.Imports {
			importPath, err := strconv.Unquote(importSpec.Path.Value)
			if err != nil {
				continue
			}

			importedPackage, isInModule := sourceModule.getRelativeImport(importPath)
			if !isInModule {
				continue
			}

			violations = append(violations, checkImport(importEdge{
				position:        sourceModule.getPosition(file, importSpec.Pos()),
				importerPackage: importerPackage,
				importerLayer:   importerLayer,
				importedPackage: importedPackage,
			})...)
		}
	}

	return violations
}

type importEdge struct {
	position        string
	importerPackage string
	importerLayer   layer
	importedPackage string
}

func checkImport(edge importEdge) []violation {
	importedLayer := getLayer(edge.importedPackage)

	if edge.importerLayer == layerAgent {
		if importedLayer == layerLib || importedLayer == layerEngines || importedLayer == layerAgent {
			return nil
		}

		return []violation{{
			position: edge.position,
			rule:     "agent-imports",
			message: fmt.Sprintf(
				"%s imports %s; the verification agent imports only internal/lib, internal/engines and internal/agent",
				edge.importerPackage, edge.importedPackage,
			),
		}}
	}

	if importedLayer == layerNone || edge.importerLayer == layerNone {
		return nil
	}

	if importedLayer == layerAgent || importedLayer > edge.importerLayer {
		return []violation{{
			position: edge.position,
			rule:     "layer",
			message: fmt.Sprintf(
				"%s imports %s; dependencies point cmd → app → modules → engines → lib, and only the agent imports the agent",
				edge.importerPackage,
				edge.importedPackage,
			),
		}}
	}

	importedModule, isModuleImport := getModuleName(edge.importedPackage)
	importerModule, _ := getModuleName(edge.importerPackage)

	if isModuleImport && importerModule != importedModule &&
		edge.importedPackage != path.Join("internal/modules", importedModule) {
		return []violation{{
			position: edge.position,
			rule:     "module-boundary",
			message: fmt.Sprintf(
				"%s imports %s; another package reaches the %s module only through internal/modules/%s",
				edge.importerPackage, edge.importedPackage, importedModule, importedModule,
			),
		}}
	}

	return nil
}

// checkModuleRoots keeps every production file but module.go out of a module's root package.
func checkModuleRoots(sourceModule *sourceModule) []violation {
	var violations []violation

	for _, file := range sourceModule.files {
		moduleName, isModuleFile := getModuleName(file.relativePath)
		if file.isTest || !isModuleFile || file.getDirectory() != path.Join("internal/modules", moduleName) {
			continue
		}

		if path.Base(file.relativePath) == "module.go" {
			continue
		}

		violations = append(violations, violation{
			position: sourceModule.getPosition(file, file.syntax.Package),
			rule:     "module-root",
			message: fmt.Sprintf(
				"%s is in the root package of the %s module, which holds only module.go; move it under internal/",
				file.relativePath, moduleName,
			),
		})
	}

	return violations
}
