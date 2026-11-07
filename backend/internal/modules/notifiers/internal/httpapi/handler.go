package httpapi

import (
	"databasus-backend/internal/modules/notifiers/internal/notifier"
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
