package main

import (
	"go/ast"
)

// checkBareGo keeps bare go statements inside internal/lib, whose helpers tie a goroutine to a
// context and recover its panic.
func checkBareGo(sourceModule *sourceModule) []violation {
	var violations []violation

	for _, file := range sourceModule.files {
		if file.isTest || !isInTargetLayout(file.relativePath) || hasPathPrefix(file.getDirectory(), "internal/lib") {
			continue
		}

		ast.Inspect(file.syntax, func(node ast.Node) bool {
			if statement, isGo := node.(*ast.GoStmt); isGo {
				violations = append(violations, violation{
					position: sourceModule.getPosition(file, statement.Pos()),
					rule:     "bare-go",
					message:  "a bare go statement belongs only in internal/lib; start the goroutine through its helpers",
				})
			}

			return true
		})
	}

	return violations
}
