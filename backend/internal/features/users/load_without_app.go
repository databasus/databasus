//go:build !testhelpers

package users_bridge

import "databasus-backend/internal/modules/users"

func loadWithoutApp() users.Module {
	panic("users_bridge.GetModule is called before app.New installed the users module")
}
