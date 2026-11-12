package containers

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"

	"databasus-backend/internal/lib/testwait"
)

func (fixture *Patroni) Connect(t *testing.T, endpoint Endpoint) *pgx.Conn {
	t.Helper()

	var connection *pgx.Conn
	testwait.Until(t, "Patroni node accepts SQL connections", func(ctx context.Context) error {
		var err error
		connection, err = pgx.Connect(
			ctx,
			fmt.Sprintf(
				"host=%s port=%d user=%s password=%s dbname=postgres sslmode=disable connect_timeout=3",
				endpoint.Host,
				endpoint.Port,
				PostgresUsername,
				PostgresPassword,
			),
		)

		return err
	})

	t.Cleanup(func() { _ = connection.Close(context.WithoutCancel(t.Context())) })

	return connection
}
