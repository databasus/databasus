package main

import (
	"fmt"
	"go/build"
	"path"
	"path/filepath"
)

func checkModuleContractUsage(sourceModule *sourceModule) ([]violation, error) {
	hasCandidates, err := hasContractCandidates(sourceModule)
	if err != nil || !hasCandidates {
		return nil, err
	}

	loaded, fileSet, err := loadContractPackages(sourceModule.root)
	if err != nil {
		return nil, err
	}

	root, err := filepath.Abs(sourceModule.root)
	if err != nil {
		return nil, fmt.Errorf("resolve contract root: %w", err)
	}

	references := newContractReferences(root, fileSet)
	for _, loadedPackage := range loaded {
		references.addDeclarations(loadedPackage)
	}

	for _, loadedPackage := range loaded {
		references.addReferences(loadedPackage)
	}

	consumed := references.getConsumedDeclarations()
	var violations []violation
	for identity, declaration := range references.declarations {
		if declaration.isCandidate && !consumed[identity] {
			violations = append(violations, violation{
				position: declaration.position,
				rule:     "module-contract-usage",
				message: fmt.Sprintf(
					"%s has no consumer outside the %s module and is not required by a consumed contract declaration",
					declaration.name,
					declaration.moduleName,
				),
			})
		}
	}

	return violations, nil
}

func hasContractCandidates(sourceModule *sourceModule) (bool, error) {
	for _, file := range sourceModule.files {
		moduleName, isModule := getModuleName(file.relativePath)
		if !isModule || file.relativePath != path.Join("internal/modules", moduleName, "module.go") {
			continue
		}

		for _, declaration := range getContractDeclarations(file.syntax) {
			if !declaration.isCandidate {
				continue
			}

			isActive, err := build.Default.MatchFile(filepath.Join(sourceModule.root, file.getDirectory()), "module.go")
			if err != nil {
				return false, fmt.Errorf("select contract %s: %w", file.relativePath, err)
			}

			if isActive {
				return true, nil
			}

			break
		}
	}

	return false, nil
}
