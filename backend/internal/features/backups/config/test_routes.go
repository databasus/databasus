//go:build testhelpers

package backups_bridge

import (
	"github.com/gin-gonic/gin"

	"databasus-backend/internal/lib/apierror"
	"databasus-backend/internal/util/logger"
)

// TestRoutes mounts the module's contract routes on a router of the old tests. That router signs a
// request in through the session middleware instead of the contract guard.
type TestRoutes struct{}

func GetTestRoutes() TestRoutes {
	return TestRoutes{}
}

func (TestRoutes) RegisterRoutes(router *gin.RouterGroup) {
	GetModule().RegisterRoutes(router, apierror.NewHooks(apierror.Deps{Logger: logger.GetLogger()}))
}
