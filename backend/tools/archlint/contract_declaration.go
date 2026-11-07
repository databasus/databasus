package main

import (
	"go/ast"
	"go/token"
)

type contractDeclaration struct {
	name        *ast.Ident
	surfaces    []ast.Node
	isCandidate bool
}

func getContractDeclarations(syntax *ast.File) []contractDeclaration {
	var declarations []contractDeclaration

	for _, declaration := range syntax.Decls {
		switch declaration := declaration.(type) {
		case *ast.FuncDecl:
			surfaces := []ast.Node{declaration.Type}
			if declaration.Recv != nil {
				surfaces = append(surfaces, declaration.Recv)
			}

			declarations = append(declarations, contractDeclaration{name: declaration.Name, surfaces: surfaces})
		case *ast.GenDecl:
			declarations = append(declarations, getGroupedContractDeclarations(declaration)...)
		}
	}

	return declarations
}

func getGroupedContractDeclarations(group *ast.GenDecl) []contractDeclaration {
	var declarations []contractDeclaration
	var previousValues []ast.Expr
	var previousType ast.Expr

	for _, spec := range group.Specs {
		switch spec := spec.(type) {
		case *ast.TypeSpec:
			declarations = append(declarations, contractDeclaration{
				name: spec.Name, surfaces: []ast.Node{spec},
				isCandidate: spec.Name.IsExported() && spec.Assign.IsValid(),
			})
		case *ast.ValueSpec:
			values, declaredType := spec.Values, spec.Type
			if group.Tok == token.CONST && len(values) == 0 {
				values, declaredType = previousValues, previousType
			}
			previousValues, previousType = values, declaredType

			for index, name := range spec.Names {
				var surfaces []ast.Node
				if declaredType != nil {
					surfaces = append(surfaces, declaredType)
				}

				if len(values) == len(spec.Names) {
					surfaces = append(surfaces, values[index])
				} else {
					for _, value := range values {
						surfaces = append(surfaces, value)
					}
				}

				declarations = append(declarations, contractDeclaration{
					name: name, surfaces: surfaces,
					isCandidate: group.Tok == token.CONST && name.IsExported(),
				})
			}
		}
	}

	return declarations
}

func visitContractSurface(node ast.Node, visit func(*ast.Ident)) {
	ast.Inspect(node, func(node ast.Node) bool {
		if function, isFunction := node.(*ast.FuncLit); isFunction {
			visitContractSurface(function.Type, visit)

			return false
		}

		if identifier, isIdentifier := node.(*ast.Ident); isIdentifier {
			visit(identifier)
		}

		return true
	})
}
