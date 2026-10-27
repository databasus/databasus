//go:build !testhelpers

package backups_bridge

import "databasus-backend/internal/modules/backups"

func loadWithoutApp() backups.Module {
	panic("backups_bridge.GetModule is called before app.New installed the backups module")
}
