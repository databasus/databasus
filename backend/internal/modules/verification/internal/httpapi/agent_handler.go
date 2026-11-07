package httpapi

import (
	"context"

	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/modules/verification/internal/agents"
)

func (h *Handler) GetVerificationAgentCount(
	ctx context.Context,
	_ GetVerificationAgentCountRequestObject,
) (GetVerificationAgentCountResponseObject, error) {
	count, err := h.agents.CountAgents(ctx)
	if err != nil {
		return nil, err
	}
	return GetVerificationAgentCount200JSONResponse{AgentCount: count}, nil
}

func (h *Handler) CreateVerificationAgent(
	ctx context.Context,
	request CreateVerificationAgentRequestObject,
) (CreateVerificationAgentResponseObject, error) {
	actor, err := users.RequireActor(ctx)
	if err != nil {
		return nil, errNoActor
	}
	created, err := h.agents.CreateAgent(ctx, actor, request.Body.Name)
	if err != nil {
		return nil, err
	}
	return CreateVerificationAgent201JSONResponse{Agent: fromAgent(created.Agent), Token: created.Token}, nil
}

func (h *Handler) ListVerificationAgents(
	ctx context.Context,
	request ListVerificationAgentsRequestObject,
) (ListVerificationAgentsResponseObject, error) {
	page, err := h.agents.ListPage(ctx, agents.ListQuery{Limit: request.Params.Limit, After: request.Params.After})
	if err != nil {
		return nil, err
	}
	return ListVerificationAgents200JSONResponse{Items: fromAgents(page.Agents), NextCursor: page.NextCursor}, nil
}

func (h *Handler) RotateVerificationAgentToken(
	ctx context.Context,
	request RotateVerificationAgentTokenRequestObject,
) (RotateVerificationAgentTokenResponseObject, error) {
	actor, err := users.RequireActor(ctx)
	if err != nil {
		return nil, errNoActor
	}
	token, err := h.agents.RotateToken(ctx, actor, request.AgentId)
	if err != nil {
		return nil, err
	}
	return RotateVerificationAgentToken200JSONResponse{Token: token}, nil
}

func (h *Handler) DeleteVerificationAgent(
	ctx context.Context,
	request DeleteVerificationAgentRequestObject,
) (DeleteVerificationAgentResponseObject, error) {
	actor, err := users.RequireActor(ctx)
	if err != nil {
		return nil, errNoActor
	}
	err = h.agents.DeleteAgent(ctx, actor, request.AgentId)
	if err != nil {
		return nil, err
	}
	return DeleteVerificationAgent204Response{}, nil
}
