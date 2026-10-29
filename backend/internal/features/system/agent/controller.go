package system_agent

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"

	system_version "databasus-backend/internal/features/system/version"
)

type AgentController struct {
	BinariesDir string
}

func (c *AgentController) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/system/verification-agent", c.DownloadVerificationAgent)
}

func (c *AgentController) DownloadVerificationAgent(ctx *gin.Context) {
	arch := ctx.Query("arch")
	if arch != "amd64" && arch != "arm64" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "arch must be amd64 or arm64"})
		return
	}

	binaryPath := filepath.Join(c.BinariesDir, "databasus-verification-agent-linux-"+arch)
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "verification agent binary not found"})
		return
	}

	ctx.Header("Content-Type", "application/octet-stream")
	ctx.Header("Content-Disposition", "attachment; filename=databasus-verification-agent")
	ctx.Header(AgentVersionHeader, system_version.GetAppVersion())

	ctx.File(binaryPath)
}
