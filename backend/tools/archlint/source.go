package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type sourceFile struct {
	// relativePath is slash-separated and relative to the module root.
	relativePath string
	syntax       *ast.File
	isTest       bool
}

func (f sourceFile) getDirectory() string {
	return path.Dir(f.relativePath)
}

type sourceModule struct {
	root       string
	modulePath string
	fileSet    *token.FileSet
	files      []sourceFile
}

func loadModule(root string) (*sourceModule, error) {
	modulePath, err := readModulePath(filepath.Join(root, "go.mod"))
	if err != nil {
		return nil, err
	}

	loaded := &sourceModule{root: root, modulePath: modulePath, fileSet: token.NewFileSet(), files: nil}

	err = filepath.WalkDir(root, func(filePath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if entry.IsDir() {
			return skipDirectory(root, filePath, entry.Name())
		}

		if !strings.HasSuffix(entry.Name(), ".go") {
			return nil
		}

		syntax, parseErr := parser.ParseFile(loaded.fileSet, filePath, nil, parser.ParseComments)
		if parseErr != nil {
			return fmt.Errorf("parse %s: %w", filePath, parseErr)
		}

		relativePath, relErr := filepath.Rel(root, filePath)
		if relErr != nil {
			return fmt.Errorf("relative path of %s: %w", filePath, relErr)
		}

		loaded.files = append(loaded.files, sourceFile{
			relativePath: filepath.ToSlash(relativePath),
			syntax:       syntax,
			isTest:       strings.HasSuffix(entry.Name(), "_test.go"),
		})

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("load %s: %w", root, err)
	}

	return loaded, nil
}

// skipDirectory leaves out what the go command leaves out of ./..., and nested modules.
func skipDirectory(root, directoryPath, name string) error {
	if directoryPath == root {
		return nil
	}

	if strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") || name == "testdata" || name == "vendor" {
		return filepath.SkipDir
	}

	if _, err := os.Stat(filepath.Join(directoryPath, "go.mod")); err == nil {
		return filepath.SkipDir
	}

	return nil
}

func readModulePath(goModPath string) (string, error) {
	goMod, err := os.ReadFile(goModPath)
	if err != nil {
		return "", fmt.Errorf("read go.mod: %w", err)
	}

	for line := range strings.Lines(string(goMod)) {
		if modulePath, isModuleLine := strings.CutPrefix(strings.TrimSpace(line), "module "); isModuleLine {
			return strings.Trim(strings.TrimSpace(modulePath), `"`), nil
		}
	}

	return "", fmt.Errorf("%s: %w", goModPath, errModulePathMissing)
}

func (m *sourceModule) getPosition(file sourceFile, pos token.Pos) string {
	return fmt.Sprintf("%s:%d", file.relativePath, m.fileSet.Position(pos).Line)
}

// getRelativeImport returns the import path relative to the module root, or false for an import from
// outside the module.
func (m *sourceModule) getRelativeImport(importPath string) (string, bool) {
	return strings.CutPrefix(importPath, m.modulePath+"/")
}
