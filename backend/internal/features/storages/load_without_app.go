//go:build !testhelpers

package storages_bridge

import "databasus-backend/internal/modules/storages"

func loadWithoutApp() storages.Module {
	panic("storages_bridge.GetModule is called before app.New installed the storages module")
}
