package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"path"
	"path/filepath"

	"golang.org/x/tools/go/packages"
)

// Test variants have separate type objects for the same source declaration.
type contractIdentity struct {
	filePath string
	offset   int
}

type contractReference struct {
	name                string
	moduleName          string
	position            string
	isCandidate         bool
	hasExternalConsumer bool
	dependencies        map[contractIdentity]struct{}
}

type contractReferences struct {
	root         string
	fileSet      *token.FileSet
	declarations map[contractIdentity]*contractReference
	bindings     map[contractIdentity]contractIdentity
}

func newContractReferences(root string, fileSet *token.FileSet) *contractReferences {
	return &contractReferences{
		root: root, fileSet: fileSet,
		declarations: make(map[contractIdentity]*contractReference),
		bindings:     make(map[contractIdentity]contractIdentity),
	}
}

func (r *contractReferences) getIdentity(pos token.Pos) contractIdentity {
	position := r.fileSet.PositionFor(pos, false)

	return contractIdentity{filePath: position.Filename, offset: position.Offset}
}

func (r *contractReferences) getRelativePath(pos token.Pos) string {
	filePath := r.fileSet.PositionFor(pos, false).Filename
	relativePath, err := filepath.Rel(r.root, filePath)
	if err != nil {
		return filepath.ToSlash(filePath)
	}

	return filepath.ToSlash(relativePath)
}

func (r *contractReferences) addDeclarations(loaded *packages.Package) {
	for _, syntax := range loaded.Syntax {
		relativePath := r.getRelativePath(syntax.Pos())
		moduleName, isModule := getModuleName(relativePath)
		if !isModule || relativePath != path.Join("internal/modules", moduleName, "module.go") {
			continue
		}

		for _, declaration := range getContractDeclarations(syntax) {
			identity := r.getIdentity(declaration.name.Pos())
			if _, hasDeclaration := r.declarations[identity]; !hasDeclaration {
				r.declarations[identity] = &contractReference{
					name:       declaration.name.Name,
					moduleName: moduleName,
					position: fmt.Sprintf(
						"%s:%d",
						relativePath,
						r.fileSet.PositionFor(declaration.name.Pos(), false).Line,
					),
					isCandidate:  declaration.isCandidate,
					dependencies: make(map[contractIdentity]struct{}),
				}
			}

			r.bindings[identity] = identity
			for _, surface := range declaration.surfaces {
				visitContractSurface(surface, func(identifier *ast.Ident) {
					if object := loaded.TypesInfo.Defs[identifier]; object != nil {
						r.bindings[r.getIdentity(object.Pos())] = identity
					}
				})
			}
		}
	}
}

func (r *contractReferences) addReferences(loaded *packages.Package) {
	for identifier, object := range loaded.TypesInfo.Uses {
		identity, isContract := r.bindings[r.getIdentity(object.Pos())]
		if !isContract {
			continue
		}

		declaration := r.declarations[identity]
		if !hasPathPrefix(r.getRelativePath(identifier.Pos()), path.Join("internal/modules", declaration.moduleName)) {
			declaration.hasExternalConsumer = true
		}
	}

	for _, syntax := range loaded.Syntax {
		for _, declaration := range getContractDeclarations(syntax) {
			identity := r.getIdentity(declaration.name.Pos())
			reference, isContract := r.declarations[identity]
			if !isContract {
				continue
			}

			for _, surface := range declaration.surfaces {
				visitContractSurface(surface, func(identifier *ast.Ident) {
					r.addDependency(reference, loaded.TypesInfo.Uses[identifier])
				})
			}
		}
	}
}

func (r *contractReferences) addDependency(declaration *contractReference, object types.Object) {
	if object == nil {
		return
	}

	identity, isContract := r.bindings[r.getIdentity(object.Pos())]
	if isContract && r.declarations[identity].moduleName == declaration.moduleName {
		declaration.dependencies[identity] = struct{}{}
	}
}

func (r *contractReferences) getConsumedDeclarations() map[contractIdentity]bool {
	consumed := make(map[contractIdentity]bool)
	var pending []contractIdentity

	for identity, declaration := range r.declarations {
		if declaration.hasExternalConsumer {
			pending = append(pending, identity)
		}
	}

	for len(pending) > 0 {
		identity := pending[len(pending)-1]
		pending = pending[:len(pending)-1]

		if consumed[identity] {
			continue
		}

		consumed[identity] = true
		for dependency := range r.declarations[identity].dependencies {
			pending = append(pending, dependency)
		}
	}

	return consumed
}
