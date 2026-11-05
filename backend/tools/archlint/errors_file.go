package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"path"
	"strings"
)

// checkErrorsFile keeps a package's sentinel errors and error types in its errors.go. A module's
// root package holds only module.go, so the errors other modules see are declared there.
func checkErrorsFile(sourceModule *sourceModule) []violation {
	errorTypesByPackage := collectErrorTypes(sourceModule)

	var violations []violation

	for _, file := range sourceModule.files {
		if file.isTest || isErrorsFile(file) {
			continue
		}

		for _, declaration := range file.syntax.Decls {
			general, isGeneral := declaration.(*ast.GenDecl)
			if !isGeneral {
				continue
			}

			for _, spec := range general.Specs {
				for _, name := range getErrorNames(general.Tok, spec, errorTypesByPackage[file.getDirectory()]) {
					violations = append(violations, violation{
						position: sourceModule.getPosition(file, name.Pos()),
						rule:     "errors-file",
						message: fmt.Sprintf(
							"error %s is declared in %s; a package declares its errors in errors.go",
							name.Name, file.relativePath,
						),
					})
				}
			}
		}
	}

	return violations
}

func isErrorsFile(file sourceFile) bool {
	if path.Base(file.relativePath) == "errors.go" {
		return true
	}

	moduleName, isModuleFile := getModuleName(file.relativePath)

	return isModuleFile && file.relativePath == path.Join("internal/modules", moduleName, "module.go")
}

func getErrorNames(tok token.Token, spec ast.Spec, errorTypes map[string]bool) []*ast.Ident {
	switch typed := spec.(type) {
	case *ast.ValueSpec:
		if tok != token.VAR {
			return nil
		}

		var names []*ast.Ident

		for _, name := range typed.Names {
			if isSentinelErrorName(name.Name) {
				names = append(names, name)
			}
		}

		return names
	case *ast.TypeSpec:
		if errorTypes[typed.Name.Name] {
			return []*ast.Ident{typed.Name}
		}
	}

	return nil
}

func isSentinelErrorName(name string) bool {
	rest, hasPrefix := strings.CutPrefix(name, "Err")
	if !hasPrefix {
		rest, hasPrefix = strings.CutPrefix(name, "err")
	}

	return hasPrefix && rest != "" && strings.ToUpper(rest[:1]) == rest[:1]
}

// collectErrorTypes finds, per package directory, the types that declare an Error() string method.
func collectErrorTypes(sourceModule *sourceModule) map[string]map[string]bool {
	errorTypesByPackage := map[string]map[string]bool{}

	for _, file := range sourceModule.files {
		if file.isTest {
			continue
		}

		for _, declaration := range file.syntax.Decls {
			function, isFunction := declaration.(*ast.FuncDecl)
			if !isFunction || function.Recv == nil || !isErrorMethod(function) {
				continue
			}

			if errorTypesByPackage[file.getDirectory()] == nil {
				errorTypesByPackage[file.getDirectory()] = map[string]bool{}
			}

			errorTypesByPackage[file.getDirectory()][getReceiverTypeName(function)] = true
		}
	}

	return errorTypesByPackage
}

func isErrorMethod(function *ast.FuncDecl) bool {
	if function.Name.Name != "Error" || function.Type.Params.NumFields() != 0 ||
		function.Type.Results.NumFields() != 1 {
		return false
	}

	result, isIdentifier := function.Type.Results.List[0].Type.(*ast.Ident)

	return isIdentifier && result.Name == "string"
}
