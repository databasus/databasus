package system_version

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

const defaultVersion = "3.26.0"

type VersionController struct{}

func (c *VersionController) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/system/version", c.GetVersion)
}

func (c *VersionController) GetVersion(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, VersionResponse{Version: GetAppVersion()})
}

// GetAppVersion returns the current application version, falling back to a
// hardcoded default when APP_VERSION is unset.
func GetAppVersion() string {
	version := os.Getenv("APP_VERSION")
	if version == "" {
		return defaultVersion
	}

	return version
}
