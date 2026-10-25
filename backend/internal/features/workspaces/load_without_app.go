//go:build !testhelpers

package workspaces_bridge

import "databasus-backend/internal/modules/workspaces"

func loadWithoutApp() workspaces.Module {
	panic("workspaces_bridge.GetModule is called before app.New installed the workspaces module")
}
