//go:build testhelpers

package storage

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"databasus-backend/internal/config"
	"databasus-backend/internal/lib/database"
)

func loadWithoutApp() *gorm.DB {
	db, err := database.Open(context.Background(), config.GetEnv().DatabaseDsn)
	if err != nil {
		panic(fmt.Sprintf("open the test database: %v", err))
	}

	return db
}
