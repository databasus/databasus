package restore

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5"

	"databasus-backend/internal/agent/dbconn"
)

const bootstrapRole = "postgres"

var tocEntryPattern = regexp.MustCompile(`^\d+; \d+ \d+ \S+ .+ \S+$`)

// The throwaway target starts with only the bootstrap superuser, and pg_restore runs
// --no-owner --no-acl, so object ownership never needs these roles. Data does:
// TimescaleDB's _timescaledb_config.bgw_job.owner is a regrole column, and COPY resolves
// each role name to an OID as it loads, so a policy owned by an application role fails the
// whole restore with `role "..." does not exist`. Coverage is the archive's TOC
// owners, which is every role the dump names as an owner — a regrole value pointing at a role
// that owns no archived object is out of reach and would still fail.
// The ensured role names are returned so the caller can log them under its job scope.
func (r *Restorer) EnsureArchiveOwnerRoles(
	ctx context.Context,
	exec ExecRunner,
	archivePath string,
	conn dbconn.Conn,
) ([]string, error) {
	roleNames, err := getArchiveOwnerRoleNames(ctx, exec, archivePath)
	if err != nil {
		return nil, err
	}

	if len(roleNames) == 0 {
		return nil, nil
	}

	if err := createMissingOwnerRoles(ctx, conn, roleNames); err != nil {
		return nil, err
	}

	return roleNames, nil
}

func getArchiveOwnerRoleNames(
	ctx context.Context,
	exec ExecRunner,
	archivePath string,
) ([]string, error) {
	result, err := exec.Exec(ctx, []string{"pg_restore", "-l", archivePath}, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("pg_restore -l exec: %w", err)
	}

	// An unreadable archive is a verdict on the backup, not on this step, and the restore that
	// follows delivers it with the exit code the backend needs to mark the backup rejected.
	// Failing here instead would report no exit code, which the backend reads as agent
	// infrastructure trouble and retries forever.
	if result.ExitCode != 0 {
		return nil, nil
	}

	return parseTocOwnerRoleNames(result.Stdout), nil
}

// pg_restore -l prints one entry per line as "<id>; <tableoid> <oid> <desc> <schema> <tag>
// <owner>". Descriptions ("TABLE DATA") and tags ("add(integer, integer)") may contain
// spaces, but the owner is always last, so the trailing field is the only reliable anchor.
// An ownerless entry prints an empty trailing field, leaving the line ending in a space —
// hence the deliberate check before any trimming, without which the object tag would be
// mistaken for a role name.
func parseTocOwnerRoleNames(tocListing string) []string {
	seen := map[string]struct{}{}

	var roleNames []string

	for line := range strings.SplitSeq(tocListing, "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" || strings.HasPrefix(line, ";") || strings.HasSuffix(line, " ") {
			continue
		}

		if !tocEntryPattern.MatchString(line) {
			continue
		}

		fields := strings.Fields(line)

		roleName := fields[len(fields)-1]
		if roleName == bootstrapRole || strings.HasPrefix(roleName, "pg_") {
			continue
		}

		if _, isDup := seen[roleName]; isDup {
			continue
		}

		seen[roleName] = struct{}{}
		roleNames = append(roleNames, roleName)
	}

	return roleNames
}

// A role the target already carries (the bootstrap superuser aside, an image may ship its own)
// is not a conflict here, so each CREATE swallows duplicate_object rather than aborting.
func createMissingOwnerRoles(ctx context.Context, conn dbconn.Conn, roleNames []string) error {
	statements := make([]string, 0, len(roleNames))

	for _, roleName := range roleNames {
		statements = append(statements, fmt.Sprintf(
			"DO $$ BEGIN CREATE ROLE %s NOLOGIN; EXCEPTION WHEN duplicate_object THEN NULL; END $$",
			pgx.Identifier{roleName}.Sanitize(),
		))
	}

	return execStatements(ctx, conn, statements...)
}
