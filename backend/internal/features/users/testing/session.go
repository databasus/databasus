//go:build testhelpers

package users_testing

import (
	"github.com/gin-gonic/gin"

	users_bridge "databasus-backend/internal/features/users"
)

// GetSessionMiddleware is the session check of the users module, which the test routers of the
// old code mount where the server mounts it.
func GetSessionMiddleware() gin.HandlerFunc {
	return users_bridge.GetModule().GetSessionMiddleware()
}
