//go:build !testhelpers

package logger

import "databasus-backend/internal/lib/logging"

func loadWithoutApp() *logging.Logging {
	panic("logger.GetLogger is called before app.New installed the logging")
}
