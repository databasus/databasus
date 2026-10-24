//go:build testhelpers

package workspaces_testing

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"databasus-backend/internal/features/audit_logs"
	users_dto "databasus-backend/internal/features/users/dto"
	users_enums "databasus-backend/internal/features/users/enums"
	users_middleware "databasus-backend/internal/features/users/middleware"
	users_services "databasus-backend/internal/features/users/services"
	users_testing "databasus-backend/internal/features/users/testing"
	workspaces_dto "databasus-backend/internal/features/workspaces/dto"
	workspaces_models "databasus-backend/internal/features/workspaces/models"
	workspaces_repositories "databasus-backend/internal/features/workspaces/repositories"
)

func CreateTestRouter(controllers ...ControllerInterface) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	v1 := router.Group("/api/v1")
	protected := v1.Group("").Use(users_middleware.AuthMiddleware(users_services.GetUserService()))

	for _, controller := range controllers {
		if routerGroup, ok := protected.(*gin.RouterGroup); ok {
			controller.RegisterRoutes(routerGroup)
		}
	}

	audit_logs.SetupDependencies()

	return router
}

func CreateTestWorkspace(
	ctx context.Context,
	name string,
	owner *users_dto.SignInResponseDTO,
	router *gin.Engine,
) *workspaces_models.Workspace {
	workspace, _ := CreateTestWorkspaceViaAPI(ctx, name, owner, router)
	return workspace
}

func CreateTestWorkspaceViaAPI(
	ctx context.Context,
	name string,
	owner *users_dto.SignInResponseDTO,
	router *gin.Engine,
) (*workspaces_models.Workspace, string) {
	return createTestWorkspaceViaAPI(ctx, name, owner, router, true)
}

func CreateTestWorkspaceViaAPIWithoutSettingsChange(
	ctx context.Context,
	name string,
	owner *users_dto.SignInResponseDTO,
	router *gin.Engine,
) (*workspaces_models.Workspace, string) {
	return createTestWorkspaceViaAPI(ctx, name, owner, router, false)
}

func createTestWorkspaceViaAPI(
	ctx context.Context,
	name string,
	owner *users_dto.SignInResponseDTO,
	router *gin.Engine,
	enableMemberCreation bool,
) (*workspaces_models.Workspace, string) {
	if enableMemberCreation {
		users_testing.EnableMemberWorkspaceCreation(ctx)
		defer users_testing.ResetSettingsToDefaults(ctx)
	}

	request := workspaces_dto.CreateWorkspaceRequestDTO{Name: name}
	w := MakeAPIRequest(router, "POST", "/api/v1/workspaces", "Bearer "+owner.Token, request)

	if w.Code != http.StatusOK {
		panic(
			fmt.Sprintf(
				"Failed to create workspace. Status: %d, Body: %s",
				w.Code,
				w.Body.String(),
			),
		)
	}

	var response workspaces_dto.WorkspaceResponseDTO
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		panic(err)
	}

	workspace := &workspaces_models.Workspace{
		ID:   response.ID,
		Name: response.Name,
	}

	return workspace, owner.Token
}

func CreateTestWorkspaceWithToken(
	ctx context.Context,
	name string,
	token string,
	router *gin.Engine,
) (*workspaces_models.Workspace, string) {
	return createTestWorkspaceWithToken(ctx, name, token, router, true)
}

func CreateTestWorkspaceWithTokenWithoutSettingsChange(
	ctx context.Context,
	name string,
	token string,
	router *gin.Engine,
) (*workspaces_models.Workspace, string) {
	return createTestWorkspaceWithToken(ctx, name, token, router, false)
}

func createTestWorkspaceWithToken(
	ctx context.Context,
	name string,
	token string,
	router *gin.Engine,
	enableMemberCreation bool,
) (*workspaces_models.Workspace, string) {
	if enableMemberCreation {
		users_testing.EnableMemberWorkspaceCreation(ctx)
		defer users_testing.ResetSettingsToDefaults(ctx)
	}

	request := workspaces_dto.CreateWorkspaceRequestDTO{Name: name}
	w := MakeAPIRequest(router, "POST", "/api/v1/workspaces", "Bearer "+token, request)

	if w.Code != http.StatusOK {
		panic(
			fmt.Sprintf(
				"Failed to create workspace. Status: %d, Body: %s",
				w.Code,
				w.Body.String(),
			),
		)
	}

	var response workspaces_dto.WorkspaceResponseDTO
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		panic(err)
	}

	workspace := &workspaces_models.Workspace{
		ID:   response.ID,
		Name: response.Name,
	}

	return workspace, token
}

func AddMemberToWorkspace(
	workspace *workspaces_models.Workspace,
	member *users_dto.SignInResponseDTO,
	role users_enums.WorkspaceRole,
	ownerToken string,
	router *gin.Engine,
) {
	request := workspaces_dto.AddMemberRequestDTO{
		Email: member.Email,
		Role:  role,
	}

	w := MakeAPIRequest(
		router,
		"POST",
		"/api/v1/workspaces/memberships/"+workspace.ID.String()+"/members",
		"Bearer "+ownerToken,
		request,
	)

	if w.Code != http.StatusOK {
		panic("Failed to add member to workspace via API: " + w.Body.String())
	}
}

func AddMemberToWorkspaceViaOwner(
	ctx context.Context,
	workspace *workspaces_models.Workspace,
	member *users_dto.SignInResponseDTO,
	role users_enums.WorkspaceRole,
	router *gin.Engine,
) {
	membershipRepo := &workspaces_repositories.MembershipRepository{}
	workspaceMembers, err := membershipRepo.GetWorkspaceMembers(workspace.ID)
	if err != nil {
		panic("Failed to get workspace members: " + err.Error())
	}

	var ownerToken string
	for _, m := range workspaceMembers {
		if m.Role == users_enums.WorkspaceRoleOwner {
			userService := users_services.GetUserService()

			owner, err := userService.GetUserByID(ctx, m.UserID)
			if err != nil {
				panic("Failed to get owner user: " + err.Error())
			}

			tokenResponse, err := userService.GenerateAccessToken(ctx, owner)
			if err != nil {
				panic("Failed to generate owner token: " + err.Error())
			}

			ownerToken = tokenResponse.Token

			break
		}
	}

	if ownerToken == "" {
		panic("No workspace owner found")
	}

	AddMemberToWorkspace(workspace, member, role, ownerToken, router)
}

func InviteMemberToWorkspace(
	workspace *workspaces_models.Workspace,
	email string,
	role users_enums.WorkspaceRole,
	inviterToken string,
	router *gin.Engine,
) *workspaces_dto.AddMemberResponseDTO {
	request := workspaces_dto.AddMemberRequestDTO{
		Email: email,
		Role:  role,
	}

	w := MakeAPIRequest(
		router,
		"POST",
		"/api/v1/workspaces/memberships/"+workspace.ID.String()+"/members",
		"Bearer "+inviterToken,
		request,
	)

	if w.Code != http.StatusOK {
		panic("Failed to invite member to workspace via API: " + w.Body.String())
	}

	var response workspaces_dto.AddMemberResponseDTO
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		panic(err)
	}

	return &response
}

func ChangeMemberRole(
	workspace *workspaces_models.Workspace,
	memberUserID uuid.UUID,
	newRole users_enums.WorkspaceRole,
	changerToken string,
	router *gin.Engine,
) {
	request := workspaces_dto.ChangeMemberRoleRequestDTO{
		Role: newRole,
	}

	w := MakeAPIRequest(
		router,
		"PUT",
		fmt.Sprintf(
			"/api/v1/workspaces/memberships/%s/members/%s/role",
			workspace.ID.String(),
			memberUserID.String(),
		),
		"Bearer "+changerToken,
		request,
	)

	if w.Code != http.StatusOK {
		panic("Failed to change member role via API: " + w.Body.String())
	}
}

func RemoveMemberFromWorkspace(
	workspace *workspaces_models.Workspace,
	memberUserID uuid.UUID,
	removerToken string,
	router *gin.Engine,
) {
	w := MakeAPIRequest(
		router,
		"DELETE",
		fmt.Sprintf(
			"/api/v1/workspaces/memberships/%s/members/%s",
			workspace.ID.String(),
			memberUserID.String(),
		),
		"Bearer "+removerToken,
		nil,
	)

	if w.Code != http.StatusOK {
		panic("Failed to remove member from workspace via API: " + w.Body.String())
	}
}

func GetWorkspaceMembers(
	workspace *workspaces_models.Workspace,
	requesterToken string,
	router *gin.Engine,
) *workspaces_dto.GetMembersResponseDTO {
	w := MakeAPIRequest(
		router,
		"GET",
		"/api/v1/workspaces/memberships/"+workspace.ID.String()+"/members",
		"Bearer "+requesterToken,
		nil,
	)

	if w.Code != http.StatusOK {
		panic("Failed to get workspace members via API: " + w.Body.String())
	}

	var response workspaces_dto.GetMembersResponseDTO
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		panic(err)
	}

	return &response
}

func UpdateWorkspace(
	workspace *workspaces_models.Workspace,
	updateData *workspaces_models.Workspace,
	updaterToken string,
	router *gin.Engine,
) *workspaces_models.Workspace {
	w := MakeAPIRequest(
		router,
		"PUT",
		"/api/v1/workspaces/"+workspace.ID.String(),
		"Bearer "+updaterToken,
		updateData,
	)

	if w.Code != http.StatusOK {
		panic("Failed to update workspace via API: " + w.Body.String())
	}

	var response workspaces_models.Workspace
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		panic(err)
	}

	return &response
}

func DeleteWorkspace(
	workspace *workspaces_models.Workspace,
	deleterToken string,
	router *gin.Engine,
) {
	w := MakeAPIRequest(
		router,
		"DELETE",
		"/api/v1/workspaces/"+workspace.ID.String(),
		"Bearer "+deleterToken,
		nil,
	)

	if w.Code != http.StatusOK {
		panic("Failed to delete workspace via API: " + w.Body.String())
	}
}

// The context is stripped of cancellation because callers pass t.Context() from a t.Cleanup, and
// the test context is already cancelled by the time cleanup runs.
func RemoveTestWorkspace(ctx context.Context, workspace *workspaces_models.Workspace, router *gin.Engine) {
	ctx = context.WithoutCancel(ctx)

	membershipRepo := &workspaces_repositories.MembershipRepository{}
	workspaceMembers, err := membershipRepo.GetWorkspaceMembers(workspace.ID)
	if err != nil {
		// Workspace might already be deleted or doesn't exist, silently return
		return
	}

	if len(workspaceMembers) == 0 {
		// No members found, workspace might have been deleted, silently return
		return
	}

	var ownerToken string
	for _, m := range workspaceMembers {
		if m.Role == users_enums.WorkspaceRoleOwner {
			userService := users_services.GetUserService()

			owner, err := userService.GetUserByID(ctx, m.UserID)
			if err != nil {
				// Owner user not found, workspace might be in inconsistent state, try direct deletion
				_ = RemoveTestWorkspaceDirect(workspace.ID)
				return
			}

			tokenResponse, err := userService.GenerateAccessToken(ctx, owner)
			if err != nil {
				// Cannot generate token, try direct deletion
				_ = RemoveTestWorkspaceDirect(workspace.ID)
				return
			}

			ownerToken = tokenResponse.Token
			break
		}
	}

	if ownerToken == "" {
		// No owner found, try direct deletion
		_ = RemoveTestWorkspaceDirect(workspace.ID)
		return
	}

	DeleteWorkspace(workspace, ownerToken, router)
}

func MakeAPIRequest(
	router *gin.Engine,
	method, url, authToken string,
	body any,
) *httptest.ResponseRecorder {
	var requestBody *bytes.Buffer
	if body != nil {
		bodyJSON, err := json.Marshal(body)
		if err != nil {
			panic(err)
		}
		requestBody = bytes.NewBuffer(bodyJSON)
	} else {
		requestBody = bytes.NewBuffer(nil)
	}

	req, err := http.NewRequestWithContext(context.Background(), method, url, requestBody)
	if err != nil {
		panic(err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if authToken != "" {
		req.Header.Set("Authorization", authToken)
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// CreateTestWorkspaceDirect creates a workspace directly without using HTTP API
func CreateTestWorkspaceDirect(
	name string,
	ownerID uuid.UUID,
) (*workspaces_models.Workspace, error) {
	repo := &workspaces_repositories.WorkspaceRepository{}
	membershipRepo := &workspaces_repositories.MembershipRepository{}

	workspace := &workspaces_models.Workspace{
		Name: name,
	}

	err := repo.CreateWorkspace(workspace)
	if err != nil {
		return nil, err
	}

	// Create owner membership
	membership := &workspaces_models.WorkspaceMembership{
		WorkspaceID: workspace.ID,
		UserID:      ownerID,
		Role:        users_enums.WorkspaceRoleOwner,
	}

	err = membershipRepo.CreateMembership(membership)
	if err != nil {
		return nil, err
	}

	return workspace, nil
}

// RemoveTestWorkspaceDirect removes a workspace directly without using HTTP API
func RemoveTestWorkspaceDirect(workspaceID uuid.UUID) error {
	repo := &workspaces_repositories.WorkspaceRepository{}
	return repo.DeleteWorkspace(workspaceID)
}
