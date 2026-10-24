package databases

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	postgresql_shared "databasus-backend/internal/features/databases/databases/postgresql/shared"
	users_middleware "databasus-backend/internal/features/users/middleware"
	users_services "databasus-backend/internal/features/users/services"
	workspaces_services "databasus-backend/internal/features/workspaces/services"
)

type DatabaseController struct {
	databaseService  *DatabaseService
	userService      *users_services.UserService
	workspaceService *workspaces_services.WorkspaceService
}

func (c *DatabaseController) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/databases/create", c.CreateDatabase)
	router.POST("/databases/update", c.UpdateDatabase)
	router.DELETE("/databases/:id", c.DeleteDatabase)
	router.GET("/databases/:id", c.GetDatabase)
	router.GET("/databases", c.GetDatabases)
	router.POST("/databases/:id/test-connection", c.TestDatabaseConnection)
	router.POST("/databases/test-connection-direct", c.TestDatabaseConnectionDirect)
	router.POST("/databases/:id/copy", c.CopyDatabase)
	router.GET("/databases/notifier/:id/is-using", c.IsNotifierUsing)
	router.GET("/databases/notifier/:id/databases-count", c.CountDatabasesByNotifier)
	router.POST("/databases/should-suggest-readonly-user", c.ShouldSuggestReadOnlyUser)
	router.POST("/databases/create-readonly-user", c.CreateReadOnlyUser)
	router.POST("/databases/create-replication-only-user", c.CreateReplicationOnlyUser)
}

func (c *DatabaseController) RegisterPublicRoutes(_ *gin.RouterGroup) {
}

func (c *DatabaseController) CreateDatabase(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var request Database
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if request.WorkspaceID == nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "workspaceId is required"})
		return
	}

	database, err := c.databaseService.CreateDatabase(ctx.Request.Context(), user, *request.WorkspaceID, &request)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, database)
}

func (c *DatabaseController) UpdateDatabase(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var request Database
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := c.databaseService.UpdateDatabase(ctx.Request.Context(), user, &request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, request)
}

func (c *DatabaseController) DeleteDatabase(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid database ID"})
		return
	}

	if err := c.databaseService.DeleteDatabase(ctx.Request.Context(), user, id); err != nil {
		_ = ctx.Error(err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.Status(http.StatusNoContent)
}

func (c *DatabaseController) GetDatabase(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid database ID"})
		return
	}

	database, err := c.databaseService.GetDatabase(ctx.Request.Context(), user, id)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, database)
}

func (c *DatabaseController) GetDatabases(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	workspaceIDStr := ctx.Query("workspace_id")
	if workspaceIDStr == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "workspace_id query parameter is required"})
		return
	}

	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid workspace_id"})
		return
	}

	databases, err := c.databaseService.GetDatabasesByWorkspace(ctx.Request.Context(), user, workspaceID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, databases)
}

func (c *DatabaseController) TestDatabaseConnection(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid database ID"})
		return
	}

	if err := c.databaseService.TestDatabaseConnection(ctx.Request.Context(), user, id); err != nil {
		respondConnectionTestError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "connection successful"})
}

func (c *DatabaseController) TestDatabaseConnectionDirect(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var request Database
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := c.databaseService.TestDatabaseConnectionDirect(ctx.Request.Context(), user, &request); err != nil {
		respondConnectionTestError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "connection successful"})
}

func respondConnectionTestError(ctx *gin.Context, err error) {
	if errors.Is(err, ErrInsufficientPermissionsToTestDatabaseConnection) {
		ctx.JSON(http.StatusForbidden, gin.H{"error": ErrInsufficientPermissionsToTestDatabaseConnection.Error()})
		return
	}

	if errors.Is(err, ErrDatabaseConnectionTargetLookup) {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": ErrDatabaseConnectionTargetLookup.Error()})
		return
	}

	if errors.Is(err, ErrDatabaseConnectionAuthorizationLookup) {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": ErrDatabaseConnectionAuthorizationLookup.Error()})
		return
	}

	if connErr, ok := errors.AsType[*postgresql_shared.ConnectionTestError](err); ok {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": connErr.Code})
		return
	}

	ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
}

func (c *DatabaseController) IsNotifierUsing(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid notifier ID"})
		return
	}

	isUsing, err := c.databaseService.IsNotifierUsing(ctx.Request.Context(), user, id)
	if err != nil {
		_ = ctx.Error(err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"isUsing": isUsing})
}

func (c *DatabaseController) CountDatabasesByNotifier(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid notifier ID"})
		return
	}

	count, err := c.databaseService.CountDatabasesByNotifier(ctx.Request.Context(), user, id)
	if err != nil {
		_ = ctx.Error(err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"count": count})
}

func (c *DatabaseController) CopyDatabase(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid database ID"})
		return
	}

	copiedDatabase, err := c.databaseService.CopyDatabase(ctx.Request.Context(), user, id)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, copiedDatabase)
}

func (c *DatabaseController) ShouldSuggestReadOnlyUser(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var request Database
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	shouldSuggestReadOnlyUser, privileges, err := c.databaseService.ShouldSuggestReadOnlyUser(
		ctx.Request.Context(),
		user,
		&request,
	)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, ShouldSuggestReadOnlyUserResponse{
		ShouldSuggestReadOnlyUser: shouldSuggestReadOnlyUser,
		Privileges:                privileges,
	})
}

func (c *DatabaseController) CreateReadOnlyUser(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var request Database
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	username, password, err := c.databaseService.CreateReadOnlyUser(ctx.Request.Context(), user, &request)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, CreateReadOnlyUserResponse{
		Username: username,
		Password: password,
	})
}

func (c *DatabaseController) CreateReplicationOnlyUser(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var request Database
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	createdUser, err := c.databaseService.CreateReplicationOnlyUser(ctx.Request.Context(), user, &request)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, CreateReplicationOnlyUserResponse{
		Username:                     createdUser.Username,
		Password:                     createdUser.Password,
		IsForcedWalRotationAvailable: createdUser.IsForcedWalRotationAvailable,
	})
}
