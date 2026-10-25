package workspaces_controllers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	workspaces_dto "databasus-backend/internal/features/workspaces/dto"
	workspaces_errors "databasus-backend/internal/features/workspaces/errors"
	workspaces_services "databasus-backend/internal/features/workspaces/services"
	"databasus-backend/internal/modules/users"
)

type MembershipController struct {
	membershipService *workspaces_services.MembershipService
}

func (c *MembershipController) RegisterRoutes(router *gin.RouterGroup) {
	workspaceRoutes := router.Group("/workspaces/memberships/:id")

	workspaceRoutes.GET("/members", c.ListMembers)
	workspaceRoutes.POST("/members", c.AddMember)
	workspaceRoutes.PUT("/members/:userId/role", c.ChangeMemberRole)
	workspaceRoutes.DELETE("/members/:userId", c.RemoveMember)
	workspaceRoutes.POST("/transfer-ownership", c.TransferOwnership)
}

func (c *MembershipController) ListMembers(ctx *gin.Context) {
	actor, ok := users.GetActor(ctx.Request.Context())
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	workspaceIDStr := ctx.Param("id")
	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid workspace ID"})
		return
	}

	response, err := c.membershipService.GetMembers(ctx.Request.Context(), workspaceID, actor)
	if err != nil {
		if errors.Is(err, workspaces_errors.ErrInsufficientPermissionsToViewMembers) {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

func (c *MembershipController) AddMember(ctx *gin.Context) {
	actor, ok := users.GetActor(ctx.Request.Context())
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	workspaceIDStr := ctx.Param("id")
	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid workspace ID"})
		return
	}

	var request workspaces_dto.AddMemberRequestDTO
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	if !request.Role.IsValid() {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role"})
		return
	}

	response, err := c.membershipService.AddMember(ctx.Request.Context(), workspaceID, &request, actor)
	if err != nil {
		if errors.Is(err, workspaces_errors.ErrInsufficientPermissionsToManageMembers) ||
			errors.Is(err, workspaces_errors.ErrOnlyOwnerCanAddManageAdmins) {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

func (c *MembershipController) ChangeMemberRole(ctx *gin.Context) {
	actor, ok := users.GetActor(ctx.Request.Context())
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	workspaceIDStr := ctx.Param("id")
	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid workspace ID"})
		return
	}

	userIDStr := ctx.Param("userId")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var request workspaces_dto.ChangeMemberRoleRequestDTO
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	if err := c.membershipService.ChangeMemberRole(
		ctx.Request.Context(),
		workspaceID,
		userID,
		&request,
		actor,
	); err != nil {
		if errors.Is(err, workspaces_errors.ErrInsufficientPermissionsToManageMembers) ||
			errors.Is(err, workspaces_errors.ErrOnlyOwnerCanAddManageAdmins) {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Member role changed successfully"})
}

func (c *MembershipController) RemoveMember(ctx *gin.Context) {
	actor, ok := users.GetActor(ctx.Request.Context())
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	workspaceIDStr := ctx.Param("id")
	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid workspace ID"})
		return
	}

	userIDStr := ctx.Param("userId")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	if err := c.membershipService.RemoveMember(ctx.Request.Context(), workspaceID, userID, actor); err != nil {
		if errors.Is(err, workspaces_errors.ErrInsufficientPermissionsToRemoveMembers) ||
			errors.Is(err, workspaces_errors.ErrOnlyOwnerCanRemoveAdmins) {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Member removed successfully"})
}

func (c *MembershipController) TransferOwnership(ctx *gin.Context) {
	actor, ok := users.GetActor(ctx.Request.Context())
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	workspaceIDStr := ctx.Param("id")
	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid workspace ID"})
		return
	}

	var request workspaces_dto.TransferOwnershipRequestDTO
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	if err := c.membershipService.TransferOwnership(ctx.Request.Context(), workspaceID, &request, actor); err != nil {
		if errors.Is(err, workspaces_errors.ErrOnlyOwnerOrAdminCanTransferOwnership) {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Ownership transferred successfully"})
}
