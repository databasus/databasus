package main

import (
	"fmt"
	"go/token"
	"slices"
	"strings"

	"golang.org/x/tools/go/packages"
)

func loadContractPackages(root string) ([]*packages.Package, *token.FileSet, error) {
	fileSet := token.NewFileSet()
	loaded, err := packages.Load(&packages.Config{
		Dir: root, Fset: fileSet, Mode: packages.LoadSyntax, Tests: true, BuildFlags: []string{"-tags="},
	}, "./...")
	if err != nil {
		return nil, nil, fmt.Errorf("load contract consumers: %w", err)
	}

	var messages []string
	packages.Visit(loaded, nil, func(loadedPackage *packages.Package) {
		for _, packageError := range loadedPackage.Errors {
			messages = append(messages, packageError.Error())
		}
	})
	if len(messages) != 0 {
		slices.Sort(messages)

		return nil, nil, fmt.Errorf("load contract consumers: %w: %s",
			errPackagesHaveErrors, strings.Join(slices.Compact(messages), "\n"))
	}

	return loaded, fileSet, nil
}
