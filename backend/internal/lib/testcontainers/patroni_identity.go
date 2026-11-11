package containers

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"

	"databasus-backend/internal/lib/testwait"
)

type PatroniIdentity struct {
	Node             string
	Timeline         int
	SystemIdentifier string
}

func (fixture *Patroni) GetIdentity(t *testing.T) PatroniIdentity {
	t.Helper()

	var identity PatroniIdentity
	testwait.Until(t, "stable Patroni endpoint serves the elected primary", func(ctx context.Context) error {
		connection, err := pgx.Connect(
			ctx,
			fmt.Sprintf(
				"host=%s port=%d user=%s password=%s dbname=postgres sslmode=disable connect_timeout=2",
				fixture.Endpoint.Host,
				fixture.Endpoint.Port,
				PostgresUsername,
				PostgresPassword,
			),
		)
		if err != nil {
			return err
		}

		defer func() { _ = connection.Close(ctx) }()

		var address string
		var isStandby bool
		err = connection.QueryRow(ctx, "SELECT host(inet_server_addr()), pg_is_in_recovery(), ('x'||substr(pg_walfile_name(pg_current_wal_lsn()),1,8))::bit(32)::int, system_identifier::text FROM pg_control_system()").
			Scan(&address, &isStandby, &identity.Timeline, &identity.SystemIdentifier)
		if err != nil {
			return err
		}

		if isStandby {
			return fmt.Errorf("endpoint serves a standby")
		}

		for _, node := range fixture.Nodes {
			if node.Address == address {
				identity.Node = node.Name
				return nil
			}
		}

		return fmt.Errorf("unknown serving node %s", address)
	})

	return identity
}
