package httpapi

import (
	"context"

	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/modules/workspaces/internal/workspace"
)

func (h *Handler) ListWorkspaces(
	ctx context.Context,
	request ListWorkspacesRequestObject,
) (ListWorkspacesResponseObject, error) {
	actor, err := users.RequireActor(ctx)
	if err != nil {
		return nil, err
	}

	page, err := h.workspaces.ListWorkspaces(ctx, actor, workspace.WorkspaceListQuery{
		Limit: request.Params.Limit,
		After: request.Params.After,
	})
	if err != nil {
		return nil, err
	}

	items := make([]Workspace, 0, len(page.Workspaces))
	for _, listedWorkspace := range page.Workspaces {
		items = append(items, toWorkspaceBody(listedWorkspace))
	}

	return ListWorkspaces200JSONResponse{Items: items, NextCursor: page.NextCursor}, nil
}

func (h *Handler) CreateWorkspace(
	ctx context.Context,
	request CreateWorkspaceRequestObject,
) (CreateWorkspaceResponseObject, error) {
	actor, err := users.RequireActor(ctx)
	if err != nil {
		return nil, err
	}

	createdWorkspace, err := h.workspaces.CreateWorkspace(ctx, actor, request.Body.Name)
	if err != nil {
		return nil, err
	}

	return CreateWorkspace201JSONResponse(toWorkspaceBody(createdWorkspace)), nil
}

func (h *Handler) GetWorkspace(
	ctx context.Context,
	request GetWorkspaceRequestObject,
) (GetWorkspaceResponseObject, error) {
	actor, err := users.RequireActor(ctx)
	if err != nil {
		return nil, err
	}

	foundWorkspace, err := h.workspaces.GetWorkspace(ctx, actor, request.WorkspaceId)
	if err != nil {
		return nil, err
	}

	return GetWorkspace200JSONResponse(toWorkspaceBody(foundWorkspace)), nil
}

func (h *Handler) UpdateWorkspace(
	ctx context.Context,
	request UpdateWorkspaceRequestObject,
) (UpdateWorkspaceResponseObject, error) {
	actor, err := users.RequireActor(ctx)
	if err != nil {
		return nil, err
	}

	updatedWorkspace, err := h.workspaces.UpdateWorkspace(ctx, actor, request.WorkspaceId, request.Body.Name)
	if err != nil {
		return nil, err
	}

	return UpdateWorkspace200JSONResponse(toWorkspaceBody(updatedWorkspace)), nil
}

func (h *Handler) DeleteWorkspace(
	ctx context.Context,
	request DeleteWorkspaceRequestObject,
) (DeleteWorkspaceResponseObject, error) {
	actor, err := users.RequireActor(ctx)
	if err != nil {
		return nil, err
	}

	err = h.workspaces.DeleteWorkspace(ctx, actor, request.WorkspaceId)
	if err != nil {
		return nil, err
	}

	return DeleteWorkspace204Response{}, nil
}

func toWorkspaceBody(listedWorkspace workspace.ListedWorkspace) Workspace {
	var currentUserRole *WorkspaceRole

	if listedWorkspace.CurrentUserRole != nil {
		role := WorkspaceRole(*listedWorkspace.CurrentUserRole)
		currentUserRole = &role
	}

	return Workspace{
		Id:              listedWorkspace.ID,
		Name:            listedWorkspace.Name,
		CreatedAt:       listedWorkspace.CreatedAt,
		CurrentUserRole: currentUserRole,
	}
}
