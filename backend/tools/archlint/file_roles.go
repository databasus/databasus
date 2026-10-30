package main

import (
	"fmt"
	"go/ast"
	"path"
	"strings"
)

func getFileRoles() []string {
	return []string{"Handler", "Service", "Repository"}
}

// checkFileRoles keeps the methods of a handler, a service or a repository in a file of that role.
// Generated code is left out: oapi-codegen puts the strict handler of a server into api.gen.go.
func checkFileRoles(sourceModule *sourceModule) []violation {
	var violations []violation

	for _, file := range sourceModule.files {
		if file.isTest || ast.IsGenerated(file.syntax) {
			continue
		}

		fileName := path.Base(file.relativePath)

		for _, declaration := range file.syntax.Decls {
			function, isFunction := declaration.(*ast.FuncDecl)
			if !isFunction || function.Recv == nil {
				continue
			}

			receiverType := getReceiverTypeName(function)

			for _, role := range getFileRoles() {
				roleFile := strings.ToLower(role) + ".go"
				if !hasRole(receiverType, role) || isRoleFile(fileName, roleFile) {
					continue
				}

				violations = append(violations, violation{
					position: sourceModule.getPosition(file, function.Pos()),
					rule:     "file-role",
					message: fmt.Sprintf(
						"method %s.%s is declared in %s; methods of a %s belong in %s or *_%s",
						receiverType, function.Name.Name, fileName, strings.ToLower(role), roleFile, roleFile,
					),
				})
			}
		}
	}

	return violations
}

// hasRole matches StorageService, storageRepository and the bare unexported names service, handler
// and repository.
func hasRole(typeName, role string) bool {
	return strings.HasSuffix(typeName, role) || typeName == strings.ToLower(role)
}

func isRoleFile(fileName, roleFile string) bool {
	return fileName == roleFile || strings.HasSuffix(fileName, "_"+roleFile)
}
