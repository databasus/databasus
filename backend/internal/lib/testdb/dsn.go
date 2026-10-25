package testdb

import (
	"strings"
)

// GetDatabaseName returns the dbname of a keyword/value connection string, or "" when it has none.
func GetDatabaseName(dsn string) string {
	for dsnPart := range strings.FieldsSeq(dsn) {
		if databaseName, isDatabaseName := strings.CutPrefix(dsnPart, "dbname="); isDatabaseName {
			return databaseName
		}
	}

	return ""
}

func replaceDatabaseName(dsn, databaseName string) string {
	dsnParts := strings.Fields(dsn)
	for partIndex, dsnPart := range dsnParts {
		if strings.HasPrefix(dsnPart, "dbname=") {
			dsnParts[partIndex] = "dbname=" + databaseName
		}
	}

	return strings.Join(dsnParts, " ")
}

func removePassword(dsn string) string {
	dsnParts := strings.Fields(dsn)
	keptParts := make([]string, 0, len(dsnParts))

	for _, dsnPart := range dsnParts {
		if !strings.HasPrefix(dsnPart, "password=") {
			keptParts = append(keptParts, dsnPart)
		}
	}

	return strings.Join(keptParts, " ")
}
