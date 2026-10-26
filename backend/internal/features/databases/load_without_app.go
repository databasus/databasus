//go:build !testhelpers

package databases_bridge

import "databasus-backend/internal/modules/databases"

func loadWithoutApp() databases.Module {
	panic("databases_bridge.GetModule is called before app.New installed the databases module")
}
