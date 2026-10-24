package main

import (
	"go/ast"
)

const oldBackupCodeDirectory = "internal/features/backups"

// checkProviderWrites keeps the old backup code from calling a storage provider's own save or delete
// method. Every stored file goes through the file store, which is what keeps a failed backup from
// leaving a file behind. Tests may call providers to prepare data.
func checkProviderWrites(sourceModule *sourceModule) []violation {
	var violations []violation

	for _, file := range sourceModule.files {
		if file.isTest || !hasPathPrefix(file.getDirectory(), oldBackupCodeDirectory) {
			continue
		}

		ast.Inspect(file.syntax, func(node ast.Node) bool {
			call, isCall := node.(*ast.CallExpr)
			if !isCall {
				return true
			}

			selector, isSelector := call.Fun.(*ast.SelectorExpr)
			if !isSelector || (selector.Sel.Name != "SaveFile" && selector.Sel.Name != "DeleteFile") {
				return true
			}

			violations = append(violations, violation{
				position: sourceModule.getPosition(file, call.Pos()),
				rule:     "provider-writes",
				message: "backup code calls " + selector.Sel.Name +
					" on a storage provider; write and delete stored files through the file store in internal/features/storages/files",
			})

			return true
		})
	}

	return violations
}
