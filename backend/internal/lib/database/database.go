// Package database opens the pool of the internal database and applies its migrations.
package database

import (
	"context"
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

const maxConnections = 10

func Open(ctx context.Context, dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormLogger.Default.LogMode(gormLogger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("connect to the database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get the connection pool: %w", err)
	}

	sqlDB.SetMaxOpenConns(maxConnections)
	sqlDB.SetMaxIdleConns(maxConnections)

	err = sqlDB.PingContext(ctx)
	if err != nil {
		_ = sqlDB.Close()

		return nil, fmt.Errorf("ping the database: %w", err)
	}

	return db, nil
}
