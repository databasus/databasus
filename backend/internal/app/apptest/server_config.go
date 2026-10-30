package apptest

import "databasus-backend/internal/lib/config"

func (a *App) GetServerConfig() config.Config {
	return a.environment.Config
}
