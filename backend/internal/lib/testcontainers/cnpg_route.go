package containers

import (
	"context"
	"crypto/tls"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/testwait"
)

// GetIdentity reads the route's identity with the backup role, the timeline and system identifier on
// a replication connection as Databasus sees them, and requires the answering pod to be the one
// CloudNativePG reports as its current primary.
func (cluster *CNPGCluster) GetIdentity(t *testing.T) RouteIdentity {
	t.Helper()

	var identity RouteIdentity
	testwait.Until(
		t,
		"the Kubernetes Service routes to CloudNativePG's current primary",
		func(ctx context.Context) error {
			var err error
			identity, err = cluster.readIdentity(ctx, t)

			return err
		},
	)

	return identity
}

func (cluster *CNPGCluster) readIdentity(ctx context.Context, t *testing.T) (RouteIdentity, error) {
	t.Helper()

	connection, err := pgx.ConnectConfig(ctx, cluster.getConnectionConfig(t, routeLogin{
		Username: CNPGBackupRole,
		Database: CNPGDatabase,
	}))
	if err != nil {
		return RouteIdentity{}, err
	}
	defer func() { _ = connection.Close(context.WithoutCancel(ctx)) }()

	var address string
	var isInRecovery bool
	if err := connection.QueryRow(ctx, "SELECT host(inet_server_addr()), pg_is_in_recovery()").
		Scan(&address, &isInRecovery); err != nil {
		return RouteIdentity{}, err
	}

	if isInRecovery {
		return RouteIdentity{}, fmt.Errorf("the route serves a standby at %s", address)
	}

	replicationConfig := cluster.getConnectionConfig(t, routeLogin{Username: CNPGBackupRole, Database: CNPGDatabase})
	replicationConfig.RuntimeParams["replication"] = "database"
	replication, err := pgconn.ConnectConfig(ctx, &replicationConfig.Config)
	if err != nil {
		return RouteIdentity{}, err
	}
	defer func() { _ = replication.Close(context.WithoutCancel(ctx)) }()

	results, err := replication.Exec(ctx, "IDENTIFY_SYSTEM").ReadAll()
	if err != nil {
		return RouteIdentity{}, err
	}

	if len(results) != 1 || len(results[0].Rows) != 1 || len(results[0].Rows[0]) < 2 {
		return RouteIdentity{}, fmt.Errorf("unexpected IDENTIFY_SYSTEM result %v", results)
	}

	row := results[0].Rows[0]
	timeline, err := strconv.Atoi(string(row[1]))
	if err != nil {
		return RouteIdentity{}, err
	}

	status := cluster.getStatus(ctx, t)
	primaryAddress, err := cluster.getPodAddress(ctx, status.CurrentPrimary)
	if err != nil {
		return RouteIdentity{}, err
	}

	if primaryAddress != address {
		return RouteIdentity{}, fmt.Errorf(
			"the route serves %s while CloudNativePG reports %s at %s",
			address,
			status.CurrentPrimary,
			primaryAddress,
		)
	}

	return RouteIdentity{Node: status.CurrentPrimary, Timeline: timeline, SystemIdentifier: string(row[0])}, nil
}

func (cluster *CNPGCluster) getPodAddress(ctx context.Context, pod string) (string, error) {
	output, err := cluster.K3s.Kubectl(ctx, KubectlCommand{Arguments: []string{
		"-n", cluster.Namespace, "get", "pod", pod, "-o", "jsonpath={.status.podIP}",
	}})

	return strings.TrimSpace(string(output)), err
}

// Connect opens a connection to the routed primary as the database owner, verified against the run's
// authority. The harness writes its control rows and schema changes through it.
func (cluster *CNPGCluster) Connect(t *testing.T) *pgx.Conn {
	t.Helper()

	return cluster.connectAs(t, routeLogin{Username: PostgresUsername, Database: CNPGDatabase})
}

// ConnectAsBackupRole opens a connection with the role Databasus uses, to check what it may not do.
func (cluster *CNPGCluster) ConnectAsBackupRole(t *testing.T) *pgx.Conn {
	t.Helper()

	return cluster.connectAs(t, routeLogin{Username: CNPGBackupRole, Database: CNPGDatabase})
}

type routeLogin struct {
	Username string
	Database string
}

func (cluster *CNPGCluster) connectAs(t *testing.T, login routeLogin) *pgx.Conn {
	t.Helper()

	var connection *pgx.Conn
	testwait.Until(t, "the routed primary accepts verified TLS connections", func(ctx context.Context) error {
		var err error
		connection, err = pgx.ConnectConfig(ctx, cluster.getConnectionConfig(t, login))

		return err
	})
	t.Cleanup(func() { _ = connection.Close(context.WithoutCancel(t.Context())) })

	return connection
}

func (cluster *CNPGCluster) getConnectionConfig(t *testing.T, login routeLogin) *pgx.ConnConfig {
	t.Helper()

	config, err := pgx.ParseConfig(fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable connect_timeout=3",
		cluster.Endpoint.Host,
		cluster.Endpoint.Port,
		login.Username,
		PostgresPassword,
		login.Database,
	))
	require.NoError(t, err)
	config.TLSConfig = cluster.getHarnessTLS()
	config.Fallbacks = nil

	return config
}

func (cluster *CNPGCluster) getHarnessTLS() *tls.Config {
	verifiedName := cluster.verifiedName
	if verifiedName == "" {
		verifiedName = cluster.Endpoint.Host
	}

	return cluster.issuer.GetClientTLS(verifiedName)
}
