package mysqlfamily

import (
	"regexp"
	"strings"
)

type Grant struct {
	Privileges    []string
	Scope         GrantScope
	SchemaPattern string
	TableName     string
}

var (
	grantHeadPattern  = regexp.MustCompile(`(?is)^\s*GRANT\s+(.+?)\s+ON\s+(.+?)\s+TO\s`)
	revokeHeadPattern = regexp.MustCompile(`(?is)^\s*REVOKE\s+(.+?)\s+ON\s+(.+?)\s+FROM\s`)
)

// A naive substring search over the whole grant line falsely matches privilege keywords that
// appear inside other privilege names, so the list has to be split before it is read: "SHOW
// CREATE ROUTINE" would otherwise register as "CREATE".
func ParseGrantLine(grantLine string) *Grant {
	return parseGrantHead(grantHeadPattern, grantLine)
}

// SHOW GRANTS reports a partial revoke (partial_revokes=ON, the managed-MySQL default shape) as
// a REVOKE line; ignoring it would count a globally granted privilege the server refuses on
// this schema.
func parseRevokeLine(grantLine string) *Grant {
	return parseGrantHead(revokeHeadPattern, grantLine)
}

func parseGrantHead(headPattern *regexp.Regexp, grantLine string) *Grant {
	matches := headPattern.FindStringSubmatch(grantLine)
	if matches == nil {
		return nil
	}

	privileges := parseGrantedPrivileges(matches[1])
	if len(privileges) == 0 {
		return nil
	}

	grant := parseGrantTarget(strings.TrimSpace(matches[2]))
	if grant == nil {
		return nil
	}

	grant.Privileges = privileges

	return grant
}

func parseGrantedPrivileges(privilegeList string) []string {
	parts := splitOutsideParentheses(privilegeList)

	privileges := make([]string, 0, len(parts))

	for _, part := range parts {
		// A column-qualified privilege reaches only the listed columns, so counting it as
		// table-wide would pass an account whose dump later fails on the unlisted ones.
		if strings.Contains(part, "(") {
			continue
		}

		privilege := strings.TrimSpace(strings.ToUpper(part))
		if privilege != "" {
			privileges = append(privileges, privilege)
		}
	}

	return privileges
}

// A column list carries its own commas, so the privilege separator is only the comma that
// sits outside parentheses: "SELECT (col1, col2), UPDATE (col3)" is two privileges, not four.
func splitOutsideParentheses(privilegeList string) []string {
	var parts []string

	parenthesisDepth := 0
	partStart := 0

	for index, character := range privilegeList {
		switch character {
		case '(':
			parenthesisDepth++
		case ')':
			if parenthesisDepth > 0 {
				parenthesisDepth--
			}
		case ',':
			if parenthesisDepth == 0 {
				parts = append(parts, privilegeList[partStart:index])
				partStart = index + 1
			}
		}
	}

	return append(parts, privilegeList[partStart:])
}

// A PROXY grant names an account rather than a schema, and its target is quoted with single
// quotes, which is how it is told apart from the schema forms.
func parseGrantTarget(target string) *Grant {
	if strings.HasPrefix(target, "'") {
		return nil
	}

	schemaPart, tablePart, hasTablePart := cutOnScopeSeparator(target)
	if !hasTablePart {
		return nil
	}

	schemaPattern := unquoteGrantIdentifier(schemaPart)
	tableName := unquoteGrantIdentifier(tablePart)

	if schemaPattern == "*" && tableName == "*" {
		return &Grant{Scope: GrantScopeGlobal}
	}

	if tableName == "*" {
		return &Grant{Scope: GrantScopeSchema, SchemaPattern: schemaPattern}
	}

	return &Grant{
		Scope:         GrantScopeTable,
		SchemaPattern: schemaPattern,
		TableName:     tableName,
	}
}

// A backquoted schema name may itself contain a dot, so the split has to happen on the separator
// that sits outside the quotes rather than on the first dot in the string.
func cutOnScopeSeparator(target string) (string, string, bool) {
	isInsideQuotes := false

	for index, character := range target {
		switch character {
		case '`':
			isInsideQuotes = !isInsideQuotes
		case '.':
			if !isInsideQuotes {
				return target[:index], target[index+1:], true
			}
		}
	}

	return target, "", false
}

func unquoteGrantIdentifier(identifier string) string {
	return strings.Trim(strings.TrimSpace(identifier), "`\"")
}

// A schema grant is written as a LIKE pattern, so "app%" and "test\_db" have to be matched the way
// the server matches them rather than compared literally.
func isMatchingSchemaPattern(schemaPattern, schemaName string) bool {
	var translatedPattern strings.Builder
	translatedPattern.WriteString("^")

	for index := 0; index < len(schemaPattern); index++ {
		character := schemaPattern[index]

		if character == '\\' && index+1 < len(schemaPattern) {
			index++
			translatedPattern.WriteString(regexp.QuoteMeta(string(schemaPattern[index])))

			continue
		}

		switch character {
		case '%':
			translatedPattern.WriteString(".*")
		case '_':
			translatedPattern.WriteString(".")
		default:
			translatedPattern.WriteString(regexp.QuoteMeta(string(character)))
		}
	}

	translatedPattern.WriteString("$")

	compiledPattern, err := regexp.Compile(translatedPattern.String())
	if err != nil {
		return schemaPattern == schemaName
	}

	return compiledPattern.MatchString(schemaName)
}

type SchemaGrants struct {
	Global  map[string]bool
	Schema  map[string]bool
	Tables  map[string]map[string]bool
	Revoked map[string]bool
}

// ALL PRIVILEGES has to be expanded into the privileges this package consults, and a grant on a
// single table cannot carry the ones that only exist schema-wide.
var (
	schemaWideGrantablePrivileges = []string{"SELECT", "SHOW VIEW", "TRIGGER", "EVENT"}
	tableGrantablePrivileges      = []string{"SELECT", "SHOW VIEW", "TRIGGER"}
)

func CollectSchemaGrants(grantLines []string, schemaName string) SchemaGrants {
	collectedGrants := SchemaGrants{
		Global:  map[string]bool{},
		Schema:  map[string]bool{},
		Tables:  map[string]map[string]bool{},
		Revoked: map[string]bool{},
	}

	for _, grantLine := range grantLines {
		if revokedGrant := parseRevokeLine(grantLine); revokedGrant != nil {
			if revokedGrant.Scope == GrantScopeSchema &&
				isMatchingSchemaPattern(revokedGrant.SchemaPattern, schemaName) {
				addGrantedPrivileges(
					collectedGrants.Revoked,
					revokedGrant.Privileges,
					schemaWideGrantablePrivileges,
				)
			}

			continue
		}

		grant := ParseGrantLine(grantLine)
		if grant == nil {
			continue
		}

		switch grant.Scope {
		case GrantScopeGlobal:
			addGrantedPrivileges(collectedGrants.Global, grant.Privileges, schemaWideGrantablePrivileges)
		case GrantScopeSchema:
			if isMatchingSchemaPattern(grant.SchemaPattern, schemaName) {
				addGrantedPrivileges(
					collectedGrants.Schema,
					grant.Privileges,
					schemaWideGrantablePrivileges,
				)
			}
		case GrantScopeTable:
			if !isMatchingSchemaPattern(grant.SchemaPattern, schemaName) {
				continue
			}

			if collectedGrants.Tables[grant.TableName] == nil {
				collectedGrants.Tables[grant.TableName] = map[string]bool{}
			}

			addGrantedPrivileges(
				collectedGrants.Tables[grant.TableName],
				grant.Privileges,
				tableGrantablePrivileges,
			)
		}
	}

	return collectedGrants
}

func addGrantedPrivileges(
	grantedPrivileges map[string]bool,
	privileges []string,
	grantableAtThisScope []string,
) {
	for _, privilege := range privileges {
		if privilege == "ALL PRIVILEGES" || privilege == "ALL" {
			for _, grantablePrivilege := range grantableAtThisScope {
				grantedPrivileges[grantablePrivilege] = true
			}

			continue
		}

		grantedPrivileges[privilege] = true
	}
}

// A partial revoke subtracts only from the global level: the server lifts it again the moment
// the same privilege is granted on the schema explicitly.
func (g SchemaGrants) HasForSchema(privilege string) bool {
	if g.Schema[privilege] {
		return true
	}

	return g.Global[privilege] && !g.Revoked[privilege]
}

func (g SchemaGrants) HasForTable(privilege, tableName string) bool {
	if g.HasForSchema(privilege) {
		return true
	}

	return g.Tables[tableName][privilege]
}
