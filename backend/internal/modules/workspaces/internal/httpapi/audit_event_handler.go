package httpapi

import (
	"context"

	"databasus-backend/internal/lib/httpserver"
	"databasus-backend/internal/modules/workspaces/internal/auditlog"
)

func (h *Handler) ListAuditEvents(
	ctx context.Context,
	request ListAuditEventsRequestObject,
) (ListAuditEventsResponseObject, error) {
	ctx = httpserver.GetRequestContext(ctx)

	actor, err := getActor(ctx)
	if err != nil {
		return nil, err
	}

	page, err := h.auditLog.ListEvents(ctx, actor, auditlog.EventListQuery{
		WorkspaceID:  request.Params.WorkspaceId,
		AuthorUserID: request.Params.AuthorUserId,
		Limit:        request.Params.Limit,
		After:        request.Params.After,
	})
	if err != nil {
		return nil, err
	}

	items := make([]AuditEvent, 0, len(page.Events))
	for _, listedEvent := range page.Events {
		items = append(items, toAuditEventBody(listedEvent))
	}

	return ListAuditEvents200JSONResponse{Items: items, NextCursor: page.NextCursor}, nil
}

func toAuditEventBody(listedEvent auditlog.ListedEvent) AuditEvent {
	var author *AuditEventAuthor

	if listedEvent.Author != nil {
		author = &AuditEventAuthor{
			UserId: listedEvent.Author.UserID,
			Email:  listedEvent.Author.Email,
			Name:   listedEvent.Author.Name,
		}
	}

	var eventWorkspace *AuditEventWorkspace

	if listedEvent.Workspace != nil {
		eventWorkspace = &AuditEventWorkspace{Id: listedEvent.Workspace.ID, Name: listedEvent.Workspace.Name}
	}

	return AuditEvent{
		Id:        listedEvent.ID,
		Message:   listedEvent.Message,
		CreatedAt: listedEvent.CreatedAt,
		Author:    author,
		Workspace: eventWorkspace,
	}
}
