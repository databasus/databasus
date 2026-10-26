package httpapi

import (
	"context"

	"databasus-backend/internal/modules/notifiers/internal/notifier"
	"databasus-backend/internal/modules/users"
)

var _ StrictServerInterface = (*Handler)(nil)

type Deps struct {
	Notifiers *notifier.Service
	Transfers *notifier.TransferService
	TestSends *notifier.TestSendService
}

type Handler struct {
	notifiers *notifier.Service
	transfers *notifier.TransferService
	testSends *notifier.TestSendService
}

func NewHandler(deps Deps) *Handler {
	return &Handler{notifiers: deps.Notifiers, transfers: deps.Transfers, testSends: deps.TestSends}
}

func getActor(ctx context.Context) (users.Actor, error) {
	actor, isFound := users.GetActor(ctx)
	if !isFound {
		return users.Actor{}, errNoActor
	}

	return actor, nil
}
