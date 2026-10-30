package physical

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/jackc/pgx/v5"

	"databasus-backend/internal/lib/crypto"
)

const maxReplicationUserAttempts = 3

// platform is the detected kind of the source cluster. It picks the replication grant and the fix
// a refusal message names, because managed PostgreSQL cannot ALTER SYSTEM and the user sets the
// parameter elsewhere.
type platform string

const (
	platformSelfManaged    platform = "self_managed"
	platformRds            platform = "rds"
	platformAzure          platform = "azure"
	platformGcp            platform = "gcp"
	platformUnknownManaged platform = "unknown_managed"
)

type ReplicationOnlyUser struct {
	Username                     string
	Password                     string
	IsForcedWalRotationAvailable bool
}

type cloudAdminRole struct {
	Name  string
	Label string
}

func getExcessiveCloudRoles() []cloudAdminRole {
	return []cloudAdminRole{
		{Name: "rds_superuser", Label: "rds_superuser (RDS admin)"},
		{Name: "azure_pg_admin", Label: "azure_pg_admin (Azure admin)"},
		{Name: "cloudsqlsuperuser", Label: "cloudsqlsuperuser (GCP admin)"},
		{Name: "pg_write_all_data", Label: "pg_write_all_data"},
	}
}

//nolint:gochecknoglobals // Shared immutable privilege lookup is reused across table grants; it is never mutated.
var tableWritePrivileges = map[string]bool{
	"INSERT":     true,
	"UPDATE":     true,
	"DELETE":     true,
	"TRUNCATE":   true,
	"REFERENCES": true,
	"TRIGGER":    true,
}

// ShouldSuggestReplicationOnlyUser lists what the connecting role may do beyond replication. Cloud
// admin roles (rds_superuser, azure_pg_admin, cloudsqlsuperuser) count as excessive even though
// they do not set rolsuper on managed PostgreSQL. REPLICATION itself is the baseline and is not
// excessive; TestReplicationConnection validates it.
func (s *Source) ShouldSuggestReplicationOnlyUser(ctx context.Context, logger *slog.Logger) (bool, []string, error) {
	conn, err := s.connect(ctx)
	if err != nil {
		return false, nil, fmt.Errorf("failed to connect to cluster: %w", err)
	}
	defer closeConn(ctx, conn, logger)

	var excessivePrivileges []string

	for _, listPrivileges := range []func(context.Context, *pgx.Conn) ([]string, error){
		listExcessiveRoleAttributes,
		listCloudAdminMemberships,
		listTableWritePrivileges,
		listSchemaCreatePrivilege,
		listSecurityDefinerExecutePrivilege,
	} {
		listedPrivileges, err := listPrivileges(ctx, conn)
		if err != nil {
			return false, nil, err
		}

		excessivePrivileges = append(excessivePrivileges, listedPrivileges...)
	}

	return len(excessivePrivileges) > 0, excessivePrivileges, nil
}

// CreateReplicationOnlyUser creates a role that may replicate and, where the source permits it,
// force a WAL switch. Managed platforms grant replication through platform roles and may withhold
// EXECUTE on pg_switch_wal(), so the two capabilities are negotiated independently. A generated name
// that is already taken is retried with a new one; any other failure ends the creation.
func (s *Source) CreateReplicationOnlyUser(ctx context.Context, logger *slog.Logger) (ReplicationOnlyUser, error) {
	conn, err := s.connect(ctx)
	if err != nil {
		return ReplicationOnlyUser{}, fmt.Errorf("failed to connect to cluster: %w", err)
	}
	defer closeConn(ctx, conn, logger)

	if err := checkCanCreateRole(ctx, conn); err != nil {
		return ReplicationOnlyUser{}, err
	}

	sourcePlatform := detectPlatform(ctx, conn)

	for range maxReplicationUserAttempts {
		roleName := s.generateUsername()

		replicationUser, err := createReplicationRole(ctx, conn, logger, roleName, sourcePlatform)
		if classifyServerError(err) == serverErrorNameTaken {
			continue
		}

		if err != nil {
			return ReplicationOnlyUser{}, err
		}

		logger.InfoContext(ctx, "replication-only user created",
			slog.String("username", roleName), slog.String("platform", string(sourcePlatform)))

		if !replicationUser.IsForcedWalRotationAvailable {
			logger.WarnContext(ctx,
				"source refused the pg_switch_wal grant; continuous wal streaming is unavailable for this user",
				slog.String("username", roleName), slog.String("platform", string(sourcePlatform)))
		}

		return replicationUser, nil
	}

	return ReplicationOnlyUser{}, fmt.Errorf("%w after %d attempts", ErrUserNameCollision, maxReplicationUserAttempts)
}

// createReplicationRole is one attempt in a transaction of its own: a failed CREATE ROLE aborts the
// transaction it ran in, so a retry needs a fresh one.
func createReplicationRole(
	ctx context.Context,
	conn *pgx.Conn,
	logger *slog.Logger,
	roleName string,
	sourcePlatform platform,
) (ReplicationOnlyUser, error) {
	tx, err := conn.Begin(ctx)
	if err != nil {
		return ReplicationOnlyUser{}, fmt.Errorf("failed to begin transaction: %w", err)
	}

	isCommitted := false

	defer func() {
		if isCommitted {
			return
		}

		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
			logger.WarnContext(ctx, "failed to rollback transaction", slog.Any("error", rollbackErr))
		}
	}()

	password := crypto.GeneratePassword()

	if _, err := tx.Exec(
		ctx,
		fmt.Sprintf(`CREATE USER "%s" WITH PASSWORD '%s' LOGIN`, roleName, password),
	); err != nil {
		return ReplicationOnlyUser{}, fmt.Errorf("failed to create user: %w", err)
	}

	if err := grantReplication(ctx, tx, roleName, sourcePlatform); err != nil {
		return ReplicationOnlyUser{}, err
	}

	isWalSwitchGranted, err := grantWalSwitchIfPermitted(ctx, tx, roleName)
	if err != nil {
		return ReplicationOnlyUser{}, err
	}

	var verifiedRoleName string

	if err := tx.QueryRow(ctx, `SELECT rolname FROM pg_roles WHERE rolname = $1`, roleName).
		Scan(&verifiedRoleName); err != nil {
		return ReplicationOnlyUser{}, fmt.Errorf("failed to verify user creation: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return ReplicationOnlyUser{}, fmt.Errorf("failed to commit transaction: %w", err)
	}

	isCommitted = true

	return ReplicationOnlyUser{
		Username:                     roleName,
		Password:                     password,
		IsForcedWalRotationAvailable: isWalSwitchGranted,
	}, nil
}

func detectPlatform(ctx context.Context, conn *pgx.Conn) platform {
	if isRolePresent(ctx, conn, "rds_replication") {
		return platformRds
	}

	if isRolePresent(ctx, conn, "azure_pg_admin") {
		return platformAzure
	}

	if isRolePresent(ctx, conn, "cloudsqlsuperuser") {
		return platformGcp
	}

	var isSuperuser string
	if err := conn.QueryRow(ctx, "SHOW is_superuser").
		Scan(&isSuperuser); err == nil &&
		strings.EqualFold(isSuperuser, "off") {
		return platformUnknownManaged
	}

	return platformSelfManaged
}

func isRolePresent(ctx context.Context, conn *pgx.Conn, roleName string) bool {
	var isPresent bool

	err := conn.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_roles WHERE rolname = $1)`, roleName).Scan(&isPresent)

	return err == nil && isPresent
}

// Cloud admins frequently have rolcreaterole=false yet can create users through membership in the
// platform admin role.
func checkCanCreateRole(ctx context.Context, conn *pgx.Conn) error {
	var canCreateRole, isSuperuser bool

	if err := conn.QueryRow(ctx, `
		SELECT rolcreaterole, rolsuper
		FROM pg_roles
		WHERE rolname = current_user
	`).Scan(&canCreateRole, &isSuperuser); err != nil {
		return fmt.Errorf("failed to check role-create capability: %w", err)
	}

	if canCreateRole || isSuperuser {
		return nil
	}

	for _, cloudAdminRoleName := range []string{"rds_superuser", "azure_pg_admin", "cloudsqlsuperuser"} {
		if !isRolePresent(ctx, conn, cloudAdminRoleName) {
			continue
		}

		var isMember bool
		if err := conn.QueryRow(ctx, `SELECT pg_has_role(current_user, $1, 'MEMBER')`, cloudAdminRoleName).
			Scan(&isMember); err == nil && isMember {
			return nil
		}
	}

	return ErrCannotCreateRoles
}

func grantReplication(ctx context.Context, tx pgx.Tx, roleName string, sourcePlatform platform) error {
	switch sourcePlatform {
	case platformSelfManaged:
		if _, err := tx.Exec(ctx, fmt.Sprintf(`ALTER ROLE "%s" REPLICATION`, roleName)); err != nil {
			return fmt.Errorf("failed to grant REPLICATION: %w", err)
		}
	case platformRds:
		if _, err := tx.Exec(ctx, fmt.Sprintf(`GRANT rds_replication TO "%s"`, roleName)); err != nil {
			return fmt.Errorf("failed to grant rds_replication: %w", err)
		}
	case platformAzure:
		return alterRoleReplicationOnManagedPlatform(
			ctx,
			tx,
			roleName,
			ErrAzureReplicationOff,
			"failed to grant REPLICATION on Azure",
		)
	case platformGcp:
		return alterRoleReplicationOnManagedPlatform(
			ctx,
			tx,
			roleName,
			ErrGcpReplicationOff,
			"failed to grant REPLICATION on GCP",
		)
	case platformUnknownManaged:
		return alterRoleReplicationOnManagedPlatform(
			ctx,
			tx,
			roleName,
			ErrReplicationGrantDeny,
			"failed to grant REPLICATION",
		)
	default:
		return fmt.Errorf("unknown platform: %s", sourcePlatform)
	}

	return nil
}

// A managed platform refuses ALTER ROLE ... REPLICATION until replication is enabled on its side,
// so a refusal turns into the instruction that platform needs.
func alterRoleReplicationOnManagedPlatform(
	ctx context.Context,
	tx pgx.Tx,
	roleName string,
	refusalErr error,
	failureMessage string,
) error {
	_, err := tx.Exec(ctx, fmt.Sprintf(`ALTER ROLE "%s" REPLICATION`, roleName))
	if err == nil {
		return nil
	}

	if classifyServerError(err) == serverErrorPermissionDenied {
		return refusalErr
	}

	return fmt.Errorf("%s: %w", failureMessage, err)
}

func listExcessiveRoleAttributes(ctx context.Context, conn *pgx.Conn) ([]string, error) {
	var isSuperuser, canCreateRole, canCreateDB, canBypassRLS bool

	err := conn.QueryRow(ctx, `
		SELECT rolsuper, rolcreaterole, rolcreatedb, rolbypassrls
		FROM pg_roles
		WHERE rolname = current_user
	`).Scan(&isSuperuser, &canCreateRole, &canCreateDB, &canBypassRLS)
	if err != nil {
		return nil, fmt.Errorf("failed to read role attributes: %w", err)
	}

	return collectHeldPrivileges([]heldPrivilege{
		{Name: "SUPERUSER", IsHeld: isSuperuser},
		{Name: "CREATEROLE", IsHeld: canCreateRole},
		{Name: "CREATEDB", IsHeld: canCreateDB},
		{Name: "BYPASSRLS", IsHeld: canBypassRLS},
	}), nil
}

func listCloudAdminMemberships(ctx context.Context, conn *pgx.Conn) ([]string, error) {
	cloudRoles := getExcessiveCloudRoles()

	cloudRoleNames := make([]string, 0, len(cloudRoles))
	for _, cloudRole := range cloudRoles {
		cloudRoleNames = append(cloudRoleNames, cloudRole.Name)
	}

	rows, err := conn.Query(ctx, `
		SELECT rolname
		FROM pg_roles
		WHERE rolname = ANY($1)
		  AND pg_has_role(current_user, rolname, 'MEMBER')
	`, cloudRoleNames)
	if err != nil {
		return nil, fmt.Errorf("failed to check cloud admin role membership: %w", err)
	}

	memberRoleNames, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, fmt.Errorf("error iterating admin roles: %w", err)
	}

	var heldPrivileges []heldPrivilege
	for _, cloudRole := range cloudRoles {
		heldPrivileges = append(heldPrivileges, heldPrivilege{
			Name:   cloudRole.Label,
			IsHeld: slices.Contains(memberRoleNames, cloudRole.Name),
		})
	}

	return collectHeldPrivileges(heldPrivileges), nil
}

func listTableWritePrivileges(ctx context.Context, conn *pgx.Conn) ([]string, error) {
	rows, err := conn.Query(ctx, `
		SELECT DISTINCT privilege_type
		FROM information_schema.role_table_grants
		WHERE grantee = current_user
		  AND table_schema NOT IN ('pg_catalog', 'information_schema')
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to check table grants: %w", err)
	}

	tablePrivileges, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, fmt.Errorf("error iterating table grants: %w", err)
	}

	var writePrivileges []string

	for _, tablePrivilege := range tablePrivileges {
		if tableWritePrivileges[tablePrivilege] {
			writePrivileges = append(writePrivileges, tablePrivilege)
		}
	}

	return writePrivileges, nil
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
		return nil, fmt.Errorf("failed to check schema CREATE: %w", err)
	}

	return collectHeldPrivileges([]heldPrivilege{{Name: "CREATE (schema)", IsHeld: hasSchemaCreate}}), nil
}

func listSecurityDefinerExecutePrivilege(ctx context.Context, conn *pgx.Conn) ([]string, error) {
	var functionCount int

	err := conn.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM pg_proc proc
		JOIN pg_namespace n ON proc.pronamespace = n.oid
		WHERE n.nspname NOT IN ('pg_catalog', 'information_schema')
		  AND proc.prosecdef = true
		  AND has_function_privilege(current_user, proc.oid, 'EXECUTE')
	`).Scan(&functionCount)
	if err != nil {
		return nil, fmt.Errorf("failed to check SECURITY DEFINER grants: %w", err)
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
