package mysqlfamily

import (
	"fmt"
	"slices"
	"strings"

	"databasus-backend/internal/lib/namelist"
)

const viewTableType = "VIEW"

// A view is dumped by its definition, which is what SHOW VIEW unlocks; a base table is dumped by
// its rows. Requiring SHOW VIEW on a schema that holds no views would reject an account whose dump
// would have succeeded.
type DumpedTable struct {
	Name   string
	IsView bool
}

func (t DumpedTable) getRequiredPrivileges() []string {
	if t.IsView {
		return []string{"SELECT", "SHOW VIEW"}
	}

	return []string{"SELECT"}
}

// The dump tool aborts on an object it can list but cannot read, so these decide whether a dump can
// run at all. Which of them a given object actually needs is DumpedTable's business.
func getRequiredBackupPrivileges() []string {
	return []string{"SELECT", "SHOW VIEW"}
}

// TRIGGER and EVENT are not required: without them the dump tool is told to leave those objects
// out rather than fail, and the stored set is what the backup consults to decide that.
func getRecordedBackupPrivileges() []string {
	return []string{"SELECT", "SHOW VIEW", "TRIGGER", "EVENT"}
}

type BackupPrivilegesSpec struct {
	SchemaName   string
	Grantee      string
	DumpedTables []DumpedTable
	Grants       SchemaGrants
}

type MissingBackupPrivilege struct {
	Privilege string
	Scope     GrantScope
	Tables    []string
}

type BackupPrivileges struct {
	EffectivePrivileges []string
	Missing             []MissingBackupPrivilege
	SchemaName          string
	Grantee             string
}

func GetBackupPrivileges(spec BackupPrivilegesSpec) BackupPrivileges {
	backupPrivileges := BackupPrivileges{
		SchemaName: spec.SchemaName,
		Grantee:    spec.Grantee,
	}

	for _, privilege := range getRecordedBackupPrivileges() {
		if isPrivilegeCoveringDump(spec, privilege) {
			backupPrivileges.EffectivePrivileges = append(backupPrivileges.EffectivePrivileges, privilege)
		}
	}

	slices.Sort(backupPrivileges.EffectivePrivileges)

	for _, privilege := range getRequiredBackupPrivileges() {
		if missingPrivilege, isMissing := getMissingPrivilege(spec, privilege); isMissing {
			backupPrivileges.Missing = append(backupPrivileges.Missing, missingPrivilege)
		}
	}

	return backupPrivileges
}

func isPrivilegeCoveringDump(spec BackupPrivilegesSpec, privilege string) bool {
	if len(spec.DumpedTables) == 0 {
		return spec.Grants.HasForSchema(privilege)
	}

	for _, dumpedTable := range spec.DumpedTables {
		if !spec.Grants.HasForTable(privilege, dumpedTable.Name) {
			return false
		}
	}

	return true
}

// A privilege absent everywhere is reported once against the schema; listing every table would
// bury the fix under noise when the role simply holds nothing.
func getMissingPrivilege(
	spec BackupPrivilegesSpec,
	privilege string,
) (MissingBackupPrivilege, bool) {
	if len(spec.DumpedTables) == 0 {
		if spec.Grants.HasForSchema(privilege) {
			return MissingBackupPrivilege{}, false
		}

		return MissingBackupPrivilege{Privilege: privilege, Scope: GrantScopeSchema}, true
	}

	var tablesMissingPrivilege []string
	requiringTableCount := 0

	for _, dumpedTable := range spec.DumpedTables {
		if !slices.Contains(dumpedTable.getRequiredPrivileges(), privilege) {
			continue
		}

		requiringTableCount++

		if !spec.Grants.HasForTable(privilege, dumpedTable.Name) {
			tablesMissingPrivilege = append(tablesMissingPrivilege, dumpedTable.Name)
		}
	}

	if len(tablesMissingPrivilege) == 0 {
		return MissingBackupPrivilege{}, false
	}

	if len(tablesMissingPrivilege) == requiringTableCount {
		return MissingBackupPrivilege{Privilege: privilege, Scope: GrantScopeSchema}, true
	}

	return MissingBackupPrivilege{
		Privilege: privilege,
		Scope:     GrantScopeTable,
		Tables:    tablesMissingPrivilege,
	}, true
}

func (p BackupPrivileges) IsSufficientForDump() bool {
	return len(p.Missing) == 0
}

func (p BackupPrivileges) GetEffectivePrivilegesCsv() string {
	return strings.Join(p.EffectivePrivileges, ",")
}

func (p BackupPrivileges) NewInsufficiencyError() error {
	missingDescriptions := make([]string, 0, len(p.Missing))
	missingPrivilegeNames := make([]string, 0, len(p.Missing))

	for _, missingPrivilege := range p.Missing {
		missingPrivilegeNames = append(missingPrivilegeNames, missingPrivilege.Privilege)

		if missingPrivilege.Scope == GrantScopeTable {
			missingDescriptions = append(missingDescriptions, fmt.Sprintf(
				"%s on %s",
				missingPrivilege.Privilege,
				namelist.FormatTruncatedNames(qualifyTableNames(p.SchemaName, missingPrivilege.Tables)),
			))

			continue
		}

		missingDescriptions = append(missingDescriptions, fmt.Sprintf(
			"%s on `%s`.*",
			missingPrivilege.Privilege,
			p.SchemaName,
		))
	}

	// The suggested grant is always schema-wide: it repairs every listed table at once and keeps
	// covering the tables the schema gains later.
	return fmt.Errorf(
		"insufficient permissions for backup. Missing: %s. Run: GRANT %s ON `%s`.* TO %s;",
		strings.Join(missingDescriptions, ", "),
		strings.Join(missingPrivilegeNames, ", "),
		p.SchemaName,
		p.Grantee,
	)
}

func qualifyTableNames(schemaName string, tableNames []string) []string {
	qualifiedNames := make([]string, 0, len(tableNames))

	for _, tableName := range tableNames {
		qualifiedNames = append(qualifiedNames, fmt.Sprintf("`%s`.`%s`", schemaName, tableName))
	}

	return qualifiedNames
}
