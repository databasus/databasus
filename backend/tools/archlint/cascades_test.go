package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCascades(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		migrationName  string
		statement      string
		isViolation    bool
		messageSnippet string
	}{
		{
			name:          "fails when a new migration cascades between two modules",
			migrationName: "20261101000000_add_backups_logical.sql",
			statement: "ALTER TABLE backups_logical\n    ADD CONSTRAINT fk_backups_logical_storage\n" +
				"    FOREIGN KEY (storage_id)\n    REFERENCES storages (id)\n    ON DELETE CASCADE;\n",
			isViolation:    true,
			messageSnippet: "fk_backups_logical_storage cascades deletes from backups_logical (owner backups) to storages",
		},
		{
			name:          "fails when a new migration cascades from a backups table to a shared library table",
			migrationName: "20261101000000_add_backups_audit.sql",
			statement: "CREATE TABLE backups_audit (\n    event_id UUID NOT NULL REFERENCES lib_audit_events (id) " +
				"ON DELETE CASCADE\n);\n",
			isViolation:    true,
			messageSnippet: "to lib_audit_events (owner lib_audit)",
		},
		{
			name:          "passes when a new migration cascades between two tables of one lib owner",
			migrationName: "20261101000000_add_lib_audit_details.sql",
			statement: "CREATE TABLE lib_audit_details (\n    event_id UUID NOT NULL REFERENCES lib_audit_events (id) " +
				"ON DELETE CASCADE\n);\n",
		},
		{
			name:          "passes when a new migration cascades inside one module",
			migrationName: "20261101000000_add_storages_s3.sql",
			statement: "ALTER TABLE storages_s3\n    ADD CONSTRAINT fk_storages_s3_storage\n" +
				"    FOREIGN KEY (storage_id)\n    REFERENCES storages (id)\n    ON DELETE CASCADE;\n",
		},
		{
			name:          "passes when a new engine table without a module owner cascades to databases",
			migrationName: "20261101000000_add_oracle_databases.sql",
			statement: "CREATE TABLE oracle_databases (\n    database_id UUID REFERENCES databases(id) " +
				"ON DELETE CASCADE\n);\n",
		},
		{
			name:          "passes when a migration before the cutoff cascades between two modules",
			migrationName: "20250101000000_add_backups_logical.sql",
			statement: "ALTER TABLE backups_logical\n    ADD CONSTRAINT fk_backups_logical_storage\n" +
				"    FOREIGN KEY (storage_id)\n    REFERENCES storages (id)\n    ON DELETE CASCADE;\n",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			migration := "-- +goose Up\n-- +goose StatementBegin\n\n" + testCase.statement +
				"\n-- +goose StatementEnd\n\n-- +goose Down\nALTER TABLE storages_s3 ADD CONSTRAINT down_only " +
				"FOREIGN KEY (storage_id) REFERENCES backups (id) ON DELETE CASCADE;\n"
			violations := checkFixture(t, map[string]string{"migrations/" + testCase.migrationName: migration})

			if !testCase.isViolation {
				assert.Empty(t, violations)

				return
			}

			requireSingleViolation(t, violations, violationSpec{
				position:     "migrations/" + testCase.migrationName + ":5",
				rule:         "cascade",
				messageParts: []string{testCase.messageSnippet, "ON DELETE RESTRICT"},
			})
		})
	}
}
