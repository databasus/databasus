package system_agent

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"

	system_version "databasus-backend/internal/features/system/version"
)

type AgentController struct{}

func (c *AgentController) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/system/verification-agent", c.DownloadVerificationAgent)
}

var verificationAgentBinaryPaths = map[string]string{
	"amd64": "agent-binaries/databasus-verification-agent-linux-amd64",
	"arm64": "agent-binaries/databasus-verification-agent-linux-arm64",
}

func (c *AgentController) DownloadVerificationAgent(ctx *gin.Context) {
	binaryPath, isOk := verificationAgentBinaryPaths[ctx.Query("arch")]
	if !isOk {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "arch must be amd64 or arm64"})
		return
	}

	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "verification agent binary not found"})
		return
	}

	ctx.Header("Content-Type", "application/octet-stream")
	ctx.Header("Content-Disposition", "attachment; filename=databasus-verification-agent")
	ctx.Header(AgentVersionHeader, system_version.GetAppVersion())

	ctx.File(binaryPath)
}
