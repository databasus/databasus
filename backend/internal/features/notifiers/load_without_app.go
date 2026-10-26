//go:build !testhelpers

package notifiers_bridge

import "databasus-backend/internal/modules/notifiers"

func loadWithoutApp() notifiers.Module {
	panic("notifiers_bridge.GetModule is called before app.New installed the notifiers module")
}
