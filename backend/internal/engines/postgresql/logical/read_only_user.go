package logical

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/jackc/pgx/v5"

	"databasus-backend/internal/lib/crypto"
)

const maxReadOnlyUserAttempts = 3

//nolint:gochecknoglobals // Shared immutable privilege lookup is reused across table grants; it is never mutated.
var tableWritePrivileges = map[string]bool{
	"INSERT":     true,
	"UPDATE":     true,
	"DELETE":     true,
	"TRUNCATE":   true,
	"REFERENCES": true,
	"TRIGGER":    true,
}

// ShouldSuggestReadOnlyUser lists what the connecting role may do beyond reading, and suggests a
// read-only user when it may do anything more and no table in scope needs BYPASSRLS, which a
// generated role cannot be given.
func (s *Source) ShouldSuggestReadOnlyUser(
	ctx context.Context,
	logger *slog.Logger,
	includeSchemas []string,
) (bool, []string, error) {
	conn, err := s.connect(ctx, s.settings.Database)
	if err != nil {
		return false, nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	defer closeConn(ctx, conn, logger)

	var privileges []string

	for _, listPrivileges := range []func(context.Context, *pgx.Conn) ([]string, error){
		listRoleAttributes,
		listDatabasePrivileges,
		listSchemaCreatePrivilege,
		listTableWritePrivileges,
		listSecurityDefinerExecutePrivilege,
	} {
		listedPrivileges, err := listPrivileges(ctx, conn)
		if err != nil {
			return false, nil, err
		}

		privileges = append(privileges, listedPrivileges...)
	}

	rowLevelSecurityTables, err := getRowLevelSecurityTables(ctx, conn, includeSchemas)
	if err != nil {
		return false, nil, err
	}

	return len(privileges) > 0 && len(rowLevelSecurityTables) == 0, privileges, nil
}

// CreateReadOnlyUser creates a role that can read every schema in scope, now and in the future,
// and returns the user name to connect with and its password. A generated name that is already
// taken is retried with a new one; any other failure ends the creation.
func (s *Source) CreateReadOnlyUser(
	ctx context.Context,
	logger *slog.Logger,
	includeSchemas []string,
) (string, string, error) {
	conn, err := s.connect(ctx, s.settings.Database)
	if err != nil {
		return "", "", fmt.Errorf("failed to connect to database: %w", err)
	}
	defer closeConn(ctx, conn, logger)

	// These checks run on the bare connection, so a refusal costs no transaction and leaks no
	// generated credentials.
	if err := checkCanCreateRole(ctx, conn); err != nil {
		return "", "", err
	}

	if err := refuseRowLevelSecurityTables(ctx, conn, includeSchemas); err != nil {
		return "", "", err
	}

	for range maxReadOnlyUserAttempts {
		roleName := s.generateUsername()

		password, err := s.createReadOnlyRole(ctx, conn, logger, roleName, includeSchemas)
		if classifyServerError(err) == serverErrorNameTaken {
			continue
		}

		if err != nil {
			return "", "", err
		}

		connectionUsername := s.getConnectionUsername(roleName)
		logger.InfoContext(ctx, "read-only user created successfully",
			slog.String("username", roleName),
			slog.String("connection_username", connectionUsername),
		)

		return connectionUsername, password, nil
	}

	return "", "", fmt.Errorf("%w after %d attempts", ErrUserNameCollision, maxReadOnlyUserAttempts)
}

// createReadOnlyRole is one attempt in a transaction of its own: a failed CREATE ROLE aborts the
// transaction it ran in, so a retry needs a fresh one.
func (s *Source) createReadOnlyRole(
	ctx context.Context,
	conn *pgx.Conn,
	logger *slog.Logger,
	roleName string,
	includeSchemas []string,
) (string, error) {
	tx, err := conn.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to begin transaction: %w", err)
	}

	isCommitted := false

	defer func() {
		if isCommitted {
			return
		}

		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
			logger.ErrorContext(ctx, "failed to rollback transaction", slog.Any("error", rollbackErr))
		}
	}()

	password := crypto.GeneratePassword()

	if _, err := tx.Exec(
		ctx,
		fmt.Sprintf(`CREATE USER "%s" WITH PASSWORD '%s' LOGIN`, roleName, password),
	); err != nil {
		return "", fmt.Errorf("failed to create user: %w", err)
	}

	if err := revokeCreateOnPublicSchema(ctx, tx, logger, roleName); err != nil {
		return "", err
	}

	if err := grantDatabaseConnect(ctx, tx, logger, s.settings.Database, roleName); err != nil {
		return "", err
	}

	schemas, err := listSchemasInScope(ctx, tx, includeSchemas)
	if err != nil {
		return "", err
	}

	if err := grantSchemaUsage(ctx, tx, logger, schemas, roleName); err != nil {
		return "", err
	}

	if err := grantReadOnExistingObjects(ctx, tx, schemas, roleName); err != nil {
		return "", err
	}

	if err := grantReadOnFutureOwnObjects(ctx, tx, schemas, roleName); err != nil {
		return "", err
	}

	grantReadOnFutureObjectsOfOtherOwners(ctx, tx, logger, includeSchemas, roleName)

	if err := verifyRoleExists(ctx, tx, roleName); err != nil {
		return "", err
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("failed to commit transaction: %w", err)
	}

	isCommitted = true

	return password, nil
}

// For the Supabase session pooler the user connects as "<role>.<project>", while the role itself is
// created under the bare name; the pooler strips the suffix when it authenticates.
func (s *Source) getConnectionUsername(roleName string) string {
	isSupabase := strings.Contains(strings.ToLower(s.settings.Host), "supabase") ||
		strings.Contains(strings.ToLower(s.settings.Username), "supabase")
	if !isSupabase {
		return roleName
	}

	_, supabaseProjectID, hasProjectID := strings.Cut(s.settings.Username, ".")
	if !hasProjectID || supabaseProjectID == "" {
		return roleName
	}

	return roleName + "." + supabaseProjectID
}

func checkCanCreateRole(ctx context.Context, conn *pgx.Conn) error {
	var canCreateRole, isSuperuser bool

	err := conn.QueryRow(ctx, `
		SELECT rolcreaterole, rolsuper
		FROM pg_roles
		WHERE rolname = current_user
	`).Scan(&canCreateRole, &isSuperuser)
	if err != nil {
		return fmt.Errorf("failed to check permissions: %w", err)
	}

	if !canCreateRole && !isSuperuser {
		return ErrCreateRoleMissing
	}

	return nil
}

// Every role inherits CREATE on the public schema through PUBLIC, so revoking it from PUBLIC is
// what keeps the new role from writing there. It affects the whole database and only the public
// schema, and a role that may not change it only gets a warning.
func revokeCreateOnPublicSchema(ctx context.Context, tx pgx.Tx, logger *slog.Logger, roleName string) error {
	var isPublicSchemaPresent bool

	err := tx.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM information_schema.schemata
			WHERE schema_name = 'public'
		)
	`).Scan(&isPublicSchemaPresent)
	if err != nil {
		return fmt.Errorf("failed to check if public schema exists: %w", err)
	}

	if !isPublicSchemaPresent {
		logger.InfoContext(ctx, "public schema does not exist, skipping CREATE privilege revocation")

		return nil
	}

	if _, err := tx.Exec(ctx, `REVOKE CREATE ON SCHEMA public FROM PUBLIC`); err != nil {
		if classifyServerError(err) != serverErrorPermissionDenied {
			return fmt.Errorf("failed to revoke CREATE from PUBLIC on existing public schema: %w", err)
		}

		logger.WarnContext(ctx, "failed to revoke CREATE on public from PUBLIC (permission denied)",
			slog.Any("error", err))
	}

	if _, err := tx.Exec(ctx, fmt.Sprintf(`REVOKE CREATE ON SCHEMA public FROM "%s"`, roleName)); err != nil {
		logger.WarnContext(ctx, "failed to revoke CREATE on public schema from user",
			slog.Any("error", err), slog.String("username", roleName))
	}

	return nil
}

// TEMP, like CREATE on the public schema, reaches every role through PUBLIC.
func grantDatabaseConnect(
	ctx context.Context,
	tx pgx.Tx,
	logger *slog.Logger,
	databaseName string,
	roleName string,
) error {
	if _, err := tx.Exec(
		ctx,
		fmt.Sprintf(`GRANT CONNECT ON DATABASE "%s" TO "%s"`, databaseName, roleName),
	); err != nil {
		return fmt.Errorf("failed to grant connect privilege: %w", err)
	}

	if _, err := tx.Exec(ctx, fmt.Sprintf(`REVOKE TEMP ON DATABASE "%s" FROM PUBLIC`, databaseName)); err != nil {
		logger.WarnContext(ctx, "failed to revoke TEMP from PUBLIC", slog.Any("error", err))
	}

	if _, err := tx.Exec(
		ctx,
		fmt.Sprintf(`REVOKE TEMP ON DATABASE "%s" FROM "%s"`, databaseName, roleName),
	); err != nil {
		logger.WarnContext(ctx, "failed to revoke TEMP privilege",
			slog.Any("error", err), slog.String("username", roleName))
	}

	return nil
}

// listSchemasInScope returns the included schemas, or every schema outside the system catalogs
// when none is included.
func listSchemasInScope(ctx context.Context, tx pgx.Tx, includeSchemas []string) ([]string, error) {
	var (
		rows pgx.Rows
		err  error
	)

	if len(includeSchemas) > 0 {
		rows, err = tx.Query(ctx, `
			SELECT schema_name
			FROM information_schema.schemata
			WHERE schema_name NOT IN ('pg_catalog', 'information_schema')
			AND schema_name = ANY($1::text[])
		`, includeSchemas)
	} else {
		rows, err = tx.Query(ctx, `
			SELECT schema_name
			FROM information_schema.schemata
			WHERE schema_name NOT IN ('pg_catalog', 'information_schema')
		`)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get schemas: %w", err)
	}

	schemas, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, fmt.Errorf("failed to read schemas: %w", err)
	}

	return schemas, nil
}

// The explicit REVOKE CREATE covers what the role would otherwise inherit from PUBLIC.
func grantSchemaUsage(ctx context.Context, tx pgx.Tx, logger *slog.Logger, schemas []string, roleName string) error {
	for _, schema := range schemas {
		if _, err := tx.Exec(ctx, fmt.Sprintf(`REVOKE CREATE ON SCHEMA "%s" FROM "%s"`, schema, roleName)); err != nil {
			logger.WarnContext(ctx, "failed to revoke CREATE on schema",
				slog.Any("error", err), slog.String("schema", schema), slog.String("username", roleName))
		}

		if _, err := tx.Exec(ctx, fmt.Sprintf(`GRANT USAGE ON SCHEMA "%s" TO "%s"`, schema, roleName)); err != nil {
			return fmt.Errorf("failed to grant usage on schema %s: %w", schema, err)
		}
	}

	return nil
}

func grantReadOnExistingObjects(ctx context.Context, tx pgx.Tx, schemas []string, roleName string) error {
	for _, schema := range schemas {
		statement := fmt.Sprintf(`GRANT SELECT ON ALL TABLES IN SCHEMA "%s" TO "%s"`, schema, roleName)
		if _, err := tx.Exec(ctx, statement); err != nil {
			return fmt.Errorf("failed to grant select on tables in schema %s: %w", schema, err)
		}

		statement = fmt.Sprintf(`GRANT SELECT ON ALL SEQUENCES IN SCHEMA "%s" TO "%s"`, schema, roleName)
		if _, err := tx.Exec(ctx, statement); err != nil {
			return fmt.Errorf("failed to grant select on sequences in schema %s: %w", schema, err)
		}
	}

	return nil
}

// ALTER DEFAULT PRIVILEGES without FOR ROLE covers only objects the connecting role creates later.
func grantReadOnFutureOwnObjects(ctx context.Context, tx pgx.Tx, schemas []string, roleName string) error {
	for _, schema := range schemas {
		statement := fmt.Sprintf(
			`ALTER DEFAULT PRIVILEGES IN SCHEMA "%s" GRANT SELECT ON TABLES TO "%s"`,
			schema,
			roleName,
		)
		if _, err := tx.Exec(ctx, statement); err != nil {
			return fmt.Errorf("failed to set default privileges for tables in schema %s: %w", schema, err)
		}

		statement = fmt.Sprintf(
			`ALTER DEFAULT PRIVILEGES IN SCHEMA "%s" GRANT SELECT ON SEQUENCES TO "%s"`,
			schema,
			roleName,
		)
		if _, err := tx.Exec(ctx, statement); err != nil {
			return fmt.Errorf("failed to set default privileges for sequences in schema %s: %w", schema, err)
		}
	}

	return nil
}

type schemaOwner struct {
	SchemaName string
	RoleName   string
}

// Tables that other roles create later, such as new partitions of a table another role owns, need
// ALTER DEFAULT PRIVILEGES FOR ROLE of that owner. This is best effort: a role owned by another
// superuser may refuse it, and the user is created anyway.
func grantReadOnFutureObjectsOfOtherOwners(
	ctx context.Context,
	tx pgx.Tx,
	logger *slog.Logger,
	includeSchemas []string,
	roleName string,
) {
	schemaOwners, err := listOtherSchemaOwners(ctx, tx, includeSchemas)
	if err != nil {
		logger.WarnContext(ctx, "failed to query object owners for default privileges", slog.Any("error", err))

		return
	}

	for _, owner := range schemaOwners {
		for _, objectKind := range []string{"TABLES", "SEQUENCES"} {
			statement := fmt.Sprintf(
				`ALTER DEFAULT PRIVILEGES FOR ROLE "%s" IN SCHEMA "%s" GRANT SELECT ON %s TO "%s"`,
				owner.RoleName, owner.SchemaName, objectKind, roleName,
			)
			if _, err := tx.Exec(ctx, statement); err != nil {
				logger.WarnContext(ctx, "failed to set default privileges for role",
					slog.Any("error", err),
					slog.String("object_kind", objectKind),
					slog.String("role", owner.RoleName),
					slog.String("schema", owner.SchemaName),
					slog.String("readonly_user", roleName),
				)
			}
		}
	}

	if len(schemaOwners) > 0 {
		logger.InfoContext(ctx, "set default privileges for existing object owners",
			slog.String("readonly_user", roleName), slog.Int("owner_count", len(schemaOwners)))
	}
}

func listOtherSchemaOwners(ctx context.Context, tx pgx.Tx, includeSchemas []string) ([]schemaOwner, error) {
	var (
		rows pgx.Rows
		err  error
	)

	if len(includeSchemas) > 0 {
		rows, err = tx.Query(ctx, `
			SELECT DISTINCT n.nspname as schema_name, pg_get_userbyid(c.relowner) as role_name
			FROM pg_class c
			JOIN pg_namespace n ON c.relnamespace = n.oid
			WHERE n.nspname NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
			  AND n.nspname = ANY($1::text[])
			  AND c.relkind IN ('r', 'p', 'v', 'm', 'f')
			  AND pg_get_userbyid(c.relowner) != current_user
			ORDER BY n.nspname, role_name
		`, includeSchemas)
	} else {
		rows, err = tx.Query(ctx, `
			SELECT DISTINCT n.nspname as schema_name, pg_get_userbyid(c.relowner) as role_name
			FROM pg_class c
			JOIN pg_namespace n ON c.relnamespace = n.oid
			WHERE n.nspname NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
			  AND c.relkind IN ('r', 'p', 'v', 'm', 'f')
			  AND pg_get_userbyid(c.relowner) != current_user
			ORDER BY n.nspname, role_name
		`)
	}

	if err != nil {
		return nil, err
	}

	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (schemaOwner, error) {
		var owner schemaOwner
		err := row.Scan(&owner.SchemaName, &owner.RoleName)

		return owner, err
	})
}

func verifyRoleExists(ctx context.Context, tx pgx.Tx, roleName string) error {
	var verifiedRoleName string

	if err := tx.QueryRow(ctx, `SELECT rolname FROM pg_roles WHERE rolname = $1`, roleName).
		Scan(&verifiedRoleName); err != nil {
		return fmt.Errorf("failed to verify user creation: %w", err)
	}

	return nil
}

func listRoleAttributes(ctx context.Context, conn *pgx.Conn) ([]string, error) {
	var isSuperuser, canCreateRole, canCreateDB, canBypassRLS, canReplicate bool

	err := conn.QueryRow(ctx, `
		SELECT
			rolsuper,
			rolcreaterole,
			rolcreatedb,
			rolbypassrls,
			rolreplication
		FROM pg_roles
		WHERE rolname = current_user
	`).Scan(&isSuperuser, &canCreateRole, &canCreateDB, &canBypassRLS, &canReplicate)
	if err != nil {
		return nil, fmt.Errorf("failed to check role attributes: %w", err)
	}

	return collectHeldPrivileges([]heldPrivilege{
		{Name: "SUPERUSER", IsHeld: isSuperuser},
		{Name: "CREATEROLE", IsHeld: canCreateRole},
		{Name: "CREATEDB", IsHeld: canCreateDB},
		{Name: "BYPASSRLS", IsHeld: canBypassRLS},
		{Name: "REPLICATION", IsHeld: canReplicate},
	}), nil
}

func listDatabasePrivileges(ctx context.Context, conn *pgx.Conn) ([]string, error) {
	var canCreate, canCreateTemp bool

	err := conn.QueryRow(ctx, `
		SELECT
			has_database_privilege(current_user, current_database(), 'CREATE') as can_create,
			has_database_privilege(current_user, current_database(), 'TEMP') as can_temp
	`).Scan(&canCreate, &canCreateTemp)
	if err != nil {
		return nil, fmt.Errorf("failed to check database privileges: %w", err)
	}

	return collectHeldPrivileges([]heldPrivilege{
		{Name: "CREATE (database)", IsHeld: canCreate},
		{Name: "TEMP", IsHeld: canCreateTemp},
	}), nil
}

func listSchemaCreatePrivilege(ctx context.Context, conn *pgx.Conn) ([]string, error) {
	var hasSchemaCreate bool

	err := conn.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM pg_namespace n
			WHERE has_schema_privilege(current_user, n.nspname, 'CREATE')
			AND nspname NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
		)
	`).Scan(&hasSchemaCreate)
	if err != nil {
		return nil, fmt.Errorf("failed to check schema privileges: %w", err)
	}

	return collectHeldPrivileges([]heldPrivilege{{Name: "CREATE (schema)", IsHeld: hasSchemaCreate}}), nil
}

func listTableWritePrivileges(ctx context.Context, conn *pgx.Conn) ([]string, error) {
	rows, err := conn.Query(ctx, `
		SELECT DISTINCT privilege_type
		FROM information_schema.role_table_grants
		WHERE grantee = current_user
		AND table_schema NOT IN ('pg_catalog', 'information_schema')
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to check table privileges: %w", err)
	}

	tablePrivileges, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, fmt.Errorf("error iterating privileges: %w", err)
	}

	var writePrivileges []string

	for _, tablePrivilege := range tablePrivileges {
		if tableWritePrivileges[tablePrivilege] {
			writePrivileges = append(writePrivileges, tablePrivilege)
		}
	}

	return writePrivileges, nil
}

func listSecurityDefinerExecutePrivilege(ctx context.Context, conn *pgx.Conn) ([]string, error) {
	var functionCount int

	err := conn.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM pg_proc p
		JOIN pg_namespace n ON p.pronamespace = n.oid
		WHERE n.nspname NOT IN ('pg_catalog', 'information_schema')
		AND p.prosecdef = true
		AND has_function_privilege(current_user, p.oid, 'EXECUTE')
	`).Scan(&functionCount)
	if err != nil {
		return nil, fmt.Errorf("failed to check function privileges: %w", err)
	}

	return collectHeldPrivileges([]heldPrivilege{{Name: "EXECUTE (SECURITY DEFINER)", IsHeld: functionCount > 0}}), nil
}

type heldPrivilege struct {
	Name   string
	IsHeld bool
}

func collectHeldPrivileges(privileges []heldPrivilege) []string {
	var heldPrivilegeNames []string

	for _, privilege := range privileges {
		if privilege.IsHeld {
			heldPrivilegeNames = append(heldPrivilegeNames, privilege.Name)
		}
	}

	return heldPrivilegeNames
}
