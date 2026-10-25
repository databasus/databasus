package main

import (
	"fmt"
	"path"
)

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
