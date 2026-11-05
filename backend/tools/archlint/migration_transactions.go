package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var noTransactionPattern = regexp.MustCompile(`(?i)^\s*--\s*\+goose\s+NO\s+TRANSACTION\b`)

// checkMigrationTransactions fails a migration that opts out of its transaction: without one, a
// migration that fails halfway leaves its objects behind, and the next start trips over them.
func checkMigrationTransactions(root, migrationsDirectory string) ([]violation, error) {
	entries, err := os.ReadDir(migrationsDirectory)
	if os.IsNotExist(err) {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("read migrations: %w", err)
	}

	var violations []violation

	for _, entry := range entries {
		if entry.IsDir() || !migrationVersionPattern.MatchString(entry.Name()) {
			continue
		}

		migrationPath := filepath.Join(migrationsDirectory, entry.Name())

		content, err := os.ReadFile(migrationPath)
		if err != nil {
			return nil, fmt.Errorf("read migration: %w", err)
		}

		relativePath, err := filepath.Rel(root, migrationPath)
		if err != nil {
			return nil, fmt.Errorf("relative path of %s: %w", migrationPath, err)
		}

		for lineIndex, line := range strings.Split(string(content), "\n") {
			if !noTransactionPattern.MatchString(line) {
				continue
			}

			violations = append(violations, violation{
				position: fmt.Sprintf("%s:%d", filepath.ToSlash(relativePath), lineIndex+1),
				rule:     "migration-transaction",
				message:  "a migration runs in one transaction; remove -- +goose NO TRANSACTION",
			})
		}
	}

	return violations, nil
}
