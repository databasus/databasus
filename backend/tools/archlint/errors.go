package main

import "errors"

var (
	errModulePathMissing  = errors.New("go.mod declares no module path")
	errPackagesHaveErrors = errors.New("packages have errors")
)
