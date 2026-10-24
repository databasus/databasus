//go:build testhelpers

package logger

import (
	"context"
	"fmt"

	"databasus-backend/internal/lib/config"
	"databasus-backend/internal/lib/logging"
)

func loadWithoutApp() *logging.Logging {
	loadedConfig, _, err := config.Load(config.LoadSpec{
		IsDatabaseRequired:       false,
		RootDirectory:            "",
		PublishedDatabaseDsnPath: "",
	})
	if err != nil {
		panic(fmt.Sprintf("load the logging settings: %v", err))
	}

	loadedLogging, err := logging.New(context.Background(), loadedConfig)
	if err != nil {
		panic(fmt.Sprintf("build the logging: %v", err))
	}

	return loadedLogging
}
