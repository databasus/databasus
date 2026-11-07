package httpapi

import (
	"context"

	"databasus-backend/internal/modules/healthcheck/internal/attempts"
	"databasus-backend/internal/modules/users"
)

func (h *Handler) ListHealthcheckAttempts(
	ctx context.Context,
	request ListHealthcheckAttemptsRequestObject,
) (ListHealthcheckAttemptsResponseObject, error) {
	actor, err := users.RequireActor(ctx)
	if err != nil {
		return nil, errNoActor
	}

	limit := 1440
	if request.Params.Limit != nil {
		limit = *request.Params.Limit
	}

	page, err := h.attempts.ListAttempts(
		ctx,
		actor,
		attempts.ListInput{
			DatabaseID: request.Params.DatabaseId,
			Since:      request.Params.Since,
			Limit:      limit,
			After:      request.Params.After,
		},
	)
	if err != nil {
		return nil, err
	}

	items := make([]HealthcheckAttempt, len(page.Items))
	for index, attempt := range page.Items {
		items[index] = HealthcheckAttempt{
			Id:        attempt.ID,
			Status:    new(DatabaseHealthStatus(attempt.Status)),
			CreatedAt: attempt.CreatedAt.UTC(),
		}
	}

	return ListHealthcheckAttempts200JSONResponse{Items: items, NextCursor: page.NextCursor}, nil
}
