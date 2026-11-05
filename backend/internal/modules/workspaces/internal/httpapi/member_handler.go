package httpapi

import (
	"context"

	"databasus-backend/internal/lib/httpserver"
	"databasus-backend/internal/modules/workspaces/internal/access"
	"databasus-backend/internal/modules/workspaces/internal/members"
)

func (h *Handler) ListWorkspaceMembers(
	ctx context.Context,
	request ListWorkspaceMembersRequestObject,
) (ListWorkspaceMembersResponseObject, error) {
	ctx = httpserver.GetRequestContext(ctx)

	actor, err := getActor(ctx)
	if err != nil {
		return nil, err
	}

	page, err := h.members.ListMembers(ctx, actor, members.MemberListQuery{
		WorkspaceID: request.Params.WorkspaceId,
		Limit:       request.Params.Limit,
		After:       request.Params.After,
	})
	if err != nil {
		return nil, err
	}

	items := make([]WorkspaceMember, 0, len(page.Members))
	for _, listedMember := range page.Members {
		items = append(items, toMemberBody(listedMember))
	}

	return ListWorkspaceMembers200JSONResponse{Items: items, NextCursor: page.NextCursor}, nil
}

func (h *Handler) AddWorkspaceMember(
	ctx context.Context,
	request AddWorkspaceMemberRequestObject,
) (AddWorkspaceMemberResponseObject, error) {
	ctx = httpserver.GetRequestContext(ctx)

	actor, err := getActor(ctx)
	if err != nil {
		return nil, err
	}

	addedMember, err := h.members.AddMember(ctx, actor, members.Addition{
		WorkspaceID: request.Body.WorkspaceId,
		Email:       string(request.Body.Email),
		Role:        access.Role(request.Body.Role),
	})
	if err != nil {
		return nil, err
	}

	memberBody := toMemberBody(addedMember.ListedMember)

	return AddWorkspaceMember201JSONResponse{
		Id:          memberBody.Id,
		WorkspaceId: memberBody.WorkspaceId,
		UserId:      memberBody.UserId,
		Email:       memberBody.Email,
		Name:        memberBody.Name,
		Role:        memberBody.Role,
		IsInvited:   memberBody.IsInvited,
		CreatedAt:   memberBody.CreatedAt,
		Outcome:     AddedWorkspaceMemberOutcome(addedMember.Outcome),
	}, nil
}

func (h *Handler) ChangeWorkspaceMemberRole(
	ctx context.Context,
	request ChangeWorkspaceMemberRoleRequestObject,
) (ChangeWorkspaceMemberRoleResponseObject, error) {
	ctx = httpserver.GetRequestContext(ctx)

	actor, err := getActor(ctx)
	if err != nil {
		return nil, err
	}

	changedMember, err := h.members.ChangeMemberRole(ctx, actor, request.MemberId, access.Role(request.Body.Role))
	if err != nil {
		return nil, err
	}

	return ChangeWorkspaceMemberRole200JSONResponse(toMemberBody(changedMember)), nil
}

func (h *Handler) RemoveWorkspaceMember(
	ctx context.Context,
	request RemoveWorkspaceMemberRequestObject,
) (RemoveWorkspaceMemberResponseObject, error) {
	ctx = httpserver.GetRequestContext(ctx)

	actor, err := getActor(ctx)
	if err != nil {
		return nil, err
	}

	err = h.members.RemoveMember(ctx, actor, request.MemberId)
	if err != nil {
		return nil, err
	}

	return RemoveWorkspaceMember204Response{}, nil
}

func (h *Handler) TransferWorkspaceOwnership(
	ctx context.Context,
	request TransferWorkspaceOwnershipRequestObject,
) (TransferWorkspaceOwnershipResponseObject, error) {
	ctx = httpserver.GetRequestContext(ctx)

	actor, err := getActor(ctx)
	if err != nil {
		return nil, err
	}

	err = h.members.TransferOwnership(ctx, actor, request.WorkspaceId, request.Body.NewOwnerMemberId)
	if err != nil {
		return nil, err
	}

	return TransferWorkspaceOwnership204Response{}, nil
}

func toMemberBody(listedMember members.ListedMember) WorkspaceMember {
	return WorkspaceMember{
		Id:          listedMember.ID,
		WorkspaceId: listedMember.WorkspaceID,
		UserId:      listedMember.UserID,
		Email:       listedMember.Email,
		Name:        listedMember.Name,
		Role:        WorkspaceRole(listedMember.Role),
		IsInvited:   listedMember.IsInvited,
		CreatedAt:   listedMember.CreatedAt,
	}
}
