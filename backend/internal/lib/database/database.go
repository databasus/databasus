// Package database opens the pool of the internal database, applies its migrations, runs
// transactions, recognizes constraint errors and encodes keyset cursors.
package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/stdlib"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

const maxConnections = 10

// Open builds the pool itself rather than letting GORM open it from the DSN, because GORM replaces the
// pool's only after-connect hook with its own when the DSN names a time zone.
func Open(ctx context.Context, dsn string) (*gorm.DB, error) {
	connConfig, err := pgx.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse the connection string: %w", err)
	}

	sqlDB := stdlib.OpenDB(*connConfig, stdlib.OptionAfterConnect(readTimestamptzInUtc))
	sqlDB.SetMaxOpenConns(maxConnections)
	sqlDB.SetMaxIdleConns(maxConnections)

	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{
		Logger: gormLogger.Default.LogMode(gormLogger.Silent),
	})
	if err != nil {
		_ = sqlDB.Close()

		return nil, fmt.Errorf("connect to the database: %w", err)
	}

	err = sqlDB.PingContext(ctx)
	if err != nil {
		_ = sqlDB.Close()

		return nil, fmt.Errorf("ping the database: %w", err)
	}

	return db, nil
}

// readTimestamptzInUtc keeps times read back in UTC. The driver's default is the process time zone,
// which follows the container's TZ and moves the calendar date and hour of every stored time.
func readTimestamptzInUtc(_ context.Context, conn *pgx.Conn) error {
	conn.TypeMap().RegisterType(&pgtype.Type{
		Name:  "timestamptz",
		OID:   pgtype.TimestamptzOID,
		Codec: &pgtype.TimestamptzCodec{ScanLocation: time.UTC},
	})

	return nil
}
