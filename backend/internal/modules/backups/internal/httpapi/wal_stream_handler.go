package httpapi

import (
	"context"

	"databasus-backend/internal/modules/backups/internal/physical"
	"databasus-backend/internal/modules/users"
)

func (h *Handler) GetWalStream(
	ctx context.Context,
	request GetWalStreamRequestObject,
) (GetWalStreamResponseObject, error) {
	actor, err := users.RequireActor(ctx)
	if err != nil {
		return nil, err
	}

	live, err := h.physicalBackups.GetWalStreamLive(ctx, actor, request.DatabaseId)
	if err != nil {
		return nil, err
	}

	body, err := toWalStreamLive(*live)
	if err != nil {
		return nil, err
	}

	return GetWalStream200JSONResponse(body), nil
}

func (h *Handler) GetWalStreamStatistics(
	ctx context.Context,
	request GetWalStreamStatisticsRequestObject,
) (GetWalStreamStatisticsResponseObject, error) {
	actor, err := users.RequireActor(ctx)
	if err != nil {
		return nil, err
	}

	statistics, err := h.physicalBackups.GetWalStreamStatistics(
		ctx,
		actor,
		request.DatabaseId,
		&physical.GetWalStreamStatisticsRequest{Period: physical.WalStreamStatisticsPeriod(request.Params.Period)},
	)
	if err != nil {
		return nil, err
	}

	body, err := toWalStreamStatistics(*statistics)
	if err != nil {
		return nil, err
	}

	return GetWalStreamStatistics200JSONResponse(body), nil
}
