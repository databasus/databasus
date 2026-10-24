package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// moduleNames lists the domain modules of the module table in docs/architecture.md. A table belongs
// to a module when its name equals the module name or starts with it and an underscore.
var moduleNames = []string{
	"users",
	"workspaces",
	"notifiers",
	"storages",
	"databases",
	"backups",
	"verification",
	"healthcheck",
	"telemetry",
}

var (
	migrationVersionPattern = regexp.MustCompile(`^(\d+)_.*\.sql$`)
	sqlCommentPattern       = regexp.MustCompile(`--[^\n]*`)
	createTablePattern      = regexp.MustCompile(
		`(?is)^\s*CREATE\s+(?:UNLOGGED\s+)?TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?([\w."]+)`,
	)
	alterTablePattern = regexp.MustCompile(`(?is)^\s*ALTER\s+TABLE\s+(?:IF\s+EXISTS\s+)?(?:ONLY\s+)?([\w."]+)`)
	foreignKeyPattern = regexp.MustCompile(
		`(?is)(?:CONSTRAINT\s+([\w"]+)\s+)?(?:FOREIGN\s+KEY\s*\([^)]*\)\s*)?REFERENCES\s+([\w."]+)\s*(?:\([^)]*\))?` +
			`((?:\s+(?:ON\s+(?:DELETE|UPDATE)\s+(?:CASCADE|RESTRICT|NO\s+ACTION|SET\s+NULL|SET\s+DEFAULT)` +
			`|MATCH\s+\w+|NOT\s+DEFERRABLE|DEFERRABLE|INITIALLY\s+\w+))*)`,
	)
	onDeleteCascadePattern = regexp.MustCompile(`(?is)ON\s+DELETE\s+CASCADE`)
)

// checkCascades fails a migration newer than the cutoff that cascades a delete from a table of one
// owner to a table of another. A key that involves a table without an owner is skipped: tables older
// than the module prefixes do not follow the scheme yet.
func checkCascades(root, migrationsDirectory string, cutoff int64) ([]violation, error) {
	entries, err := os.ReadDir(migrationsDirectory)
	if os.IsNotExist(err) {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("read migrations: %w", err)
	}

	var violations []violation

	for _, entry := range entries {
		match := migrationVersionPattern.FindStringSubmatch(entry.Name())
		if entry.IsDir() || match == nil {
			continue
		}

		version, err := strconv.ParseInt(match[1], 10, 64)
		if err != nil || version <= cutoff {
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

		violations = append(violations, checkMigration(filepath.ToSlash(relativePath), string(content))...)
	}

	return violations, nil
}

func checkMigration(relativePath, content string) []violation {
	upSection := getUpSection(sqlCommentPattern.ReplaceAllStringFunc(content, func(comment string) string {
		if strings.Contains(comment, "+goose Down") {
			return comment
		}

		return strings.Repeat(" ", len(comment))
	}))

	var violations []violation

	statementStart := 0

	for _, statement := range strings.SplitAfter(upSection, ";") {
		table := getStatementTable(statement)

		for _, foreignKey := range foreignKeyPattern.FindAllStringSubmatchIndex(statement, -1) {
			referencedTable := normalizeTableName(statement[foreignKey[4]:foreignKey[5]])
			actions := statement[foreignKey[6]:foreignKey[7]]

			if table == "" || !onDeleteCascadePattern.MatchString(actions) {
				continue
			}

			tableOwner, hasTableOwner := getTableOwner(table)
			referencedOwner, hasReferencedOwner := getTableOwner(referencedTable)

			if !hasTableOwner || !hasReferencedOwner || tableOwner == referencedOwner {
				continue
			}

			constraintName := table + " → " + referencedTable
			if foreignKey[2] >= 0 {
				constraintName = strings.Trim(statement[foreignKey[2]:foreignKey[3]], `"`)
			}

			line := 1 + strings.Count(upSection[:statementStart+foreignKey[0]], "\n")
			violations = append(violations, violation{
				position: fmt.Sprintf("%s:%d", relativePath, line),
				rule:     "cascade",
				message: fmt.Sprintf(
					"constraint %s cascades deletes from %s (owner %s) to %s (owner %s); "+
						"a key to another owner's table is ON DELETE RESTRICT",
					constraintName, table, tableOwner, referencedTable, referencedOwner,
				),
			})
		}

		statementStart += len(statement)
	}

	return violations
}

// getUpSection keeps everything before "+goose Down", so line numbers still match the file.
func getUpSection(content string) string {
	upSection, _, _ := strings.Cut(content, "-- +goose Down")

	return upSection
}

func getStatementTable(statement string) string {
	for _, pattern := range []*regexp.Regexp{createTablePattern, alterTablePattern} {
		if match := pattern.FindStringSubmatch(statement); match != nil {
			return normalizeTableName(match[1])
		}
	}

	return ""
}

func normalizeTableName(name string) string {
	name = strings.ToLower(strings.ReplaceAll(name, `"`, ""))
	name = strings.TrimPrefix(name, "public.")

	return name
}

// getTableOwner returns the module that owns a table, or lib_<name> for a shared library table.
func getTableOwner(table string) (string, bool) {
	if rest, isLibraryTable := strings.CutPrefix(table, "lib_"); isLibraryTable && rest != "" {
		name, _, _ := strings.Cut(rest, "_")

		return "lib_" + name, true
	}

	for _, moduleName := range moduleNames {
		if table == moduleName || strings.HasPrefix(table, moduleName+"_") {
			return moduleName, true
		}
	}

	return "", false
}
